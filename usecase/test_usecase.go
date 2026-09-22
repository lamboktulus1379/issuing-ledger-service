package usecase

import (
	"context"
	"encoding/json"

	"github.com/lamboktulus1379/issuing-ledger-service/domain/dto"
	"github.com/lamboktulus1379/issuing-ledger-service/infrastructure/cache"
	tulushost "github.com/lamboktulus1379/issuing-ledger-service/infrastructure/clients/tulustech"
	"github.com/lamboktulus1379/issuing-ledger-service/infrastructure/clients/tulustech/models"
	"github.com/lamboktulus1379/issuing-ledger-service/infrastructure/logger"
	"github.com/lamboktulus1379/issuing-ledger-service/infrastructure/pubsub"
	"github.com/lamboktulus1379/issuing-ledger-service/infrastructure/servicebus"
)

type ITestUsecase interface {
	Test(ctx context.Context) dto.TestDto
}

type TestUsecase struct {
	TulusTechHost  tulushost.ITulusHost
	TestPubSub     pubsub.ITestPubSub
	TestServiceBus servicebus.ITestServiceBus
	TestCache      cache.ITestCache
}

// This interface is defined in infrastructure/clients/tulustech/host.go
// Keeping this here would cause duplication and potential inconsistency
// type ITulusHost interface {
// 	GetRandomTyping(reqHeader models.ReqHeader) (models.ResTypingRandom, error)
// }

func NewTestUsecase(
	tulusTechHost tulushost.ITulusHost,
	testPubSub pubsub.ITestPubSub,
	testServiceBus servicebus.ITestServiceBus,
	testCache cache.ITestCache,
) ITestUsecase {
	return &TestUsecase{
		TulusTechHost:  tulusTechHost,
		TestPubSub:     testPubSub,
		TestServiceBus: testServiceBus,
		TestCache:      testCache,
	}
}

func (testUsecase *TestUsecase) Test(ctx context.Context) dto.TestDto {
	var res dto.TestDto

	res.PubSub = "Not OK"
	res.ServiceBus = "Not OK"

	msg := "Hello"
	byteMsg, err := json.Marshal(msg)
	if err != nil {
		logger.GetLogger().Error("Error while marshalling")
		// return res
	}
	publishResponse, err := testUsecase.TestPubSub.Publish(ctx, "topic", byteMsg)
	if err != nil {
		logger.GetLogger().Error("Error while publishing message")
		res.PubSub = err.Error()
		// return res
	} else {
		logger.GetLogger().WithField("publishResponse", publishResponse).Info("Successfully published")
		res.PubSub = "OK"
	}

	err = testUsecase.TestServiceBus.SendMessage(byteMsg)
	if err != nil {
		logger.GetLogger().Error("Error while publishing message with service bus")
		res.ServiceBus = err.Error()
		// return res
	} else {
		res.ServiceBus = "OK"
	}

	testUsecase.TestCache.Set(ctx, "test", "test")
	val, err := testUsecase.TestCache.Get(ctx, "test")
	if err != nil {
		logger.GetLogger().Error("Error while getting value from cache")
		res.Cache = "Error while getting value from cache"
		// return res
	} else {
		res.Cache = val.(string)
	}

	reqHeader := models.ReqHeader{}
	randomTypingRes, err := testUsecase.TulusTechHost.GetRandomTyping(reqHeader)
	if err != nil {
		logger.GetLogger().Error("Error while get random typing")
		res.TulusTech = err.Error()
		// return res
	} else {
		logger.GetLogger().
			WithField("randomTypingResponse", randomTypingRes).
			Info("Successfully get random typing")
		res.TulusTech = "OK"
	}

	return res
}
