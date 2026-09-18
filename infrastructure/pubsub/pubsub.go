package pubsub

import (
	"context"
	"log"

	"cloud.google.com/go/pubsub"
)

var topic *pubsub.Topic

// Messages received by this instance.
// messagesMu sync.Mutex
// messages   []string
// token is used to verify push requests.

type PubSubHandler struct {
	Topic        string
	Subscription string
	Handler      Handler
}

type Handler func(ctx context.Context, msg *pubsub.Message)

func NewPubSub(ctx context.Context, projectID string) (*pubsub.Client, error) {
	client, err := pubsub.NewClient(ctx, projectID)
	if err != nil {
		return nil, err
	}

	log.Printf("PubSub connected...")
	return client, err
}
