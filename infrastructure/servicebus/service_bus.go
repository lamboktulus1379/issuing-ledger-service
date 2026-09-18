package servicebus

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus"
)

func NewServiceBus(ctx context.Context, namespace string) (*azservicebus.Client, error) {
	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		return nil, err
	}

	client, err := azservicebus.NewClient(namespace, cred, nil)
	if err != nil {
		return nil, err
	}
	return client, nil
}
