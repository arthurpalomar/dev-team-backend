package api

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/dchest/uniuri"
	"github.com/spruceid/siwe-go"
	"gorm.io/gorm/clause"
	"log"
	"math/rand"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/gin-gonic/gin"
	"test/internal/api/jwt"
	"test/internal/evm"
	"test/internal/goblockapi"
)

var ctx = context.Background()

type signinParams struct {
	Message   string `json:"message" binding:"required"`
	Signature string `json:"signature" binding:"required"`
	Hash      string `json:"fingerprint" binding:"required" validate:"required,max=50"`
	RefUrl    string `json:"invite_link" validate:"required,max=8"`
	Utm       string `json:"utm" validate:"max=500"`
	Ip        string `json:"ip" validate:"required,max=39"`
	Referer   string `json:"referer" validate:"max=150"`
	Locale    string `json:"locale" validate:"required,max=5"`
	GoogleId  string `json:"google_id" validate:"required,max=50"`
}

type oauthParams struct {
	Hash        string `json:"fingerprint" binding:"required" validate:"required,max=50"`
	RefUrl      string `json:"invite_link" validate:"required,max=8"`
	Utm         string `json:"utm" validate:"max=500"`
	Ip          string `json:"ip" validate:"required,max=39"`
	Referer     string `json:"referer" validate:"max=150"`
	Locale      string `json:"locale" validate:"required,max=5"`
	Address     string `json:"address"`
	GoogleId    string `json:"google_id" validate:"required,max=50"`
	GoogleName  string `json:"google_name" validate:"required,max=50"`
	GoogleEmail string `json:"google_email" validate:"required,max=50"`
}

type linkParams struct {
	Provider string `json:"provider" validate:"required,max=10"`
	Id       string `json:"id" validate:"required,max=100"`
	Name     string `json:"name" validate:"required,max=500"`
	Username string `json:"username" validate:"required,max=150"`
	Avatar   string `json:"avatar" validate:"max=350"`
	Email    string `json:"email" validate:"max=250"`
}

var digitCheck = regexp.MustCompile(`^[0-9]+$`)

// Nonce instead of storing the nonce in db for an inexistant user we just put it in some redis that expires
func Nonce(c *gin.Context) {
	app := c.MustGet("app").(*goblockapi.App)
	address := c.Param("address")

	if !evm.IsValidAddress(address) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid address format"})
		return
	}

	nonce := siwe.GenerateNonce()

	err := app.Rdb.Set(ctx, address, nonce, 1*time.Minute).Err()
	if err != nil {
		log.Fatal(err)
	}

	c.JSON(http.StatusOK, gin.H{
		"nonce": nonce,
	})
}

