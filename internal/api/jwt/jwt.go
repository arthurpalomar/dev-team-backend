package jwt

import (
	"errors"
	"github.com/golang-jwt/jwt/v5"
	"os"
	"time"
)

type JWTClaim struct {
	Address  string `json:"address"`
	GoogleId string `json:"google_id"`
	jwt.RegisteredClaims
}

//const JWT_EXPIRATION = 24 * 7 * time.Hour

func GenerateJWT(address string, googleId string) (token string, err error) {

	var claims = JWTClaim{
		address,
		googleId,
		jwt.RegisteredClaims{
			//ExpiresAt: jwt.NewNumericDate(time.Now().Add(JWT_EXPIRATION)),
			IssuedAt: jwt.NewNumericDate(time.Now()),
		},
	}

	resToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	secret := os.Getenv("JWT_SECRET")
	signedToken, err := resToken.SignedString([]byte(secret))
	if err != nil {
		return "", err
	}
	return signedToken, nil
}

func ValidateToken(signedToken string) (address string, googleId string, err error) {
	token, err := jwt.ParseWithClaims(signedToken, &JWTClaim{}, func(t *jwt.Token) (interface{}, error) { return []byte(os.Getenv("JWT_SECRET")), nil })
	if err != nil {
		return "", "", err
	}
	claims, ok := token.Claims.(*JWTClaim)
	if !ok {
		return "", "", errors.New("error parsing claims")
	}
	if claims.Address == "" && claims.GoogleId == "" {
		return "", "", errors.New("malformed data")
	}
	//if claims.ExpiresAt.Unix() < time.Now().Local().Unix() {
	//	return "", errors.New("token expired")
	//}

	return claims.Address, claims.GoogleId, nil
}
