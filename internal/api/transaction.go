package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
	"os"
	"strconv"
	"test/internal/goblockapi"
)

type txParams struct {
	Txid string `json:"txid"` // Blockchain transaction id
	//Type    string  `json:"type"` // 'in' for Wallet to Platform or 'out' for Platform to Wallet
	Address string  `json:"address"`
	Amount  float64 `json:"amount"` // Amount
	Token   string  `json:"token"`  // Token address
	Message string  `json:"message"`
}

type PaginatedTx struct {
	Count    int           `json:"count"`
	Next     string        `json:"next"`
	Previous string        `json:"previous"`
	Results  []interface{} `json:"results"`
}

// GetTransactionsList godoc
// @Summary Get
// @Description get the status of server.
// @Tags root
// @Accept */*
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router / [get]
func GetTransactionsList(c *gin.Context) {
	app := c.MustGet("app").(*goblockapi.App)
	page, err := strconv.Atoi(c.DefaultQuery("page", "1"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid page"})
		return
	}
	if page < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid page"})
		return
	}
	size, err := strconv.Atoi(c.DefaultQuery("size", "20"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if size < 1 || size > 100 {
		c.JSON(http.StatusBadRequest, gin.H{"error": errors.New("maximum size is 100").Error()})
		return
	}
	token := c.GetHeader("Authorization")
	if token == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": errors.New("jwt missing").Error()})
		return
	}
	address, googleId, err := GetUserFromToken(token)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var user goblockapi.User
	res := app.Db.Where(
		"address = ? AND google_id = ?",
		address,
		googleId,
	).First(&user)
	if res.RowsAffected == 1 {
	} else {
		c.JSON(http.StatusForbidden, gin.H{"error": errors.New("malformed data").Error()})
		return
	}
	// TODO: Check Redis cache first, then update the cached feed if needed
	var transactions []goblockapi.Transaction
	var txes []goblockapi.Tx
	app.Db.Where("address = ?", user.Address).
		Order("created_at DESC").
		Find(&transactions)
	app.Db.Where("user_id = ?", user.Id).
		Order("created_at DESC").
		Find(&txes)
	mixedTx := []interface{}{}
	if len(transactions) > 0 {
		bufferTx := txes
		for _, transaction := range transactions {
			timeTransaction := transaction.CreatedAt
			var newBufferTx []goblockapi.Tx
			for i, tx := range bufferTx {
				timeTx := tx.CreatedAt
				if timeTx.After(timeTransaction) {
					mixedTx = append(mixedTx, tx)
				} else {
					newBufferTx = append(newBufferTx, bufferTx[i])
				}
			}
			mixedTx = append(mixedTx, transaction)
			bufferTx = newBufferTx
		}
		if len(bufferTx) > 0 {
			for _, tx := range bufferTx {
				mixedTx = append(mixedTx, tx)
			}
		}
	} else {
		bufferTx := transactions
		for _, tx := range txes {
			timeTransaction := tx.CreatedAt
			var newBufferTx []goblockapi.Transaction
			for i, transaction := range bufferTx {
				timeTx := transaction.CreatedAt
				if timeTx.After(timeTransaction) {
					mixedTx = append(mixedTx, transaction)
				} else {
					newBufferTx = append(newBufferTx, bufferTx[i])
				}
			}
			mixedTx = append(mixedTx, tx)
			bufferTx = newBufferTx
		}
		if len(bufferTx) > 0 {
			for _, transaction := range bufferTx {
				mixedTx = append(mixedTx, transaction)
			}
		}
	}
	paginatedTx := paginateTx(mixedTx, page, size)
	c.JSON(http.StatusOK, paginatedTx)
}

func paginateTx(transactions []interface{}, page int, size int) (paginatedTx PaginatedTx) {
	paginatedTx.Results = []interface{}{}
	feedLen := len(transactions)
	i := (page - 1) * size
	if feedLen <= i {
		return paginatedTx
	}
	if feedLen > page*size {
		paginatedTx.Next = fmt.Sprintf("/users/tx/?page=%d&size=%d", page+1, size)
	}
	if page > 1 {
		paginatedTx.Previous = fmt.Sprintf("/users/tx/?page=%d&size=%d", page-1, size)
	}
	if size > feedLen {
		size = feedLen
	}
	k := i + size
	j := k
	fmt.Println("tx length: ", feedLen)
	if feedLen < page*size {
		j = feedLen
	}
	paginatedTx.Count = len(transactions)
	if k > feedLen {
		k = feedLen
	}
	paginatedTx.Results = transactions[i:j:k]
	return paginatedTx
}

func SyncRequest(c *gin.Context) {
	fmt.Println("SYNC REQUESTED")
	app := c.MustGet("app").(*goblockapi.App)
	token := c.GetHeader("Authorization")
	if token == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": errors.New("jwt missing").Error()})
		return
	}
	address, googleId, err := GetUserFromToken(token)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var user goblockapi.User
	res := app.Db.
		Where(
			"address = ? AND google_id = ?",
			address,
			googleId,
		).First(&user)
	if res.RowsAffected == 1 {
	} else {
		c.JSON(http.StatusForbidden, gin.H{"error": errors.New("invalid jwt").Error()})
		return
	}
	var tParams txParams
	if err := c.ShouldBindJSON(&tParams); err != nil {
		c.JSON(http.StatusNotFound, err.Error())
		return
	}
	var txDouble goblockapi.Tx
	res = app.Db.
		Where(
			"type = ? AND status = ? AND user_id = ?",
			"y",
			1,
			user.Id,
		).First(&txDouble)
	if res.RowsAffected == 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": errors.New("wait until resolved").Error()})
		return
	}
	fmt.Println(txDouble)
	// We check limits here
	appConfigRaw, _ := app.Rdb.Get(c, "app_config").Result()
	if len(appConfigRaw) > 0 {
		_ = json.Unmarshal([]byte(appConfigRaw), &goblockapi.CurrentAppConfig)
	}
	wMinDimp := goblockapi.UsdToDimp(goblockapi.CurrentAppConfig.Settings.Limits.WithdrawMin, 2)
	if user.WithdrawMin > 0 {
		wMinDimp = goblockapi.UsdToDimp(user.WithdrawMin, 2)
	}
	if tParams.Amount < wMinDimp {
		c.JSON(http.StatusBadRequest, gin.H{"error": errors.New(`min_withdrawal`).Error()})
		return
	}
	var txNew goblockapi.Tx
	res = app.Db.
		Where(
			"type = ? AND user_id = ?",
			"y",
			user.Id,
		).First(&txNew)
	if res.RowsAffected == 1 {
		txNew.Status = 1
		res = app.Db.Save(&txNew)
		if res.Error != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": res.Error})
			return
		}
	} else {
		tx := app.Db.Begin()
		defer func() {
			tx.Rollback()
		}()
		txNew = goblockapi.Tx{
			UserId:   user.Id,
			AuthorId: user.Id,
			Address:  user.Address,
			Amount:   user.DimpRewards,
			Type:     "y",
			Status:   1, // 1 = Activity Check Requested
		}
		res = tx.Save(&txNew)
		if res.Error != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": res.Error})
			return
		}
		tx.Commit()
	}
	fmt.Println(txNew)
	cpUrl := os.Getenv("CP_URL")
	msg := fmt.Sprintf(
		`Approve Sync Request [Transaction: %d](%s/txes/%d)
[User: %d](%s/users/%d)
Actions: %v
Rewards: %s
Balance: %s`,
		txNew.Txid,
		cpUrl,
		txNew.Txid,
		user.Id,
		cpUrl,
		user.Id,
		user.Actions,
		goblockapi.EscapeMarkdownV2(fmt.Sprintf("%f", user.DimpRewards)),
		goblockapi.EscapeMarkdownV2(fmt.Sprintf("%f", user.DimpBuffer)),
	)
	err = goblockapi.SendTelegramMessage(msg, "finance")

	fmt.Println(err)
	// Reply with 200 = Allow sync
	c.JSON(http.StatusOK, gin.H{})
}

