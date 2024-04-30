package goblockapi

import (
	"context"
	"encoding/json"
	"github.com/hibiken/asynq"
	"os"
	"strconv"
	"test/internal/evm"

	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type App struct {
	Rpc *evm.Client
	Rdb *redis.Client
	Db  *gorm.DB
	Aqc *asynq.Client
	Aqi *asynq.Inspector
}

type AppConfig struct {
	Settings    AppSettings `json:"settings"`
	DimpUsdRate float64     `json:"dimp_usd_rate"`
}

type AppSettings struct {
	Ref    RefSettings  `json:"ref"`
	Prices SettingCost  `json:"prices"`
	Limits SettingLimit `json:"limits"`
}

type RefSettings struct {
	LvlOne   float64 `json:"lvl_one"`
	LvlTwo   float64 `json:"lvl_two"`
	LvlThree float64 `json:"lvl_three"`
}

type SettingCost struct {
	View     float64 `json:"view"`
	Follower float64 `json:"follower"`
	Retweet  float64 `json:"retweet"`
	Comment  float64 `json:"comment"`
	Repost   float64 `json:"repost"`
}

type SettingLimit struct {
	WithdrawMin     float64 `json:"withdraw_min"`
	WithdrawMax     float64 `json:"withdraw_max"`
	WithdrawMinDimp float64 `json:"withdraw_min_dimp"`
	WithdrawMaxDimp float64 `json:"withdraw_max_dimp"`
}

var (
	DefaultAppConfig *AppConfig
	CurrentAppConfig *AppConfig
)

func Init() *App {
	loadEnv()
	redisClient := setupRedis()
	db := setupDb()
	asynqClient := setupAsynqClient()
	asynqInspector := setupAsynqInspector()
	client := evm.New(os.Getenv("RPC_URL"))

	DefaultAppConfig = &AppConfig{
		Settings: AppSettings{
			Ref: RefSettings{
				LvlOne:   0.07,
				LvlTwo:   0.05,
				LvlThree: 0.03,
			},
			Prices: SettingCost{
				View:     0.005,
				Follower: 0.01,
				Retweet:  0.02,
				Comment:  0.15,
				Repost:   0.2,
			},
			Limits: SettingLimit{
				WithdrawMin:     1,
				WithdrawMinDimp: 1000,
				WithdrawMax:     100,
				WithdrawMaxDimp: 100000,
			},
		},
		DimpUsdRate: 0.001,
	}

	app := &App{
		Rpc: client,
		Rdb: redisClient,
		Db:  db,
		Aqc: asynqClient,
		Aqi: asynqInspector,
	}
	isSet := false
	appConfigRaw, _ := app.Rdb.Get(context.Background(), "app_config").Result()
	if len(appConfigRaw) > 0 {
		err := json.Unmarshal([]byte(appConfigRaw), &CurrentAppConfig)
		if err != nil {
		} else {
			isSet = true
		}
	}
	if !isSet {
		currentConfig, _ := json.Marshal(DefaultAppConfig)
		app.Rdb.Set(context.Background(), "app_config", currentConfig, 0)
	}
	return app
}

type AppTrack struct {
	Rpc *evm.Client
	Rdb *redis.Client
	Db  *gorm.DB
	Aqs *asynq.Server
}

type AppScrap struct {
	Rpc *evm.Client
	Rdb *redis.Client
	Db  *gorm.DB
	Aqs *asynq.Server
}

type AppTx struct {
	Rpc *evm.Client
	Rdb *redis.Client
	Db  *gorm.DB
}

func InitTrack() *AppTrack {
	loadEnv()
	redisClient := setupRedis()
	db := setupDb()
	asynqServer := setupAsynqServer("tracker")
	client := evm.New(os.Getenv("RPC_URL"))

	app := &AppTrack{
		Rpc: client,
		Rdb: redisClient,
		Db:  db,
		Aqs: asynqServer,
	}
	isSet := false
	appConfigRaw, _ := app.Rdb.Get(context.Background(), "app_config").Result()
	if len(appConfigRaw) > 0 {
		err := json.Unmarshal([]byte(appConfigRaw), &CurrentAppConfig)
		if err != nil {
		} else {
			isSet = true
		}
	}
	if !isSet {
		currentConfig, _ := json.Marshal(DefaultAppConfig)
		app.Rdb.Set(context.Background(), "app_config", currentConfig, 0)
	}
	return app
}

func InitScrap() *AppScrap {
	loadEnv()
	redisClient := setupRedis()
	db := setupDb()
	asynqServer := setupAsynqServer("scraper")
	client := evm.New(os.Getenv("RPC_URL"))

	app := &AppScrap{
		Rpc: client,
		Rdb: redisClient,
		Db:  db,
		Aqs: asynqServer,
	}
	return app
}

func InitTx() *AppTx {
	loadEnv()
	redisClient := setupRedis()
	db := setupDb()
	client := evm.New(os.Getenv("RPC_URL"))

	app := &AppTx{
		Rpc: client,
		Rdb: redisClient,
		Db:  db,
	}
	isSet := false
	appConfigRaw, _ := app.Rdb.Get(context.Background(), "app_config").Result()
	if len(appConfigRaw) > 0 {
		err := json.Unmarshal([]byte(appConfigRaw), &CurrentAppConfig)
		if err != nil {
		} else {
			isSet = true
		}
	}
	if !isSet {
		currentConfig, _ := json.Marshal(DefaultAppConfig)
		app.Rdb.Set(context.Background(), "app_config", currentConfig, 0)
	}
	return app
}

func setupRedis() *redis.Client {
	redisClient := redis.NewClient(&redis.Options{
		Addr:     os.Getenv("REDIS_ADDR"),
		Password: os.Getenv("REDIS_PASSWORD"),
		DB:       0,
	})
	return redisClient
}

func setupDb() *gorm.DB {
	dsn := os.Getenv("DB_DSN")
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		panic("failed to connect to the db")
	}
	err = db.AutoMigrate(
		&User{},
		&Action{},
		&Transaction{},
		&Tx{},
		&Ref{},
	)
	if err != nil {
		panic("failed ti run migrations")
	}

	return db
}

