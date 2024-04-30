package api

import (
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"net/http"
	"strconv"
	"test/internal/goblockapi"
)

type PaginatedRef struct {
	Count    int              `json:"count"`
	Next     string           `json:"next"`
	Previous string           `json:"previous"`
	Results  []goblockapi.Ref `json:"results"`
}

// CreateRef Creates referral accruals
// TODO: Optimize DB read/write use, use Redis for most of it
func CreateRef(config *goblockapi.AppConfig, tx *gorm.DB, user goblockapi.User, dimp float64, dact float64) {
	fmt.Println(dimp)
	fmt.Println(config)
	if user.Upline > 0 {
		var referrerLevelOne goblockapi.User
		res := tx.
			Clauses(clause.Locking{Strength: "UPDATE"}).
			Where(
				"id = ?",
				user.Upline,
			).First(&referrerLevelOne)
		if res.RowsAffected == 1 {
			// Referral Transactions level 1
			refTransactionOne := goblockapi.Ref{
				UserId:   referrerLevelOne.Id,
				AuthorId: user.Id,
				Lvl:      1,
			}
			tx.FirstOrInit(&refTransactionOne)
			userEmail := ""
			if user.Email != "" {
				userEmail = user.Email
			} else if user.TwitterEmail != "" {
				userEmail = user.TwitterEmail
			} else if user.DiscordEmail != "" {
				userEmail = user.DiscordEmail
			} else if user.GoogleEmail != "" {
				userEmail = user.GoogleEmail
			}
			refTransactionOne.AuthorUpline = user.RefCounter
			refTransactionOne.AuthorAddress = user.Address
			refTransactionOne.AuthorEmail = userEmail
			refTransactionOne.GoogleName = user.GoogleName
			dimpRewardOne := config.Settings.Ref.LvlOne * dimp
			dactRewardOne := config.Settings.Ref.LvlOne * dact
			refTransactionOne.Dact += dactRewardOne
			refTransactionOne.Dimp += dimpRewardOne
			res = tx.Save(&refTransactionOne)
			fmt.Println("[Ref tx] created. Lvl 1. DIMP", dimpRewardOne)
			if res.Error == nil {
				referrerLevelOne.DimpRewards += dimpRewardOne
				referrerLevelOne.DimpEarned += dimpRewardOne
				referrerLevelOne.DactEarned += dactRewardOne
				res = tx.Save(&referrerLevelOne)
				if res.Error == nil {

					if referrerLevelOne.Upline > 0 {
						var referrerLevelTwo goblockapi.User
						res = tx.
							Clauses(clause.Locking{Strength: "UPDATE"}).
							Where(
								"id = ?",
								referrerLevelOne.Upline,
							).First(&referrerLevelTwo)
						if res.RowsAffected == 1 {
							// Referral Transactions level 2
							refTransactionTwo := goblockapi.Ref{
								UserId:   referrerLevelTwo.Id,
								AuthorId: user.Id,
								Lvl:      2,
							}
							tx.FirstOrInit(&refTransactionTwo)
							refTransactionTwo.AuthorUpline = user.RefCounter
							refTransactionTwo.AuthorAddress = user.Address
							refTransactionTwo.AuthorEmail = userEmail
							refTransactionTwo.GoogleName = user.GoogleName
							dimpRewardTwo := config.Settings.Ref.LvlTwo * dimp
							dactRewardTwo := config.Settings.Ref.LvlTwo * dact
							refTransactionTwo.Dact += dactRewardTwo
							refTransactionTwo.Dimp += dimpRewardTwo
							res = tx.Save(&refTransactionTwo)
							fmt.Println("[Ref tx] created. Lvl 2. DIMP", refTransactionTwo.Dimp)
							if res.Error == nil {
								referrerLevelTwo.DimpRewards += dimpRewardTwo
								referrerLevelTwo.DimpEarned += dimpRewardTwo
								referrerLevelTwo.DactEarned += dactRewardTwo
								res = tx.Save(&referrerLevelTwo)
								if res.Error == nil {

									if referrerLevelTwo.Upline > 0 {
										var referrerLevelThree goblockapi.User
										res = tx.
											Clauses(clause.Locking{Strength: "UPDATE"}).
											Where(
												"id = ?",
												referrerLevelTwo.Upline,
											).First(&referrerLevelThree)
										if res.RowsAffected == 1 {
											// Referral Transactions level 2
											refTransactionThree := goblockapi.Ref{
												UserId:   referrerLevelThree.Id,
												AuthorId: user.Id,
												Lvl:      3,
											}
											tx.FirstOrInit(&refTransactionThree)
											refTransactionThree.AuthorUpline = user.RefCounter
											refTransactionThree.AuthorAddress = user.Address
											refTransactionThree.AuthorEmail = userEmail
											refTransactionThree.GoogleName = user.GoogleName
											dimpRewardThree := config.Settings.Ref.LvlThree * dimp
											dactRewardThree := config.Settings.Ref.LvlThree * dact
											refTransactionThree.Dact += dactRewardThree
											refTransactionThree.Dimp += dimpRewardThree
											res = tx.Save(&refTransactionThree)
											fmt.Println("[Ref tx] created. Lvl 3. DIMP", refTransactionThree.Dimp)
											if res.Error == nil {
												referrerLevelThree.DimpRewards += dimpRewardThree
												referrerLevelThree.DimpEarned += dimpRewardThree
												referrerLevelThree.DactEarned += dactRewardThree
												_ = tx.Save(&referrerLevelThree)
											}
										}
									}
								}
							}
						}
					}
				}
			}
		}
	}
}

