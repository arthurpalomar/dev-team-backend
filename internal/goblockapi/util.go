package goblockapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/chenzhijie/go-web3"
	"github.com/chenzhijie/go-web3/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/hibiken/asynq"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"math"
	"math/big"
	"os"
	"reflect"
	"strconv"
	"strings"
	"test/internal/telegram"
	"time"
)

const (
	MessageTargetSync = "sync"
)

type WsResponseData struct {
	Target        string           `json:"target"` // Websocket message type: 'notify', 'alert', 'sync'
	User          UserData         `json:"user"`
	ReferralStats RefData          `json:"referral_stats"`
	Data          NotificationData `json:"data"`
	Config        AppConfig        `json:"app_config"`
}

type NotificationData struct {
	Id      int     `json:"id"`
	Style   string  `json:"style"`   // Target component style: 'success', 'warning', 'error', 'info'; mostly used with "type": "custom"
	Type    string  `json:"type"`    // Notification type: 'custom', 'quest_completed_default', 'quest_completed_follow', 'quest_completed_view', 'quest_completed_comment', 'quest_completed_quote'
	Message string  `json:"message"` // AI comment
	Url     string  `json:"url"`
	TaskId  uint    `json:"task_id"`
	Dimp    float64 `json:"dimp"`   // Reward, Transaction, etc. $DIMP amount
	Rating  float64 `json:"rating"` // [0;1] AI estimation of user contribution
}

func EscapeMarkdownV2(text string) string {
	specialChars := []string{"_", "*", "[", "]", "(", ")", "~", "`", ">", "#", "+", "-", "=", "|", "{", "}", ".", "!"}
	for _, char := range specialChars {
		text = strings.ReplaceAll(text, char, "\\"+char)
	}
	return text
}

func SendTelegramMessage(msg string, chat string) error {
	token := os.Getenv("TELEGRAM_TOKEN")
	if token == "" {
		err := errors.New("TELEGRAM_TOKEN is not set")
		return err
	}
	chatId := os.Getenv("DEFAULT_CHAT_ID")
	if chatId == "" {
		err := errors.New("DEFAULT CHAT_ID is not set")
		return err
	}
	switch chat {
	case "signup":
		chatId = os.Getenv("SIGNUP_CHAT_ID")
		if chatId == "" {
			err := errors.New("SIGNUP CHAT_ID is not set")
			return err
		}
	case "finance":
		chatId = os.Getenv("FINANCE_CHAT_ID")
		if chatId == "" {
			err := errors.New("FINANCE CHAT_ID is not set")
			return err
		}
	default:
		chatId = os.Getenv("DEFAULT_CHAT_ID")
		if chatId == "" {
			err := errors.New("DEFAULT CHAT_ID is not set")
			return err
		}
	}
	chatIdInt, err := strconv.Atoi(chatId)
	if err != nil {
		return err
	}
	id := int64(chatIdInt)
	bot, err := telegram.NewBot(token)
	if err != nil {
		return err
	}
	// Send a message
	_, err = bot.Api.SendMessage(id, msg, &gotgbot.SendMessageOpts{
		ParseMode: "MarkdownV2",
		LinkPreviewOptions: &gotgbot.LinkPreviewOptions{
			IsDisabled: true,
		},
	})
	if err != nil {
		return err
	}
	return nil
}

// InArray will search element inside array with any type.
// Will return boolean and index for matched element.
// True and index more than 0 if element is exist.
// needle is element to search, haystack is slice of value to be search.
func InArray(needle interface{}, haystack interface{}) (exists bool, index int) {
	exists = false
	index = -1

	switch reflect.TypeOf(haystack).Kind() {
	case reflect.Slice:
		s := reflect.ValueOf(haystack)

		for i := 0; i < s.Len(); i++ {
			if reflect.DeepEqual(needle, s.Index(i).Interface()) == true {
				index = i
				exists = true
				return
			}
		}
	}

	return
}

func Truncate(s string, size int) string {
	// Ensure size is within the bounds of the string length
	if size >= len(s) {
		return s
	}

	// Truncate the string by erasing characters from the beginning
	truncated := s[size:]
	return truncated
}

