package http

import (
	"net/http"

	"github.com/lamboktulus1379/issuing-ledger-service/usecase"

	"github.com/gin-gonic/gin"
)

type ITestHandler interface {
	Test(c *gin.Context)
	Healthz(c *gin.Context)
}

type TestHandler struct {
	TestUsecase usecase.ITestUsecase
}

func NewTestHandler(testUsecase usecase.ITestUsecase) ITestHandler {
	return &TestHandler{TestUsecase: testUsecase}
}

func (testHandler *TestHandler) Test(c *gin.Context) {
	res := testHandler.TestUsecase.Test(c.Request.Context())
	c.JSON(http.StatusOK, res)
}

// Healthz returns OK for health checks
func (h *TestHandler) Healthz(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, gin.H{"status": "ok"})
}