// CreateRefEmpty Creates Referral relations
func CreateRefEmpty(tx *gorm.DB, user goblockapi.User, upline uint) {
	var referrerLevelOne goblockapi.User
	res := tx.
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where(
			"id = ?",
			upline,
		).First(&referrerLevelOne)
	if res.RowsAffected == 1 {
		// Referral Relation level 1
		refTransactionOne := goblockapi.Ref{
			UserId:   referrerLevelOne.Id,
			AuthorId: user.Id,
			Lvl:      1,
		}
		tx.FirstOrInit(&refTransactionOne)
		userEmail := ""
		if user.Email != "" {
			userEmail = user.Email
		} else if user.TwitterEmail != "" {
			userEmail = user.TwitterEmail
		} else if user.DiscordEmail != "" {
			userEmail = user.DiscordEmail
		} else if user.GoogleEmail != "" {
			userEmail = user.GoogleEmail
		}
		refTransactionOne.AuthorUpline = user.RefCounter
		refTransactionOne.AuthorAddress = user.Address
		refTransactionOne.AuthorEmail = userEmail
		refTransactionOne.GoogleName = user.GoogleName
		res = tx.Save(&refTransactionOne)
		fmt.Println("[Ref tx] Relation Created. Lvl 1")
		if res.Error == nil {
			if referrerLevelOne.Upline > 0 {
				var referrerLevelTwo goblockapi.User
				res = tx.
					Clauses(clause.Locking{Strength: "UPDATE"}).
					Where(
						"id = ?",
						referrerLevelOne.Upline,
					).First(&referrerLevelTwo)
				if res.RowsAffected == 1 {
					// Referral Relation level 2
					refTransactionTwo := goblockapi.Ref{
						UserId:   referrerLevelTwo.Id,
						AuthorId: user.Id,
						Lvl:      2,
					}
					tx.FirstOrInit(&refTransactionTwo)
					refTransactionTwo.AuthorUpline = user.RefCounter
					refTransactionTwo.AuthorAddress = user.Address
					refTransactionTwo.AuthorEmail = userEmail
					refTransactionTwo.GoogleName = user.GoogleName
					res = tx.Save(&refTransactionTwo)
					fmt.Println("[Ref tx] Relation Created. Lvl 2")
					if res.Error == nil {
						if referrerLevelTwo.Upline > 0 {
							var referrerLevelThree goblockapi.User
							res = tx.
								Clauses(clause.Locking{Strength: "UPDATE"}).
								Where(
									"id = ?",
									referrerLevelTwo.Upline,
								).First(&referrerLevelThree)
							if res.RowsAffected == 1 {
								// Referral Transactions level 2
								refTransactionThree := goblockapi.Ref{
									UserId:   referrerLevelThree.Id,
									AuthorId: user.Id,
									Lvl:      3,
								}
								tx.FirstOrInit(&refTransactionThree)
								refTransactionThree.AuthorUpline = user.RefCounter
								refTransactionThree.AuthorAddress = user.Address
								refTransactionThree.AuthorEmail = userEmail
								refTransactionThree.GoogleName = user.GoogleName
								res = tx.Save(&refTransactionThree)
								fmt.Println("[Ref tx] Relation Created. Lvl 3")
							}
						}
					}
				}
			}
		}
	}
}

// GetReferrals godoc
// @Summary Get
// @Description get the status of server.
// @Tags root
// @Accept */*
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router / [get]
func GetReferrals(c *gin.Context) {
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
	userId := uint(0)
	var user goblockapi.User
	res := app.Db.Where(
		"address = ? AND google_id = ?",
		address,
		googleId,
	).First(&user)
	if res.RowsAffected == 1 {
		userId = user.Id
		address = user.Address
	} else {
		c.JSON(http.StatusForbidden, gin.H{"error": errors.New("invalid jwt").Error()})
		return
	}
	// TODO: Check Redis cache first, then update the cached feed if needed
	var referrals []goblockapi.Ref
	app.Db.Where("user_id = ?", userId).
		Order("created_at DESC"). // Ensure ordering is applied here
		Find(&referrals)
	paginatedTx := paginateRef(referrals, page, size)
	c.JSON(http.StatusOK, paginatedTx)
}

func paginateRef(referrals []goblockapi.Ref, page int, size int) (paginatedRef PaginatedRef) {
	paginatedRef.Results = []goblockapi.Ref{}
	feedLen := len(referrals)
	i := (page - 1) * size
	if feedLen <= i {
		return paginatedRef
	}
	if feedLen > page*size {
		paginatedRef.Next = fmt.Sprintf("/users/ref/?page=%d&size=%d", page+1, size)
	}
	if page > 1 {
		paginatedRef.Previous = fmt.Sprintf("/users/ref/?page=%d&size=%d", page-1, size)
	}
	if size > feedLen {
		size = feedLen
	}
	k := i + size
	j := k
	fmt.Println("ref length: ", feedLen)
	if feedLen < page*size {
		j = feedLen
	}
	paginatedRef.Count = len(referrals)
	if k > feedLen {
		k = feedLen
	}
	paginatedRef.Results = referrals[i:j:k]
	return paginatedRef
}
