package goblockapi

import (
	"gorm.io/gorm"
	_ "time/tzdata"
)

func GetRefStats(db *gorm.DB, user User) (refStats RefData) {
	var refRelations []Ref
	res := db.Where("user_id = ?", user.Id).Find(&refRelations)
	if res.RowsAffected > 0 {
		totalCounter, oneCounter, twoCounter, threeCounter := uint(0), uint(0), uint(0), uint(0)
		dimpTotal, dimpOne, dimpTwo, dimpThree := float64(0), float64(0), float64(0), float64(0)
		dactTotal, dactOne, dactTwo, dactThree := float64(0), float64(0), float64(0), float64(0)
		for _, relation := range refRelations {
			totalCounter++
			dimpTotal += relation.Dimp
			dactTotal += relation.Dact
			switch relation.Lvl {
			case 1:
				oneCounter++
				dimpOne += relation.Dimp
				dactOne += relation.Dact
			case 2:
				twoCounter++
				dimpTwo += relation.Dimp
				dactTwo += relation.Dact
			case 3:
				threeCounter++
				dimpThree += relation.Dimp
				dactThree += relation.Dact
			}
		}
		refStats.TotalCounter = totalCounter
		refStats.LlvOneCounter = oneCounter
		refStats.LlvTwoCounter = twoCounter
		refStats.LlvThreeCounter = threeCounter
		refStats.DimpTotal = dimpTotal
		refStats.DimpLvlOne = dimpOne
		refStats.DimpLvlTwo = dimpTwo
		refStats.DimpLvlThree = dimpThree
		refStats.DactTotal = dactTotal
		refStats.DactLvlOne = dactOne
		refStats.DactLvlTwo = dactTwo
		refStats.DactLvlThree = dactThree
	}
	return refStats
}
