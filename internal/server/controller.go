package server

import (
	"context"
	"encoding/json"
	"fmt"
	ratelimit "github.com/JGLTechnologies/gin-rate-limit"
	"github.com/redis/go-redis/v9"
	"log"
	"math/rand"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gorilla/websocket"
	"test/internal/api"
	"test/internal/api/middleware"
	"test/internal/goblockapi"

	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"
)

var App *goblockapi.App
var AppTrack *goblockapi.AppTrack
var AppScrap *goblockapi.AppScrap
var AppTx *goblockapi.AppTx

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func keyFunc(c *gin.Context) string {
	return c.ClientIP()
}

func errorHandler(c *gin.Context, info ratelimit.Info) {
	c.String(429, "Too many requests. Try again in "+time.Until(info.ResetTime).String())
}

func ApiInit() { // Run Api Server
	// @title Dev Backend
	// @version 0.1
	// @description Dev Backend: REST API & WebSocket Server
	// @contact.email support@dev-team.com
	// @host localhost:8000
	// @BasePath /
	// @schemes http https ws wss
	App = goblockapi.Init()
	router := gin.Default()
	router.RedirectTrailingSlash = false
	router.RedirectFixedPath = false
	// This makes it so each ip can only make 100 requests per second
	store := ratelimit.RedisStore(&ratelimit.RedisOptions{
		RedisClient: redis.NewClient(&redis.Options{
			Addr:     os.Getenv("REDIS_ADDR"),
			Password: os.Getenv("REDIS_PASSWORD"),
			DB:       1,
		}),
		Rate:  time.Second,
		Limit: 100,
	})
	mw := ratelimit.RateLimiter(store, &ratelimit.Options{
		ErrorHandler: errorHandler,
		KeyFunc:      keyFunc,
	})
	router.Use(cors.New(cors.Config{
		AllowOrigins: []string{
			"http://0.0.0.0:3000",
			"http://localhost:3000",
		},
		AllowHeaders:  []string{"Origin", "Access-Control-Allow-Origin", "Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With"},
		ExposeHeaders: []string{"Content-Length"},
		AllowMethods:  []string{"GET, POST, OPTIONS, PUT, DELETE"},
		MaxAge:        24 * time.Hour,
	}))
	router.Use(func(c *gin.Context) {
		c.Set("app", App)
	})
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	router.GET("/ws", mw, wsHandler)
	router.GET("/ws/", mw, wsHandler)
	core := router.Group("/core/")
	{
		core.GET("/gasPrice", mw, api.GetGasPrice)
		core.GET("/gasPrice/", mw, api.GetGasPrice)
		core.GET("/balance/:address", mw, api.GetBalance)
		core.GET("/balance/:address/", mw, api.GetBalance)
	}
	auth := router.Group("/auth/")
	{
		auth.GET("/nonce/:address", mw, api.Nonce)
		auth.GET("/nonce/:address/", mw, api.Nonce)
		auth.POST("/signin", mw, api.Signin)
		auth.POST("/signin/", mw, api.Signin)
		auth.POST("/oauth", mw, api.Oauth)
		auth.POST("/oauth/", mw, api.Oauth)
	}
	users := router.Group("/users/").Use(middleware.Auth())
	{
		users.GET("/me", mw, api.GetUser)
		users.GET("/me/", mw, api.GetUser)
		users.GET("/tx", mw, api.GetTransactionsList)
		users.GET("/tx/", mw, api.GetTransactionsList)
		users.GET("/ref", mw, api.GetReferrals)
		users.GET("/ref/", mw, api.GetReferrals)
	}
	tx := router.Group("/tx/").Use(middleware.Auth())
	{
		tx.POST("/withdraw", mw, api.Withdraw)
		tx.POST("/withdraw/", mw, api.Withdraw)
		tx.POST("/sync", mw, api.SyncRequest)
		tx.POST("/sync/", mw, api.SyncRequest)
	}
	fmt.Println("[ Dev Backend is up and listening to :8000 ]")
	if err := router.Run(":8000"); err != nil {
		log.Fatal("Failed to run Dev Backend on :8000: ", err)
	}
}