func Withdraw(c *gin.Context) {
	app := c.MustGet("app").(*goblockapi.App)
	token := c.GetHeader("Authorization")
	if token == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": errors.New("jwt missing").Error()})
		return
	}
	address, googleId, err := GetUserFromToken(token)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var user goblockapi.User
	res := app.Db.
		Where(
			"address = ? AND google_id = ?",
			address,
			googleId,
		).First(&user)
	if res.RowsAffected == 1 {
	} else {
		c.JSON(http.StatusForbidden, gin.H{"error": errors.New("invalid jwt").Error()})
		return
	}
	var tParams txParams
	if err := c.ShouldBindJSON(&tParams); err != nil {
		c.JSON(http.StatusNotFound, err.Error())
		return
	}
	// Check DIMP allowance
	if tParams.Amount > user.DimpBuffer {
		c.JSON(http.StatusBadRequest, errors.New(`insufficient_funds`).Error())
		return
	}

	// We check limits here
	appConfigRaw, _ := app.Rdb.Get(c, "app_config").Result()
	if len(appConfigRaw) > 0 {
		_ = json.Unmarshal([]byte(appConfigRaw), &goblockapi.CurrentAppConfig)
	}
	wMinDimp := goblockapi.UsdToDimp(goblockapi.CurrentAppConfig.Settings.Limits.WithdrawMin, 2)
	if user.WithdrawMin > 0 {
		wMinDimp = goblockapi.UsdToDimp(user.WithdrawMin, 2)
	}
	if tParams.Amount < wMinDimp {
		c.JSON(http.StatusBadRequest, errors.New(`min_withdrawal`).Error())
		return
	}
	wMaxDimp := goblockapi.UsdToDimp(goblockapi.CurrentAppConfig.Settings.Limits.WithdrawMax, 2)
	if user.WithdrawMax > 0 {
		wMaxDimp = goblockapi.UsdToDimp(user.WithdrawMax, 2)
	}
	if tParams.Amount > wMaxDimp {
		c.JSON(http.StatusBadRequest, errors.New(`max_withdrawal`).Error())
		return
	}
	// TODO: Create 0 tx to confirm and reject if there is one.
	// TODO: Always reserve balance from user wallet unless tx got cancelled.
	//tx := app.Db.Begin()
	//defer func() {
	//	tx.Rollback()
	//}()
	//transaction := goblockapi.Transaction{
	//	Txid:     log.TxHash.Hex(),
	//	UserId:   user.Id,
	//	AuthorId: user.Id,
	//	Type:     "out",
	//	Address:  address.Hex(),
	//	Status:   0, // Status [0:New, 1:Confirmed, 9:Rejected]
	//	Amount:   amountTx,
	//	Token:    os.Getenv("DIMP_CONTRACT_ADDRESS"),
	//}
	//res = tx.Create(&transaction)
	//if res.Error != nil {
	//	result = false
	//	return
	//}
	// Reply with 200 = Allow tx
	c.JSON(http.StatusOK, gin.H{
		"user": goblockapi.UserData{
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
	})
}