// Signin Sign in with SIWE
func Signin(c *gin.Context) {
	app := c.MustGet("app").(*goblockapi.App)
	var signinP signinParams
	if err := c.ShouldBindJSON(&signinP); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// parse message to siwe
	siweMessage, err := siwe.ParseMessage(signinP.Message)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// get the nonce in cache for address
	addr := siweMessage.GetAddress().String()
	nonce, err := app.Rdb.Get(ctx, addr).Result()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return

	}
	// domain will be cors restricted its fine to just use the one from the message
	domain := siweMessage.GetDomain()
	// verify signature
	publicKey, err := siweMessage.Verify(signinP.Signature, &domain, &nonce, nil)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	addr = crypto.PubkeyToAddress(*publicKey).Hex()
	// Signature is not valid
	if addr == "" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Go fuck yourself, you shitty bastard!"})
		return
	}
	fmt.Println("signature is valid")
	var user goblockapi.User
	res := app.Db.Where(""+
		"address NOT IN ('') AND address = ?",
		addr,
	).First(&user)
	// if user exists we update it
	if res.RowsAffected == 1 {
		fmt.Println("user found")
		if user.Hash == "" && signinP.Hash != "" {
			user.Hash = signinP.Hash
		}
		// If FE user has active Google session
		if user.GoogleId == "" && signinP.GoogleId != "" {
			// Link google_id to user if it is unique
			var userDoubleGoogle goblockapi.User
			res = app.Db.Where(""+
				"google_id NOT IN ('') AND google_id = ? AND id <> ?",
				signinP.GoogleId,
				user.Id,
			).First(&userDoubleGoogle)
			if res.RowsAffected == 0 {
				// TODO: Not secure at all !!!
				user.GoogleId = signinP.GoogleId
			} else {
				// Rejects if this user has another google_id linked
				c.JSON(http.StatusBadRequest, gin.H{"error": "log out with google"})
				return
			}
		}
		if user.RefUrl == "" {
			for {
				refNew := uniuri.NewLenChars(8, []byte("ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"))
				var double goblockapi.User
				res = app.Db.Where(""+
					"ref_url = ?",
					refNew,
				).First(&double)
				if res.RowsAffected == 1 {
					continue
				}
				user.RefUrl = refNew
				break
			}
		}
		app.Db.Save(&user)
		token, err := jwt.GenerateJWT(addr, user.GoogleId)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"user":      user,
			"is_signup": false,
			"jwt":       token,
		})
		return
	} else if signinP.GoogleId != "" {
		// if user with same google_id exists we update it
		var userGoogle goblockapi.User
		res = app.Db.Where(""+
			"(google_id NOT IN ('') AND google_id = ?)",
			signinP.GoogleId,
		).First(&userGoogle)
		if res.RowsAffected == 1 {
			// If User has no address set, we set it
			if userGoogle.Address == "" {
				regDimpBonus := float64(100)
				userGoogle.Address = addr
				if userGoogle.RefUrl == "" {
					for {
						refNew := uniuri.NewLenChars(8, []byte("ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"))
						var double goblockapi.User
						res = app.Db.Where(""+
							"ref_url = ?",
							refNew,
						).First(&double)
						if res.RowsAffected == 1 {
							continue
						}
						userGoogle.RefUrl = refNew
						break
					}
				}
				userGoogle.Group = 1
				userGoogle.DimpBuffer += regDimpBonus
				userGoogle.DimpEarned += regDimpBonus
				userGoogle.DactEarned += regDimpBonus * 50
				app.Db.Save(&userGoogle)
				txNew := goblockapi.Tx{
					UserId:   userGoogle.Id,
					AuthorId: userGoogle.Id,
					Address:  userGoogle.Address,
					Amount:   regDimpBonus,
					Type:     "b",
					Status:   1,
				}
				res = app.Db.Save(&txNew)
				if res.Error == nil {
					notification, _ := json.Marshal(WsResponseData{
						Target: MessageTargetNotification,
						User: goblockapi.UserData{
							ID:         userGoogle.Id,
							Balance:    userGoogle.DimpBuffer,
							Rewards:    userGoogle.DimpRewards,
							Dact:       userGoogle.DactEarned,
							DimpEarned: userGoogle.DimpEarned,
							DimpSpent:  userGoogle.DimpSpent,
							Address:    userGoogle.Address,
							Hash:       userGoogle.Hash,
							RefUrl:     userGoogle.RefUrl,
							Actions:    userGoogle.Actions,
						},
						Data: NotificationData{
							Id:      rand.Intn(99999),
							Style:   MessageStyleSuccess,
							Type:    MessageTypeQuestCompletedDefault,
							Message: "Well done! You are all set up now, here is another Bonus Reward from Aria. Complete Quests or share your Referral Link to get more rewards!",
							Dimp:    regDimpBonus,
							Rating:  float64(1),
						},
						Config: *goblockapi.CurrentAppConfig,
					})
					_ = app.Rdb.Publish(ctx, fmt.Sprintf("notification_ch@%d", userGoogle.Id), notification).Err()
				}
				cpUrl := os.Getenv("CP_URL")
				msg := fmt.Sprintf(
					`Web3 connected [User: %d](%s/users/%d)
[%s](https://polygonscan.com/address/%s)
Locale: %s
IP: [%s](%s%s)`,
					userGoogle.Id,
					cpUrl,
					userGoogle.Id,
					userGoogle.Address,
					userGoogle.Address,
					goblockapi.EscapeMarkdownV2(userGoogle.Locale),
					goblockapi.EscapeMarkdownV2(userGoogle.Ip),
					"https://iplocation.io/ip/",
					userGoogle.Ip,
				)
				if userGoogle.Upline > 0 {
					msg = fmt.Sprintf(
						`%s 
Invited by [User: %d](%s/users/%d)`,
						msg,
						userGoogle.Upline,
						cpUrl,
						userGoogle.Upline,
					)
				}
				if userGoogle.Referer != "" {
					msg = fmt.Sprintf(
						`%s 
[Referer URL](%s)`,
						msg,
						goblockapi.EscapeMarkdownV2(userGoogle.Referer),
					)
				}
				_ = goblockapi.SendTelegramMessage(msg, "signup")
				tokenNew, err := jwt.GenerateJWT(userGoogle.Address, userGoogle.GoogleId)
				if err != nil {
					c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}
				c.JSON(http.StatusOK, gin.H{
					"user":      userGoogle,
					"is_signup": false,
					"jwt":       tokenNew,
				})
				return
			} else {
				// Rejects if this user has another address linked
				c.JSON(http.StatusForbidden, gin.H{"error": "log in with google"})
				return
			}
		}
	}
	// No user to login into found
	c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	return
}