func TxInit() { // Run Transaction Tracker
	AppTx = goblockapi.InitTx()
	go TxReplenishmentHandle(AppTx)
	go TxWithdrawHandle(AppTx)
	for {
		time.Sleep(10 * time.Minute)
	}
}

func wsHandler(c *gin.Context) {
	// Extract token from query
	token := c.DefaultQuery("token", "")
	user := goblockapi.User{}
	if token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
		return
	}
	address, googleId, err := api.GetUserFromToken(token)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
		return
	}
	// Upgrade Connection
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("Failed to set websocket upgrade: %+v", err)
		return
	}
	defer conn.Close()
	// Find User
	app := c.MustGet("app").(*goblockapi.App)
	appConfigRaw, _ := app.Rdb.Get(c, "app_config").Result()
	if len(appConfigRaw) > 0 {
		_ = json.Unmarshal([]byte(appConfigRaw), &goblockapi.CurrentAppConfig)
	}
	// Set a pong handler to update the connection's last pong time
	lastPong := time.Now()
	conn.SetPongHandler(func(string) error {
		lastPong = time.Now()
		return nil
	})
	pingPeriod := 3 * time.Second
	timeout := 9 * time.Second
	var mu sync.Mutex // Mutex to synchronize writes to the WebSocket connection
	res := app.Db.Where(
		"address = ? AND google_id = ?",
		address,
		googleId,
	).First(&user)
	if res.RowsAffected == 1 {
		jsonData := goblockapi.SyncUserStats(app.Rdb, app.Db, user)
		if jsonData != nil {
			// Send the serialized JSON data over the WebSocket
			if err := conn.WriteMessage(websocket.TextMessage, jsonData); err != nil {
				fmt.Println("Socket: Failed to send data:", err)
				return
			}
		}
		if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
			fmt.Println("Socket: Failed to send ping:", err)
			_ = conn.Close()
			return
		}
		go func() {
			pubsub := app.Rdb.Subscribe(c, fmt.Sprintf("notification_ch@%d", user.Id))
			defer pubsub.Close()

			ch := pubsub.Channel()
			for msg := range ch {
				var msgDecoded goblockapi.WsResponseData
				err = json.Unmarshal([]byte(msg.Payload), &msgDecoded)
				if err == nil {
					res := app.Rdb.Set(context.Background(), fmt.Sprintf("notification_cache@%d:%d", user.Id, msgDecoded.Data.Id), msg.Payload, 1*time.Hour)
					fmt.Println("Rdb.Set", res)
				}
				mu.Lock()
				if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
					log.Println("Socket: Failed to send ping:", err)
					mu.Unlock()
					_ = conn.Close()
					return
				}
				mu.Unlock()
			}
		}()
	} else {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
		return
	}
	// Start listening for commands via ws
	go func() {
		defer conn.Close()

		for {
			messageType, p, err := conn.ReadMessage()
			if err != nil {
				log.Println(err)
				return
			}
			// Handle the received message
			switch messageType {
			case websocket.TextMessage:
				message := string(p)
				// Check if the message is an acknowledgment
				var ackMsg struct {
					Type string `json:"type"`
					Id   int    `json:"id"`
				}
				if err := json.Unmarshal([]byte(message), &ackMsg); err == nil {
					if ackMsg.Type == "ack" {
						// Remove the acknowledged message from Redis
						_, err := app.Rdb.Del(context.Background(), fmt.Sprintf("notification_cache@%d:%d", user.Id, ackMsg.Id)).Result()
						if err != nil {
							fmt.Println("failed to delete acknowledged message from Redis: %v", err)
						}
						fmt.Println("ACK RECEIVED", ackMsg)
						continue // Skip further processing since it's an ack message
					}
				}
				if message == "sync" {
					_ = app.Db.Where(
						"address = ? AND google_id = ?",
						address,
						googleId,
					).First(&user)
					jsonData := goblockapi.SyncUserStats(app.Rdb, app.Db, user)
					if jsonData != nil {
						// Send the serialized JSON data over the WebSocket
						mu.Lock()
						if err := conn.WriteMessage(websocket.TextMessage, jsonData); err != nil {
							fmt.Println("Socket: Failed to send data:", err)
							mu.Unlock()
							return
						}
						mu.Unlock()
					}
				}
				// Sends mockup notifications
				if message == api.MessageTypeQuestCompletedDefault {
					fmt.Println(message)
					data := api.WsResponseData{
						Target: api.MessageTargetNotification,
						Config: *goblockapi.CurrentAppConfig,
						User: goblockapi.UserData{
							ID:         user.Id,
							Balance:    user.DimpBuffer,
							Rewards:    user.DimpRewards,
							Dact:       user.DactEarned,
							DimpEarned: user.DimpEarned,
							DimpSpent:  user.DimpSpent,
							Address:    user.Address,
							Hash:       user.Hash,
							RefUrl:     user.RefUrl,
							Actions:    user.Actions,
						},
						Data: api.NotificationData{
							Id:      rand.Intn(99999),
							Style:   api.MessageStyleSuccess,
							Type:    api.MessageTypeQuestCompletedDefault,
							Message: "This a VERIFIED Default notification mockup. This field contains AI comment on the quality of contribution, or the reason why the contribution has been rejected. Up to 200 symbols of text.",
							Dimp:    17.5,
							Rating:  0.64,
						},
					}
					jsonData, err := json.Marshal(data)
					if err != nil {
						log.Println("Socket: Failed to serialize data:", err)
						return
					}
					mu.Lock()
					if err := conn.WriteMessage(websocket.TextMessage, jsonData); err != nil {
						log.Println("Socket: Failed to send data:", err)
						mu.Unlock()
						return
					}
					mu.Unlock()
				}
				if message == api.MessageTypeQuestRejectedDefault {
					data := api.WsResponseData{
						Target: api.MessageTargetNotification,
						Config: *goblockapi.CurrentAppConfig,
						User: goblockapi.UserData{
							ID:         user.Id,
							Balance:    user.DimpBuffer,
							Rewards:    user.DimpRewards,
							Dact:       user.DactEarned,
							DimpEarned: user.DimpEarned,
							DimpSpent:  user.DimpSpent,
							Address:    user.Address,
							Hash:       user.Hash,
							RefUrl:     user.RefUrl,
							Actions:    user.Actions,
						},
						Data: api.NotificationData{
							Id:      rand.Intn(99999),
							Style:   api.MessageStyleError,
							Type:    api.MessageTypeQuestRejectedDefault,
							Message: "This a REJECTED Default notification mockup. This field contains AI comment on the quality of contribution, or the reason why the contribution has been rejected. Up to 200 symbols of text.",
						},
					}
					jsonData, err := json.Marshal(data)
					if err != nil {
						log.Println("Socket: Failed to serialize data:", err)
						return
					}
					mu.Lock()
					if err := conn.WriteMessage(websocket.TextMessage, jsonData); err != nil {
						log.Println("Socket: Failed to send data:", err)
						mu.Unlock()
						return
					}
					mu.Unlock()
				}
			default:
				fmt.Println("Socket: Unhandled message type:", messageType)
			}
		}
	}()
	for {
		// We process all the cached notifications
		iter := app.Rdb.Scan(context.Background(), 0, fmt.Sprintf("notification_cache@%d:*", user.Id), 0).Iterator()
		for iter.Next(context.Background()) {
			lastNotification, _ := app.Rdb.Get(context.Background(), iter.Val()).Result()
			if len(lastNotification) > 0 {
				mu.Lock()
				if err := conn.WriteMessage(websocket.TextMessage, []byte(lastNotification)); err != nil {
					log.Println("Socket: Failed to send data:", err)
					mu.Unlock()
					_ = conn.Close()
					return
				}
				mu.Unlock()
			}
		}
		//if err := iter.Err(); err != nil {
		//	continue
		//}
		if time.Since(lastPong) > timeout {
			log.Println("Socket: Client did not respond to ping, closing connection")
			//return
		}
		mu.Lock()
		if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
			log.Println("Socket: Failed to send ping:", err)
			mu.Unlock()
			return
		}
		mu.Unlock()
		time.Sleep(pingPeriod)
	}
}
