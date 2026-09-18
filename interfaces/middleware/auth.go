package middleware

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/lamboktulus1379/issuing-ledger-service/domain/dto"
	"github.com/lamboktulus1379/issuing-ledger-service/domain/model"
	"github.com/lamboktulus1379/issuing-ledger-service/domain/repository"
	"github.com/lamboktulus1379/issuing-ledger-service/infrastructure/configuration"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
)

func Auth(userRepository repository.IUser) gin.HandlerFunc {
	var res dto.Res
	res.ResponseCode = "401"
	res.ResponseMessage = "Unauthorized"

	return func(ctx *gin.Context) {
		// Allow CORS preflight requests to pass without auth
		if ctx.Request.Method == http.MethodOptions {
			ctx.AbortWithStatus(http.StatusNoContent)
			return
		}
		authorization := ctx.Request.Header.Get("Authorization")
		// Support auth_token query param for SSE/EventSource where custom headers aren't possible
		if authorization == "" {
			if qt := ctx.Query("auth_token"); qt != "" {
				authorization = "Bearer " + qt
			}
		}
		secretKey := configuration.C.App.SecretKey
		if authorization == "" {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, res)
			return
		}
		auth := strings.Split(authorization, "Bearer ")
		if len(auth) != 2 {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, res)
			return
		}
		userClaims, token, err := getClaim(auth, secretKey)

		if token != nil && token.Valid {
			if !next(ctx, userRepository, userClaims) {
				ctx.AbortWithStatusJSON(http.StatusUnauthorized, res)
				return
			}
		} else {
			if abort(err, res) {
				ctx.AbortWithStatusJSON(http.StatusUnauthorized, res)
				return
			}
		}
	}
}

func abort(err error, res dto.Res) bool {
	var ve *jwt.ValidationError
	if errors.As(err, &ve) {
		if ve.Errors&jwt.ValidationErrorMalformed != 0 {
			res.ResponseMessage = "That's not even a token"
		} else if ve.Errors&(jwt.ValidationErrorExpired|jwt.ValidationErrorNotValidYet) != 0 {
			// Token is either expired or not active yet
			res.ResponseMessage = "Timing is everything"
		} else {
			res.ResponseMessage = fmt.Sprintf("Couldn't handle this token:%v", err)
		}
		return true
	}
	return false
}

func next(ctx *gin.Context, userRepository repository.IUser, userClaims model.UserClaims) bool {
	_, err := userRepository.GetByUserName(ctx.Request.Context(), userClaims.UserName)
	if err != nil {
		return false // Return false to indicate authorization failed
	}
	ctx.Set("user_id", userClaims.Issuer)
	ctx.Next()
	return true // Return true to indicate authorization succeeded
}

func getClaim(auth []string, secretKey string) (model.UserClaims, *jwt.Token, error) {
	var userClaims model.UserClaims
	token, err := jwt.ParseWithClaims(
		auth[1],
		&userClaims,
		func(token *jwt.Token) (interface{}, error) {
			return []byte(secretKey), nil
		},
	)
	return userClaims, token, err
}
