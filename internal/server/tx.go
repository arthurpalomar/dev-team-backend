package server

import (
	"context"
	"fmt"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"gorm.io/gorm/clause"
	"math/big"
	"os"
	"strings"
	"test/internal/goblockapi"
	"time"
)

func getCurrentBlockNumber(client *ethclient.Client) (uint64, error) {
	header, err := client.HeaderByNumber(context.Background(), nil)
	if err != nil {
		return 0, err
	}
	return header.Number.Uint64(), nil
}

func subscribeToContractEventsDeposit(AppTx *goblockapi.AppTx, web3Conn *ethclient.Client) {
	fromBlock, err := getCurrentBlockNumber(web3Conn)
	if err != nil {
		fmt.Println("Block number error")
		fmt.Println(err.Error())
		return
	}
	fromBlock -= 20
	addr := common.HexToAddress(os.Getenv("DIMP_EXCHANGE_CONTRACT_ADDRESS"))
	for {
		query := ethereum.FilterQuery{
			Addresses: []common.Address{addr},
			FromBlock: new(big.Int).SetUint64(fromBlock),
		}
		logs, err := web3Conn.FilterLogs(context.Background(), query)
		if err != nil {
			fmt.Println("Logs reading error")
			fmt.Println(err.Error())
			time.Sleep(30 * time.Second)
			continue
		}
		abiString := `[{"inputs":[{"internalType":"contract IERC20","name":"DIMP_","type":"address"}],"stateMutability":"nonpayable","type":"constructor"},{"inputs":[{"internalType":"address","name":"target","type":"address"}],"name":"AddressEmptyCode","type":"error"},{"inputs":[{"internalType":"address","name":"account","type":"address"}],"name":"AddressInsufficientBalance","type":"error"},{"inputs":[],"name":"FailedInnerCall","type":"error"},{"inputs":[{"internalType":"address","name":"owner","type":"address"}],"name":"OwnableInvalidOwner","type":"error"},{"inputs":[{"internalType":"address","name":"account","type":"address"}],"name":"OwnableUnauthorizedAccount","type":"error"},{"inputs":[],"name":"ReentrancyGuardReentrantCall","type":"error"},{"inputs":[{"internalType":"address","name":"token","type":"address"}],"name":"SafeERC20FailedOperation","type":"error"},{"anonymous":false,"inputs":[{"indexed":true,"internalType":"address","name":"depositer","type":"address"},{"indexed":false,"internalType":"uint256","name":"depositAmount","type":"uint256"}],"name":"Deposit","type":"event"},{"anonymous":false,"inputs":[{"indexed":true,"internalType":"address","name":"previousOwner","type":"address"},{"indexed":true,"internalType":"address","name":"newOwner","type":"address"}],"name":"OwnershipTransferred","type":"event"},{"anonymous":false,"inputs":[{"indexed":true,"internalType":"address","name":"withdrawer","type":"address"},{"indexed":false,"internalType":"uint256","name":"withdrawAmount","type":"uint256"}],"name":"Withdraw","type":"event"},{"inputs":[],"name":"DIMP","outputs":[{"internalType":"contract IERC20","name":"","type":"address"}],"stateMutability":"view","type":"function"},{"inputs":[{"internalType":"address","name":"newBlocked","type":"address"}],"name":"addBlacklist","outputs":[],"stateMutability":"nonpayable","type":"function"},{"inputs":[{"internalType":"address","name":"user","type":"address"}],"name":"blacklisted","outputs":[{"internalType":"bool","name":"","type":"bool"}],"stateMutability":"view","type":"function"},{"inputs":[{"internalType":"address","name":"account","type":"address"}],"name":"internalBalance","outputs":[{"internalType":"uint256","name":"","type":"uint256"}],"stateMutability":"view","type":"function"},{"inputs":[{"internalType":"uint256","name":"value","type":"uint256"}],"name":"internalExchange","outputs":[],"stateMutability":"nonpayable","type":"function"},{"inputs":[],"name":"minimumExchangeAmount","outputs":[{"internalType":"uint256","name":"","type":"uint256"}],"stateMutability":"view","type":"function"},{"inputs":[],"name":"owner","outputs":[{"internalType":"address","name":"","type":"address"}],"stateMutability":"view","type":"function"},{"inputs":[{"internalType":"address","name":"newUnblocked","type":"address"}],"name":"removeBlacklist","outputs":[],"stateMutability":"nonpayable","type":"function"},{"inputs":[],"name":"renounceOwnership","outputs":[],"stateMutability":"nonpayable","type":"function"},{"inputs":[{"internalType":"address","name":"user","type":"address"},{"internalType":"uint256","name":"amount","type":"uint256"}],"name":"setInternalBalance","outputs":[],"stateMutability":"nonpayable","type":"function"},{"inputs":[{"internalType":"uint256","name":"newMinimumExchangeAmount","type":"uint256"}],"name":"setMinimumExchangeAmount","outputs":[],"stateMutability":"nonpayable","type":"function"},{"inputs":[{"internalType":"address","name":"newOwner","type":"address"}],"name":"transferOwnership","outputs":[],"stateMutability":"nonpayable","type":"function"},{"inputs":[{"internalType":"uint256","name":"value","type":"uint256"}],"name":"withdraw","outputs":[],"stateMutability":"nonpayable","type":"function"}]`
		contractAbi, err := abi.JSON(strings.NewReader(abiString))
		if err != nil {
			fmt.Println("ABI reader error")
			fmt.Println(err.Error())
			time.Sleep(30 * time.Second)
			continue
		}
		for _, vLog := range logs {
			event := struct {
				DepositAmount *big.Int
			}{}
			err = contractAbi.UnpackIntoInterface(&event, "Deposit", vLog.Data)
			if err != nil {
				fmt.Println("ABI reader error")
				fmt.Println(err.Error())

				time.Sleep(30 * time.Second)
				continue
			}
			if len(vLog.Topics) != 2 {
				fmt.Println("Unexpected number of topics in log")
				time.Sleep(10 * time.Second)
				continue
			}
			eventSignature := vLog.Topics[0].Hex() // Get the event signature from the first topic
			author := common.HexToAddress(vLog.Topics[1].Hex())
			if eventSignature == os.Getenv("DIMP_DEPOSIT_SIGNATURE") {
				res := handleDepositEvent(AppTx, vLog, author, event.DepositAmount)
				if res {
					fromBlock = vLog.BlockNumber
				}
			}
		}
		time.Sleep(5 * time.Second)
	}
}

