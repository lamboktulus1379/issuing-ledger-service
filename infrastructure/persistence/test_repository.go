package persistence

import (
	"context"
	"database/sql"
	"time"

	"github.com/lamboktulus1379/issuing-ledger-service/domain/model"
	"github.com/lamboktulus1379/issuing-ledger-service/infrastructure/logger"
	"github.com/lamboktulus1379/issuing-ledger-service/infrastructure/worker"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

type ITestRepository interface {
	Test(ctx context.Context) ([]model.Project, error)
}

type TestRepository struct {
	mongoDb    *mongo.Client
	PostgresDB *sql.DB
}

func (t *TestRepository) Test(ctx context.Context) ([]model.Project, error) {
	// TODO: Remove this
	return []model.Project{}, nil
	myProjects := []model.Project{
		{
			Name:        "Project 1",
			Description: "Description 1",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
	}
	// Safely use Postgres worker if available
	if t.PostgresDB != nil {
		worker.PooledWorkError(myProjects, t.PostgresDB)
	} else {
		logger.GetLogger().Info("PostgresDB is nil - skipping worker pipeline")
	}

	// If Mongo is not available, return the local projects only
	if t.mongoDb == nil {
		logger.GetLogger().Info("MongoDB client is nil - returning static projects only")
		return myProjects, nil
	}

	collection := t.mongoDb.Database("my_project").Collection("projects")
	cursor, err := collection.Find(ctx, bson.D{})
	if err != nil {
		logger.GetLogger().WithField("error", err).Error("Error while fetching data")
		// Return static projects when Mongo fetch fails
		return myProjects, nil
	}
	defer func(cursor *mongo.Cursor, ctx context.Context) {
		err := cursor.Close(ctx)
		if err != nil {
			logger.GetLogger().WithField("error", err).Error("Error while closing cursor")
		}
	}(cursor, ctx)

	var projects []model.Project
	for cursor.Next(ctx) {
		var project model.Project
		err := cursor.Decode(&project)
		if err != nil {
			logger.GetLogger().WithField("error", err).Error("Error while decoding")
			continue
		}
		projects = append(projects, project)
	}
	if len(projects) == 0 {
		// Fallback to static projects if collection empty
		return myProjects, nil
	}
	return projects, nil
}

func NewTestRepository(db *mongo.Client, postgresDB *sql.DB) ITestRepository {
	return &TestRepository{mongoDb: db, PostgresDB: postgresDB}
}
