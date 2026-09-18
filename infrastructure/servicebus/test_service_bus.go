package servicebus

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus"
	"github.com/lamboktulus1379/issuing-ledger-service/infrastructure/logger"
)

type ITestServiceBus interface {
	SendMessage(message []byte) error
	GetMessage(count int)
}

type TestServicebus struct {
	AzservicebusClient *azservicebus.Client
}

func NewTestServiceBus(azServiceBusClient *azservicebus.Client) ITestServiceBus {
	return &TestServicebus{AzservicebusClient: azServiceBusClient}
}

func (testServiceBus *TestServicebus) SendMessage(message []byte) error {
	sender, err := testServiceBus.AzservicebusClient.NewSender("test-queue", nil)
	if err != nil {
		logger.GetLogger().
			WithField("error", err).
			Error("Error while making new sender service bus.")
		return err
	}
	defer func(sender *azservicebus.Sender, ctx context.Context) {
		err := sender.Close(ctx)
		if err != nil {
			logger.GetLogger().
				WithField("error", err).
				Error("Error while closing sender.")
		}
	}(sender, context.Background())

	sbMessage := &azservicebus.Message{
		Body: message,
	}
	err = sender.SendMessage(context.Background(), sbMessage, nil)
	if err != nil {
		logger.GetLogger().WithField("error", err).Error("Error while sending message.")
		return err
	}

	return nil
}

func (testServiceBus *TestServicebus) GetMessage(count int) {
	receiver, err := testServiceBus.AzservicebusClient.NewReceiverForQueue("testqueue", nil)
	if err != nil {
		panic(err)
	}
	defer func(receiver *azservicebus.Receiver, ctx context.Context) {
		err := receiver.Close(ctx)
		if err != nil {
			logger.GetLogger().
				WithField("error", err).
				Error("Error while closing receiver.")
		}
	}(receiver, context.Background())

	messages, err := receiver.ReceiveMessages(context.Background(), count, nil)
	if err != nil {
		panic(err)
	}

	for _, message := range messages {
		body := message.Body
		fmt.Printf("%s\n", string(body))

		err = receiver.CompleteMessage(context.Background(), message, nil)
		if err != nil {
			panic(err)
		}
	}
}