// Oauth Sign in with Social Network's oAuth
func Oauth(c *gin.Context) {
	app := c.MustGet("app").(*goblockapi.App)
	var signinP oauthParams
	if err := c.ShouldBindJSON(&signinP); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// empty parameters, Fingerprint might have been spoofed
	if signinP.GoogleId == "" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Go fuck yourself, you shitty bastard!"})
		return
	}
	// check if there is valid jwt provided
	token := c.GetHeader("Authorization")
	if token != "" {
		// if user exists we update it
		_, googleId, err := GetUserFromToken(token)
		if err == nil {
			var user goblockapi.User
			res := app.Db.
				Clauses(clause.Locking{Strength: "UPDATE"}).
				Where(
					"google_id = ?",
					googleId,
				).First(&user)
			if res.RowsAffected == 1 {
				// If FE user has active Google session
				if signinP.GoogleId != "" && signinP.GoogleId != googleId {
					// Rejects if this user has another google linked
					c.JSON(http.StatusForbidden, gin.H{"error": "log out with google"})
					return
				}
				if signinP.Address != "" && user.Address == "" {
					// Link address to user if it is unique
					var userDoubleGoogle goblockapi.User
					res = app.Db.Where(""+
						"address NOT IN ('') AND address = ? AND id <> ?",
						signinP.Address,
						user.Id,
					).First(&userDoubleGoogle)
					if res.RowsAffected == 0 {
						user.Address = signinP.Address
					} else {
						// Rejects if this user has another address linked
						c.JSON(http.StatusForbidden, gin.H{"error": "log out with web3"})
						return
					}
				}
				if user.RefUrl == "" {
					for {
						refNew := uniuri.NewLenChars(8, []byte("ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"))
						var double goblockapi.User
						res = app.Db.Where(""+
							"ref_url = ?",
							refNew,
						).First(&double)
						if res.RowsAffected == 1 {
							continue
						}
						user.RefUrl = refNew
						break
					}
				}
				app.Db.Save(&user)
				tokenNew, err := jwt.GenerateJWT(user.Address, user.GoogleId)
				if err != nil {
					c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}
				c.JSON(http.StatusOK, gin.H{
					"user":      user,
					"is_signup": false,
					"jwt":       tokenNew,
				})
				return
			}
		}
	} else {

	}
	// if user with same google_id exists we update it
	var user goblockapi.User
	res := app.Db.Where(""+
		"google_id NOT IN ('') AND google_id = ?",
		signinP.GoogleId,
	).First(&user)
	if res.RowsAffected == 1 {
		if user.RefUrl == "" {
			for {
				refNew := uniuri.NewLenChars(8, []byte("ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"))
				var double goblockapi.User
				res = app.Db.Where(""+
					"ref_url = ?",
					refNew,
				).First(&double)
				if res.RowsAffected == 1 {
					continue
				}
				user.RefUrl = refNew
				break
			}
		}
		app.Db.Save(&user)
		tokenNew, err := jwt.GenerateJWT(user.Address, user.GoogleId)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"user":      user,
			"is_signup": false,
			"jwt":       tokenNew,
		})
		return
	}
	// [[SIGN UP]]: If FE user has active Google session, but no account
	if signinP.GoogleId != "" {
		// Create a user with google_id if it is unique
		var userDoubleGoogle goblockapi.User
		res = app.Db.Where(""+
			"google_id NOT IN ('') AND google_id = ?",
			signinP.GoogleId,
		).First(&userDoubleGoogle)
		if res.RowsAffected == 0 {
			fmt.Println("[[New Sign Up]] Ref URL:", signinP.RefUrl)
			upline := uint(0)
			if len(signinP.RefUrl) > 0 {
				var referrer goblockapi.User
				res = app.Db.Where("ref_url = ?",
					signinP.RefUrl,
				).First(&referrer)
				if res.RowsAffected == 1 {
					upline = referrer.Id
					referrer.RefCounter++
					_ = app.Db.Save(&referrer)
				} else if digitCheck.MatchString(signinP.RefUrl) {
					if _, err := strconv.Atoi(signinP.RefUrl); err == nil {
						res = app.Db.Where("id = ?",
							signinP.RefUrl,
						).First(&referrer)
						if res.RowsAffected == 1 {
							upline = referrer.Id
							referrer.RefCounter++
							_ = app.Db.Save(&referrer)
						}
					}
				}
			}
			refNew := ""
			for {
				refNew = uniuri.NewLenChars(8, []byte("ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"))
				var double goblockapi.User
				res = app.Db.Where(""+
					"ref_url = ?",
					refNew,
				).First(&double)
				if res.RowsAffected == 1 {
					continue
				}
				break
			}
			regDimpBonus := float64(100)
			// if user not exist we create it
			user = goblockapi.User{
				Hash: signinP.Hash,
				Utm:  signinP.Utm,
				// TODO: use ip-api.com to set CountryCode based on IP
				Ip:          signinP.Ip,
				Locale:      signinP.Locale,
				Referer:     signinP.Referer,
				Group:       0, // 0 = WEB_2 LEAD
				DimpBuffer:  regDimpBonus,
				DimpEarned:  regDimpBonus,
				DactEarned:  regDimpBonus * 10,
				RefUrl:      refNew,
				Upline:      upline,
				GoogleId:    signinP.GoogleId,
				GoogleName:  signinP.GoogleName,
				GoogleEmail: signinP.GoogleEmail,
			}
			res = app.Db.Create(&user)
			if res.Error != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": res.Error})
				return

			}
			var userUpd goblockapi.User
			res = app.Db.Where(""+
				"google_id NOT IN ('') AND google_id = ?",
				user.GoogleId,
			).First(&userUpd)
			if res.RowsAffected == 1 {
				txNew := goblockapi.Tx{
					UserId:   userUpd.Id,
					AuthorId: userUpd.Id,
					Address:  userUpd.Address,
					Amount:   regDimpBonus,
					Type:     "b",
					Status:   1,
				}
				res = app.Db.Save(&txNew)
				if res.Error == nil {
					notification, _ := json.Marshal(WsResponseData{
						Target: MessageTargetNotification,
						User: goblockapi.UserData{
							ID:         userUpd.Id,
							Balance:    userUpd.DimpBuffer,
							Rewards:    userUpd.DimpRewards,
							Dact:       userUpd.DactEarned,
							DimpEarned: userUpd.DimpEarned,
							DimpSpent:  userUpd.DimpSpent,
							Address:    userUpd.Address,
							Hash:       userUpd.Hash,
							RefUrl:     userUpd.RefUrl,
							Actions:    userUpd.Actions,
						},
						Data: NotificationData{
							Id:      rand.Intn(99999),
							Style:   MessageStyleSuccess,
							Type:    MessageTypeQuestCompletedDefault,
							Message: "Welcome to Actocracy! Here is your Bonus Reward from Aria. Connect Web3 Wallet to get more valuable DIMP!",
							Dimp:    regDimpBonus,
							Rating:  float64(1),
						},
						Config: *goblockapi.CurrentAppConfig,
					})
					_ = app.Rdb.Publish(ctx, fmt.Sprintf("notification_ch@%d", userUpd.Id), notification).Err()
				}
			}
			if upline > 0 {
				fmt.Println("Creating Referral relations")
				CreateRefEmpty(app.Db, user, upline)
			}
			cpUrl := os.Getenv("CP_URL")
			msg := fmt.Sprintf(
				`New Signup [User: %d](%s/users/%d)
[%s](mailto:%s)
Locale: %s
IP: [%s](%s%s)`,
				user.Id,
				cpUrl,
				user.Id,
				user.Email,
				user.Email,
				goblockapi.EscapeMarkdownV2(user.Locale),
				goblockapi.EscapeMarkdownV2(user.Ip),
				"https://iplocation.io/ip/",
				user.Ip,
			)
			if user.Upline > 0 {
				msg = fmt.Sprintf(
					`%s 
Invited by [User: %d](%s/users/%d)`,
					msg,
					user.Upline,
					cpUrl,
					user.Upline,
				)
			}
			if user.Referer != "" {
				msg = fmt.Sprintf(
					`%s 
[Referer URL](%s)`,
					msg,
					goblockapi.EscapeMarkdownV2(user.Referer),
				)
			}
			_ = goblockapi.SendTelegramMessage(msg, "signup")
			tokenNew, err := jwt.GenerateJWT(user.Address, user.GoogleId)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{
				"user":      user,
				"is_signup": true,
				"jwt":       tokenNew,
			})
		} else {
			// Rejects if this user has another google_id linked
			c.JSON(http.StatusForbidden, gin.H{"error": "log out with google"})
			return
		}
	}
}
