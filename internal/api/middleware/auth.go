package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"test/internal/api/jwt"
)

func Auth() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.GetHeader("Authorization")
		if token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "jwt missing"})
			return
		}
		address, googleId, err := jwt.ValidateToken(token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}
		c.Set("address", address)
		c.Set("google_id", googleId)
		c.Next()
	}
}