func handleDepositEvent(AppTx *goblockapi.AppTx, log types.Log, address common.Address, amount *big.Int) (result bool) {
	result = true
	var txDouble goblockapi.Transaction
	res := AppTx.Db.Where(
		"txid = ?",
		log.TxHash.Hex(),
	).First(&txDouble)
	if res.RowsAffected > 0 {
		result = true
	} else {
		amountFloat := new(big.Float).SetInt(amount)
		amountRate := new(big.Float).SetFloat64(0.000001)
		amountDB := amountFloat.Mul(amountFloat, amountRate)
		amountTx, _ := amountDB.Float64()
		fmt.Println("Amount from chain:", amount, "Amount to DB:", amountTx)
		if amountTx <= 0 {
			result = false
			return
		}
		var user goblockapi.User
		res = AppTx.Db.
			Clauses(clause.Locking{Strength: "UPDATE"}).
			Where(
				"address <> '' AND address IS NOT NULL AND address = ?",
				address.Hex(),
			).First(&user)
		if res.RowsAffected == 1 {
			isSyncing, _ := AppTx.Rdb.Get(context.Background(), fmt.Sprintf(`is_syncing_%v`, user.Id)).Result()
			if len(isSyncing) > 0 {
				result = false
				fmt.Printf("[[Tx Withdraw]] Skipped because syncing Rewards for user: %v\n", user.Id)
				return
			}
			tx := AppTx.Db.Begin()
			defer func() {
				tx.Rollback()
			}()
			transaction := goblockapi.Transaction{
				Txid:     log.TxHash.Hex(),
				UserId:   user.Id,
				AuthorId: user.Id,
				Type:     "in",
				Address:  address.Hex(),
				Status:   1, // Status [0:New, 1:Confirmed, 9:Rejected]
				Amount:   amountTx,
				Token:    os.Getenv("DIMP_CONTRACT_ADDRESS"),
			}
			res = tx.Create(&transaction)
			if res.Error != nil {
				result = false
				return
			}
			// TODO: Add Telegram notification
			user.DimpBuffer = user.DimpBuffer + transaction.Amount
			res = tx.Save(&user)
			if res.Error != nil {
				result = false
				return
			}
			tx.Commit()
			fmt.Printf("[[Tx Deposit]] Platform User balance is set to: %v\n", user.DimpBuffer)
			cpUrl := os.Getenv("CP_URL")
			msg := fmt.Sprintf(
				`DEPOSITED TO PLATFORM [Transaction: %s](%s%s)
[User: %d](%s/users/%d)
Amount: %s
User Downline: %v
User Actions: %v
User Balance: %s`,
				transaction.Txid,
				goblockapi.EscapeMarkdownV2(fmt.Sprintf("%s", `https://polygonscan.com/tx/`)),
				transaction.Txid,
				user.Id,
				cpUrl,
				user.Id,
				goblockapi.EscapeMarkdownV2(fmt.Sprintf("%f", transaction.Amount)),
				user.RefCounter,
				user.Actions,
				goblockapi.EscapeMarkdownV2(fmt.Sprintf("%f", user.DimpBuffer)),
			)
			fmt.Println(goblockapi.SendTelegramMessage(msg, "finance"))
			jsonData := goblockapi.SyncUserStats(AppTx.Rdb, AppTx.Db, user)
			if jsonData != nil {
				AppTx.Rdb.Publish(context.Background(), fmt.Sprintf("notification_ch@%d", user.Id), jsonData).Err()
			}
		}
	}
	return result
}