func WaitForAsynqTaskResult(ctx context.Context, i *asynq.Inspector, queue, taskID string) (*asynq.TaskInfo, error) {
	t := time.NewTicker(time.Second)
	defer t.Stop()
	for {
		select {
		case <-t.C:
			taskInfo, err := i.GetTaskInfo(queue, taskID)
			if err != nil {
				return nil, err
			}
			if taskInfo.CompletedAt.IsZero() {
				continue
			}
			return taskInfo, nil
		case <-ctx.Done():
			return nil, fmt.Errorf("context closed")
		}
	}
}

func UsdToDimp(usdPrice float64, precision uint) (dimpPrice float64) {
	dimpPrice = RoundFloat(usdPrice/CurrentAppConfig.DimpUsdRate, precision)
	return
}

func RoundFloat(val float64, precision uint) float64 {
	ratio := math.Pow(10, float64(precision))
	return math.Round(val*ratio) / ratio
}

func SyncUserStats(rdb *redis.Client, db *gorm.DB, user User) (jsonData []byte) { //
	// Send userData
	data := WsResponseData{
		Target: MessageTargetSync,
		Config: *CurrentAppConfig,
		User: UserData{
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
		ReferralStats: GetRefStats(db, user),
	}
	// Convert the struct to JSON
	var err error
	jsonData, err = json.Marshal(data)
	if err != nil {
		return
	}
	if len(user.Address) > 0 {
		syncAllowed := false
		var txSync Tx
		// Here we check if Rewards sync is approved and merge balances
		res := db.
			Where(
				"type = ? AND status = ? AND user_id = ?",
				"y",
				2, // 2 = Approved to sync
				user.Id,
			).First(&txSync)
		if res.RowsAffected == 1 {
			tx := db.Begin()
			defer func() {
				tx.Rollback()
			}()
			var userUpd User
			res = db.
				Clauses(clause.Locking{Strength: "UPDATE"}).
				Where(
					"id = ?",
					user.Id,
				).First(&userUpd)
			if res.RowsAffected < 1 {
				return
			}
			rewardsApproved := user.DimpRewards
			if txSync.Amount > 0 {
				if txSync.Amount < rewardsApproved {
					rewardsApproved = txSync.Amount
				}
			}
			userUpd.DimpBuffer += rewardsApproved
			userUpd.DimpRewards -= rewardsApproved
			txSync.Status = 0
			res = db.Save(&txSync)
			if res.Error != nil {
				return
			}
			res = db.Save(&userUpd)
			if res.Error != nil {
				return
			}
			tx.Commit()
			syncAllowed = true
		} else {

		}
		// Here we compare it with cached value
		isSet := false
		var balancePlatform float64
		balancePlatformCached, _ := rdb.Get(context.Background(), fmt.Sprintf(`balance_%v`, user.Id)).Result()
		if len(balancePlatformCached) > 0 {
			err := json.Unmarshal([]byte(balancePlatformCached), &balancePlatform)
			if err != nil {
			} else {
				isSet = true
			}
		}
		if !isSet {
			balancePlatform = user.DimpBuffer
			balancePlatformCacheNew, _ := json.Marshal(balancePlatform)
			rdb.Set(context.Background(), fmt.Sprintf(`balance_%v`, user.Id), balancePlatformCacheNew, 0*time.Second)
		}
		fmt.Println("[[SYNC]] User balance:", user.DimpBuffer, " | Cached value:", balancePlatform)
		// We update on-chain value only if User balance is not 0 and differs from cached value to more than 99 DIMP, both directions
		if user.DimpBuffer > 0 && (!isSet || user.DimpBuffer-balancePlatform > 99 || balancePlatform-user.DimpBuffer > 99) {
			//go func() {
			web3Conn, err := web3.NewWeb3(os.Getenv("RPC_PROVIDER_URL"))
			if err != nil {
				fmt.Println(err)
				return
			}
			blockNumber, err := web3Conn.Eth.GetBlockNumber()
			if err != nil {
				fmt.Println(err)
				return
			}
			fmt.Println("block ", blockNumber)
			web3Conn.Eth.SetChainId(137)
			err = web3Conn.Eth.SetAccount(os.Getenv("DIMP_ADMIN_KEY"))
			if err != nil {
				fmt.Println(err)
				return
			}
			nonce, err := web3Conn.Eth.GetNonce(web3Conn.Eth.Address(), nil)
			if err != nil {
				fmt.Println(err)
				return
			}
			abiString := `[{"inputs":[{"internalType":"address","name":"user","type":"address"},{"internalType":"uint256","name":"amount","type":"uint256"}],"name":"setInternalBalance","outputs":[],"stateMutability":"nonpayable","type":"function"},{"inputs":[{"internalType":"address","name":"account","type":"address"}],"name":"internalBalance","outputs":[{"internalType":"uint256","name":"","type":"uint256"}],"stateMutability":"view","type":"function"}]`
			contractAddr := os.Getenv("DIMP_EXCHANGE_CONTRACT_ADDRESS")
			contract, err := web3Conn.Eth.NewContract(abiString, contractAddr)
			if err != nil {
				fmt.Println(err.Error())
				return
			}
			// Convert DimpBuffer to uint256
			balance := big.NewInt(int64(user.DimpBuffer * 1000000))
			balanceOnChain, err := contract.Call("internalBalance", common.HexToAddress(user.Address))
			if err != nil {
				fmt.Println(err.Error())
			}
			fmt.Println("[[SYNC]] User balance:", balance, " | On-chain value:", balanceOnChain)
			// We skip transaction if there is no difference in balance
			if balance.String() == fmt.Sprintf("%d", balanceOnChain) {
				//fmt.Println("[[SYNC]] Skipping: DB balance has not been changed:", balanceOnChain, balance)
				return
			}
			intOnChain := new(big.Int)
			intOnChain, ok := intOnChain.SetString(fmt.Sprintf("%d", balanceOnChain), 10)
			if !ok {
				fmt.Println("[[SYNC]] SetString error. Balance:", fmt.Sprintf("%d", balanceOnChain))
				return
			}
			// Checks critical cases
			cmp := balance.Cmp(intOnChain)
			// Means that possible withdrawal tx has not been processed or someone is fooling us!
			if !syncAllowed {
				if cmp > 0 {
					// TODO: Fire TG Notification here!
					fmt.Println("!!! [[SYNC]] !!! WTF!?")
					return
				}
			}
			rdb.Set(context.Background(), fmt.Sprintf(`is_syncing_%v`, user.Id), "y", 60*time.Second)
			defer func() {
				rdb.Del(context.Background(), fmt.Sprintf(`is_syncing_%v`, user.Id))
			}()
			data, err := contract.EncodeABI("setInternalBalance", common.HexToAddress(user.Address), balance)
			if err != nil {
				fmt.Println(err.Error())
				return
			}
			call := &types.CallMsg{
				From: web3Conn.Eth.Address(),
				To:   common.HexToAddress(contractAddr),
				Data: data,
				Gas:  types.NewCallMsgBigInt(big.NewInt(types.MAX_GAS_LIMIT)),
			}
			// fmt.Printf("call %v\n", call)
			gasLimit, err := web3Conn.Eth.EstimateGas(call)
			if err != nil {
				fmt.Println(err.Error())
				return
			}
			gasPrice, err := web3Conn.Eth.SuggestGasTipCap()
			if err != nil {
				fmt.Println(err.Error())
				return
			}
			gasPriceBase, err := web3Conn.Eth.EstimateFee()
			if err != nil {
				fmt.Println(err.Error())
				return
			}
			gasPrice.Add(gasPriceBase.MaxPriorityFeePerGas, gasPriceBase.BaseFee)
			fmt.Printf("Estimate gas limit %v\n", gasLimit)
			txHash, err := web3Conn.Eth.SyncSendRawTransaction(
				common.HexToAddress(contractAddr),
				big.NewInt(0),
				nonce,
				gasLimit,
				gasPrice,
				data,
			)
			if err != nil {
				fmt.Println(err.Error())
				return
			}
			balancePlatformCacheNew, _ := json.Marshal(user.DimpBuffer)
			rdb.Set(context.Background(), fmt.Sprintf(`balance_%v`, user.Id), balancePlatformCacheNew, 0*time.Second)
			fmt.Println("[[SYNC]] On-chain User balance is set to:", balance, " | setInternalBalance tx hash:", txHash)
			//}()
		}
	}
	return
}
