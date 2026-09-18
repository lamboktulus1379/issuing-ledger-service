package utils

import (
	"time"

	"github.com/golang-jwt/jwt"
	"github.com/lamboktulus1379/issuing-ledger-service/infrastructure/logger"
)

func GetCurrentTime() time.Time {
	return time.Now().UTC()
}

func GenerateToken(payload map[string]interface{}, secretKey string) (string, error) {
	var claims jwt.MapClaims = payload
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(secretKey))
	if err != nil {
		logger.GetLogger().WithField("error", err).Error("Error while generate token")
		return "", err
	}
	return tokenString, nil
}