func subscribeToContractEventsWithdraw(AppTx *goblockapi.AppTx, web3Conn *ethclient.Client) (result bool) {
	fromBlock, err := getCurrentBlockNumber(web3Conn)
	if err != nil {
		fmt.Println("Block number error")
		fmt.Println(err.Error())
		return
	}
	fromBlock -= 20
	addr := common.HexToAddress(os.Getenv("DIMP_EXCHANGE_CONTRACT_ADDRESS"))
	for {
		query := ethereum.FilterQuery{
			Addresses: []common.Address{addr},
			FromBlock: new(big.Int).SetUint64(fromBlock),
		}
		logs, err := web3Conn.FilterLogs(context.Background(), query)
		if err != nil {
			fmt.Println("Logs reading error")
			fmt.Println(err.Error())
			time.Sleep(30 * time.Second)
			continue
		}
		abiString := `[{"inputs":[{"internalType":"contract IERC20","name":"DIMP_","type":"address"}],"stateMutability":"nonpayable","type":"constructor"},{"inputs":[{"internalType":"address","name":"target","type":"address"}],"name":"AddressEmptyCode","type":"error"},{"inputs":[{"internalType":"address","name":"account","type":"address"}],"name":"AddressInsufficientBalance","type":"error"},{"inputs":[],"name":"FailedInnerCall","type":"error"},{"inputs":[{"internalType":"address","name":"owner","type":"address"}],"name":"OwnableInvalidOwner","type":"error"},{"inputs":[{"internalType":"address","name":"account","type":"address"}],"name":"OwnableUnauthorizedAccount","type":"error"},{"inputs":[],"name":"ReentrancyGuardReentrantCall","type":"error"},{"inputs":[{"internalType":"address","name":"token","type":"address"}],"name":"SafeERC20FailedOperation","type":"error"},{"anonymous":false,"inputs":[{"indexed":true,"internalType":"address","name":"depositer","type":"address"},{"indexed":false,"internalType":"uint256","name":"depositAmount","type":"uint256"}],"name":"Deposit","type":"event"},{"anonymous":false,"inputs":[{"indexed":true,"internalType":"address","name":"previousOwner","type":"address"},{"indexed":true,"internalType":"address","name":"newOwner","type":"address"}],"name":"OwnershipTransferred","type":"event"},{"anonymous":false,"inputs":[{"indexed":true,"internalType":"address","name":"withdrawer","type":"address"},{"indexed":false,"internalType":"uint256","name":"withdrawAmount","type":"uint256"}],"name":"Withdraw","type":"event"},{"inputs":[],"name":"DIMP","outputs":[{"internalType":"contract IERC20","name":"","type":"address"}],"stateMutability":"view","type":"function"},{"inputs":[{"internalType":"address","name":"newBlocked","type":"address"}],"name":"addBlacklist","outputs":[],"stateMutability":"nonpayable","type":"function"},{"inputs":[{"internalType":"address","name":"user","type":"address"}],"name":"blacklisted","outputs":[{"internalType":"bool","name":"","type":"bool"}],"stateMutability":"view","type":"function"},{"inputs":[{"internalType":"address","name":"account","type":"address"}],"name":"internalBalance","outputs":[{"internalType":"uint256","name":"","type":"uint256"}],"stateMutability":"view","type":"function"},{"inputs":[{"internalType":"uint256","name":"value","type":"uint256"}],"name":"internalExchange","outputs":[],"stateMutability":"nonpayable","type":"function"},{"inputs":[],"name":"minimumExchangeAmount","outputs":[{"internalType":"uint256","name":"","type":"uint256"}],"stateMutability":"view","type":"function"},{"inputs":[],"name":"owner","outputs":[{"internalType":"address","name":"","type":"address"}],"stateMutability":"view","type":"function"},{"inputs":[{"internalType":"address","name":"newUnblocked","type":"address"}],"name":"removeBlacklist","outputs":[],"stateMutability":"nonpayable","type":"function"},{"inputs":[],"name":"renounceOwnership","outputs":[],"stateMutability":"nonpayable","type":"function"},{"inputs":[{"internalType":"address","name":"user","type":"address"},{"internalType":"uint256","name":"amount","type":"uint256"}],"name":"setInternalBalance","outputs":[],"stateMutability":"nonpayable","type":"function"},{"inputs":[{"internalType":"uint256","name":"newMinimumExchangeAmount","type":"uint256"}],"name":"setMinimumExchangeAmount","outputs":[],"stateMutability":"nonpayable","type":"function"},{"inputs":[{"internalType":"address","name":"newOwner","type":"address"}],"name":"transferOwnership","outputs":[],"stateMutability":"nonpayable","type":"function"},{"inputs":[{"internalType":"uint256","name":"value","type":"uint256"}],"name":"withdraw","outputs":[],"stateMutability":"nonpayable","type":"function"}]`
		contractAbi, err := abi.JSON(strings.NewReader(abiString))
		if err != nil {
			fmt.Println("ABI reader error")
			fmt.Println(err.Error())
			time.Sleep(30 * time.Second)
			continue
		}
		for _, vLog := range logs {
			event := struct {
				WithdrawAmount *big.Int
			}{}
			err = contractAbi.UnpackIntoInterface(&event, "Withdraw", vLog.Data)
			if err != nil {
				fmt.Println("ABI reader error")
				fmt.Println(err.Error())
				time.Sleep(30 * time.Second)
				continue
			}
			if len(vLog.Topics) != 2 {
				fmt.Println("Unexpected number of topics in log")
				time.Sleep(10 * time.Second)
				continue
			}
			eventSignature := vLog.Topics[0].Hex() // Get the event signature from the first topic
			author := common.HexToAddress(vLog.Topics[1].Hex())
			if eventSignature == os.Getenv("DIMP_WITHDRAW_SIGNATURE") {
				res := handleWithdrawEvent(AppTx, vLog, author, event.WithdrawAmount)
				if res {
					fromBlock = vLog.BlockNumber
				}
			}
		}
		time.Sleep(5 * time.Second)
	}
}

