package http

import (
	"fmt"
	"net/http"

	"github.com/lamboktulus1379/issuing-ledger-service/domain/model"
	"github.com/lamboktulus1379/issuing-ledger-service/infrastructure/logger"
	"github.com/lamboktulus1379/issuing-ledger-service/usecase"

	"github.com/gin-gonic/gin"
)

const (
	ErrorUnmarshal = "Error while unmarshal"
)

type IUserHandler interface {
	Login(c *gin.Context)
	Register(c *gin.Context)
}

type UserHandler struct {
	userUsecase usecase.IUserUsecase
}

func NewUserHandler(userUsecase usecase.IUserUsecase) IUserHandler {
	return &UserHandler{userUsecase: userUsecase}
}

func (userHandler *UserHandler) Login(c *gin.Context) {
	var req model.ReqLogin

	if err := c.ShouldBindJSON(&req); err != nil {
		logger.GetLogger().WithField("error", err).Error(ErrorUnmarshal)
		c.JSON(http.StatusBadRequest, fmt.Sprintf("%s %v", ErrorUnmarshal, err.Error()))
		return
	}

	res := userHandler.userUsecase.Login(c.Request.Context(), req)

	c.JSON(http.StatusOK, res)
}

func (userHandler *UserHandler) Register(c *gin.Context) {
	var req model.ReqRegister

	if err := c.ShouldBindJSON(&req); err != nil {
		logger.GetLogger().WithField("error", err).Error(ErrorUnmarshal)
		c.JSON(http.StatusBadRequest, fmt.Sprintf("An error occurred: %v", err.Error()))
		return
	}

	res := userHandler.userUsecase.Register(c.Request.Context(), req)

	c.JSON(http.StatusOK, res)
}
