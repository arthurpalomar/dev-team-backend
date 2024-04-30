package api

import (
	"errors"
	"github.com/gin-gonic/gin"
	"net/http"
	"test/internal/api/jwt"
	"test/internal/goblockapi"
)

func GetUser(c *gin.Context) {
	app := c.MustGet("app").(*goblockapi.App)
	address := c.MustGet("address")

	var user goblockapi.User
	res := app.Db.Where("address = ?", address).First(&user)
	if res.RowsAffected == 1 {
		c.JSON(http.StatusOK, user)
	} else {
		c.JSON(http.StatusNotFound, nil)
	}
}

func GetUserFromToken(token string) (address string, googleId string, err error) {
	address, googleId, err = jwt.ValidateToken(token)
	if err != nil {
		return "", "", errors.New("invalid jwt")
	}

	return address, googleId, nil
}