func handleWithdrawEvent(AppTx *goblockapi.AppTx, log types.Log, address common.Address, amount *big.Int) (result bool) {
	result = true
	var txDouble goblockapi.Transaction
	res := AppTx.Db.Where(
		"txid = ?",
		log.TxHash.Hex(),
	).First(&txDouble)
	if res.RowsAffected > 0 {
		result = true
	} else {
		amountFloat := new(big.Float).SetInt(amount)
		amountRate := new(big.Float).SetFloat64(0.000001)
		amountDB := amountFloat.Mul(amountFloat, amountRate)
		amountTx, _ := amountDB.Float64()
		fmt.Println("Amount from chain:", amount, "Amount to DB:", amountTx)
		if amountTx <= 0 {
			result = false
			return
		}
		var user goblockapi.User
		res = AppTx.Db.
			Clauses(clause.Locking{Strength: "UPDATE"}).
			Where(
				"address <> '' AND address IS NOT NULL AND address = ?",
				address.Hex(),
			).First(&user)
		if res.RowsAffected == 1 {
			isSyncing, _ := AppTx.Rdb.Get(context.Background(), fmt.Sprintf(`is_syncing_%v`, user.Id)).Result()
			if len(isSyncing) > 0 {
				result = false
				fmt.Printf("[[Tx Withdraw]] Skipped because syncing Rewards for user: %v\n", user.Id)
				return
			}
			if amountTx > user.DimpBuffer {
				result = false
				return
			}
			tx := AppTx.Db.Begin()
			defer func() {
				tx.Rollback()
			}()
			transaction := goblockapi.Transaction{
				Txid:     log.TxHash.Hex(),
				UserId:   user.Id,
				AuthorId: user.Id,
				Type:     "out",
				Address:  address.Hex(),
				Status:   1, // Status [0:New, 1:Confirmed, 9:Rejected]
				Amount:   amountTx,
				Token:    os.Getenv("DIMP_CONTRACT_ADDRESS"),
			}
			res = tx.Create(&transaction)
			if res.Error != nil {
				result = false
				return
			}
			// TODO: Add Telegram notification
			user.DimpBuffer = user.DimpBuffer - transaction.Amount
			res = tx.Save(&user)
			if res.Error != nil {
				result = false
				return
			}
			tx.Commit()
			fmt.Printf("[[Tx Withdraw]] Platform User balance is set to: %v\n", user.DimpBuffer)
			cpUrl := os.Getenv("CP_URL")
			msg := fmt.Sprintf(
				`WITHDRAWN FROM PLATFORM [Transaction: %s](%s%s)
[User: %d](%s/users/%d)
Amount: %s
User Downline: %v
User Actions: %v
User Balance: %s`,
				transaction.Txid,
				goblockapi.EscapeMarkdownV2(fmt.Sprintf("%s", `https://polygonscan.com/tx/`)),
				transaction.Txid,
				user.Id,
				cpUrl,
				user.Id,
				goblockapi.EscapeMarkdownV2(fmt.Sprintf("%f", transaction.Amount)),
				user.RefCounter,
				user.Actions,
				goblockapi.EscapeMarkdownV2(fmt.Sprintf("%f", user.DimpBuffer)),
			)
			fmt.Println(goblockapi.SendTelegramMessage(msg, "finance"))
			jsonData := goblockapi.SyncUserStats(AppTx.Rdb, AppTx.Db, user)
			if jsonData != nil {
				AppTx.Rdb.Publish(context.Background(), fmt.Sprintf("notification_ch@%d", user.Id), jsonData).Err()
			}
		}
	}
	return result
}

func TxReplenishmentHandle(AppTx *goblockapi.AppTx) {
	ethereumNodeURL := os.Getenv("INFURA_WSS")
	web3Conn, err := ethclient.Dial(ethereumNodeURL)
	if err != nil {
		fmt.Println(err.Error())
	}
	defer web3Conn.Close()
	fmt.Println("[[Tx Deposit]] Waiting for events...")
	subscribeToContractEventsDeposit(AppTx, web3Conn)
}

func TxWithdrawHandle(AppTx *goblockapi.AppTx) {
	ethereumNodeURL := os.Getenv("INFURA_WSS")
	web3Conn, err := ethclient.Dial(ethereumNodeURL)
	if err != nil {
		fmt.Println(err.Error())
	}
	defer web3Conn.Close()
	fmt.Println("[[Tx Withdraw]] Waiting for events...")
	subscribeToContractEventsWithdraw(AppTx, web3Conn)
}
