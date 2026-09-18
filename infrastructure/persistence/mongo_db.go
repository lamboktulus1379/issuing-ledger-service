package persistence

import (
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"github.com/lamboktulus1379/issuing-ledger-service/infrastructure/logger"
)

func NewMongoDb(
	host string,
	port string,
	username string,
	password string,
	database string,
) (*mongo.Client, error) {
	// connect to mongodb
	client, err := mongo.Connect(
		options.Client().
			ApplyURI("mongodb://" + username + ":" + password + "@" + host + ":" + port + "/" + database + "s?authSource=admin&authMechanism=SCRAM-SHA-256"),
	)
	if err != nil {
		logger.GetLogger().WithField("error", err).Error("Failed to connect to MongoDB")
		return nil, err
	}

	return client, nil
}
