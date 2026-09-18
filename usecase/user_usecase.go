package usecase

import (
	"context"
	"crypto/md5"
	"errors"
	"fmt"
	"time"

	"github.com/lamboktulus1379/issuing-ledger-service/domain/dto"
	"github.com/lamboktulus1379/issuing-ledger-service/domain/model"
	"github.com/lamboktulus1379/issuing-ledger-service/domain/repository"
	"github.com/lamboktulus1379/issuing-ledger-service/infrastructure/configuration"
	"github.com/lamboktulus1379/issuing-ledger-service/infrastructure/logger"
	"github.com/lamboktulus1379/issuing-ledger-service/infrastructure/utils"

	mssql "github.com/microsoft/go-mssqldb"
)

type IUserUsecase interface {
	Login(ctx context.Context, req model.ReqLogin) dto.ResLogin
	Register(ctx context.Context, req model.ReqRegister) dto.ResRegister
}

type UserUsecase struct {
	userRepository repository.IUser
}

func NewUserUsecase(userRepository repository.IUser) IUserUsecase {
	return &UserUsecase{userRepository: userRepository}
}

func (userUsecase *UserUsecase) Login(ctx context.Context, req model.ReqLogin) dto.ResLogin {
	var res dto.ResLogin

	user, err := userUsecase.userRepository.GetByUserName(ctx, req.UserName)
	if err != nil {
		logger.GetLogger().WithField("error", err).Error("Error while Getting username")
		res.ResponseCode = "401"
		res.ResponseMessage = "Unautorized."
		return res
	}

	md5Req := fmt.Sprintf("%x", md5.Sum([]byte(req.Password)))

	if md5Req != user.Password {
		logger.GetLogger().WithField("request_password", md5Req).Error("Password not matching")
		res.ResponseCode = "401"
		res.ResponseMessage = "Unautorized."
		return res
	}

	secretKey := configuration.C.App.SecretKey

	// Create the Claims
	// Extend token validity to 24 hours (was 5 minutes)
	expiration := time.Now().Add(24 * time.Hour)

	// Build JWT claims. Use standard registered claim name "iss" for issuer so middleware can extract user_id.
	claims := make(map[string]interface{})
	claims["user_name"] = user.UserName
	claims["exp"] = expiration.Unix()
	// Use issuer as the numeric user ID string
	claims["iss"] = fmt.Sprint(user.ID)

	accessToken, err := utils.GenerateToken(claims, secretKey)
	if err != nil {
		logger.GetLogger().WithField("error", err).Info("Error while Signed string")
		res.ResponseCode = "401"
		res.ResponseMessage = "Unautorized"
		return res
	}
	res.ResponseCode = "200"
	res.ResponseMessage = "Success"
	res.Data.AccessToken = accessToken
	res.Data.ExpiresAt = expiration.Unix()

	return res
}

func (userUcase *UserUsecase) Register(ctx context.Context, req model.ReqRegister) dto.ResRegister {
	var res dto.ResRegister

	// Hash the password with MD5 before storing
	hashedPassword := fmt.Sprintf("%x", md5.Sum([]byte(req.Password)))

	reqUser := model.User{
		Name:     req.Name,
		UserName: req.UserName,
		Password: hashedPassword, // Store the hashed password
	}

	err := userUcase.userRepository.CreateUser(ctx, reqUser)
	if err != nil {
		// Map SQL Server unique constraint violations to 409 Conflict
		var sqlErr mssql.Error
		if errors.As(err, &sqlErr) {
			switch sqlErr.Number {
			case 2627, 2601: // Unique constraint or duplicate key
				res.Data = nil
				res.ResponseCode = "409"
				res.ResponseMessage = "Username already exists"
				return res
			}
		}
		res.Data = nil
		res.ResponseCode = "500"
		res.ResponseMessage = "Internal server error"
		return res
	}

	userDto := dto.UserDto{
		Name:     req.Name,
		UserName: req.UserName,
	}
	res.Data = userDto
	res.ResponseCode = "200"
	res.ResponseMessage = "Success"

	return res
}