func setupAsynqClient() *asynq.Client {
	asynqClient := asynq.NewClient(asynq.RedisClientOpt{
		Addr:     os.Getenv("REDIS_ADDR"),
		Password: os.Getenv("REDIS_PASSWORD"),
	})
	return asynqClient
}

func setupAsynqInspector() *asynq.Inspector {
	asynqInspector := asynq.NewInspector(asynq.RedisClientOpt{
		Addr:     os.Getenv("REDIS_ADDR"),
		Password: os.Getenv("REDIS_PASSWORD"),
	})
	return asynqInspector
}

func setupAsynqServer(prefix string) *asynq.Server {
	switch prefix {
	case "tracker":
		concurency, err := strconv.Atoi(os.Getenv("TRACKER_PROXY_SCALE"))
		if err != nil {
			concurency = 10
		}
		asynqServer := asynq.NewServer(
			asynq.RedisClientOpt{
				Addr:     os.Getenv("REDIS_ADDR"),
				Password: os.Getenv("REDIS_PASSWORD"),
			},
			asynq.Config{
				Concurrency: concurency,
				Queues: map[string]int{
					"verify": 1,
				},
			},
		)
		return asynqServer
	case "scraper":
		concurency, err := strconv.Atoi(os.Getenv("SCRAPER_PROXY_SCALE"))
		if err != nil {
			concurency = 10
		}
		asynqServer := asynq.NewServer(
			asynq.RedisClientOpt{
				Addr:     os.Getenv("REDIS_ADDR"),
				Password: os.Getenv("REDIS_PASSWORD"),
			},
			asynq.Config{
				Concurrency: concurency,
				Queues: map[string]int{
					"posts":   2,
					"profile": 3,
				},
			},
		)
		return asynqServer
	}
	return nil
}

func loadEnv() {
	env := os.Getenv("APP_ENV")
	if "" == env {
		env = "development"
	}

	godotenv.Load(".env." + env + ".local")

	if "test" != env {
		godotenv.Load(".env.local")
	}
	godotenv.Load(".env." + env)
	godotenv.Load()
}
