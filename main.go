package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/lamboktulus1379/issuing-ledger-service/domain/repository"
	"github.com/lamboktulus1379/issuing-ledger-service/infrastructure/cache"
	tulushost "github.com/lamboktulus1379/issuing-ledger-service/infrastructure/clients/tulustech"
	"github.com/lamboktulus1379/issuing-ledger-service/infrastructure/configuration"
	"github.com/lamboktulus1379/issuing-ledger-service/infrastructure/filecsv"
	"github.com/lamboktulus1379/issuing-ledger-service/infrastructure/googlesheet"
	"github.com/lamboktulus1379/issuing-ledger-service/infrastructure/logger"
	"github.com/lamboktulus1379/issuing-ledger-service/infrastructure/persistence"
	"github.com/lamboktulus1379/issuing-ledger-service/infrastructure/pubsub"
	"github.com/lamboktulus1379/issuing-ledger-service/infrastructure/servicebus"
	httpHandler "github.com/lamboktulus1379/issuing-ledger-service/interfaces/http"
	"github.com/lamboktulus1379/issuing-ledger-service/server"
	"github.com/lamboktulus1379/issuing-ledger-service/usecase"

	"golang.org/x/sync/errgroup"
)

var httpServer *http.Server

func recoverPanic() {
	if err := recover(); err != nil {
		logger.GetLogger().WithField("error", err).Error("Application panic recovered")
	}
}

func main() {
	// InitiateGoroutine()
	defer recoverPanic()
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(interrupt)

	g, ctx := errgroup.WithContext(ctx)

	// Load env from files (non-destructive; OS env still has precedence)
	configuration.LoadEnvFromFile("config.env", ".env")
	// Log which env files are present to help diagnose prod config loading
	if _, err := os.Stat("config.env"); err == nil {
		logger.GetLogger().Info("Detected config.env in working directory")
	} else {
		logger.GetLogger().Info("config.env not found in working directory")
	}
	if _, err := os.Stat(".env"); err == nil {
		logger.GetLogger().Info("Detected .env in working directory")
	} else {
		logger.GetLogger().Info(".env not found in working directory")
	}
	// configuration.LoadConfig()

	app := configuration.C.App

	mysqlDB, err := InitiateDatabase()
	if err != nil {
		logger.GetLogger().WithField("error", err).Error("Database initialization failed")
	}

	// mongoDb, err := persistence.NewMongoDb(
	// 	configuration.C.Database.Mongo.Host,
	// 	configuration.C.Database.Mongo.Port,
	// 	configuration.C.Database.Mongo.User,
	// 	configuration.C.Database.Mongo.Password,
	// 	configuration.C.Database.Mongo.Name,
	// )
	// if err != nil {
	// 	logger.GetLogger().WithField("error", err).Warn("MongoDB not available - continuing without Mongo features")
	// 	mongoDb = nil
	// } else {
	// 	if err := mongoDb.Ping(ctx, nil); err != nil {
	// 		logger.GetLogger().WithField("error", err).Warn("MongoDB ping failed - continuing without Mongo features")
	// 		mongoDb = nil
	// 	} else {
	// 		logger.GetLogger().Info("MongoDB connected successfully")
	// 	}
	// }

	// Log DB connectivity safely
	// var psqlPing any
	// if psqlDb != nil {
	// 	psqlPing = psqlDb.Ping()
	// } else {
	// 	psqlPing = "nil"
	// }
	logger.GetLogger().
		WithField("PrimaryDB", mysqlDB.Ping()).
		// WithField("PSQLDb", psqlPing).
		Info("Database connected.")

	pubSubClient, err := pubsub.NewPubSub(ctx, configuration.C.Pubsub.ProjectID)
	if err != nil {
		logger.GetLogger().WithField("error", err).Error("Error while instantiate PubSub")
		pubSubClient = nil // Set to nil for graceful handling
	}

	azServiceBusClient, err := servicebus.NewServiceBus(ctx, configuration.C.ServiceBus.Namespace)
	if err != nil {
		logger.GetLogger().WithField("error", err).Warn("Azure Service Bus not available - continuing without Service Bus features")
		azServiceBusClient = nil
	}
	redisClient, err := cache.NewCache(
		ctx,
		fmt.Sprintf("%s:%s", configuration.C.RedisClient.Host, configuration.C.RedisClient.Port),
		configuration.C.RedisClient.Username,
		configuration.C.RedisClient.Password,
	)

	if err != nil {
		logger.GetLogger().WithField("error", err).Error("Error while connection to Redis")
	}

	// testRepository := persistence.NewTestRepository(mongoDb, psqlDb)
	// project, err := testRepository.Test(ctx)
	// if err != nil {
	// logger.GetLogger().WithField("error", err).Error("Error while fetching data")
	// }
	// logger.GetLogger().WithField("project", project).Info("Test project data retrieved")
	testCache := cache.NewTestCache(redisClient)

	logger.GetLogger().Info("Redis client initialized successfully.")

	tulusTechHost := tulushost.NewTulusHost(configuration.C.TulusTech.Host)

	testPubSub := pubsub.NewTestPubSub(pubSubClient)
	testServiceBus := servicebus.NewTestServiceBus(azServiceBusClient)

	// Repository wiring: use MSSQL in production, otherwise PostgreSQL.
	var userRepository repository.IUser
	// if psqlDb == nil { // production/MSSQL path from InitiateDatabase
	userRepository = persistence.NewUserRepository(mysqlDB)
	// } else {
	// userRepository = persistence.NewUserRepository(psqlDb)
	// }
	userUsecase := usecase.NewUserUsecase(userRepository)
	testUsecase := usecase.NewTestUsecase(tulusTechHost, testPubSub, testServiceBus, testCache)

	userHandler := httpHandler.NewUserHandler(userUsecase)
	testHandler := httpHandler.NewTestHandler(testUsecase)

	router := server.InitiateRouter(userHandler, testHandler, userRepository)

	if err != nil {
		logger.GetLogger().WithField("error", err).Error("Error while StartSubscription")
	}

	// Comment out Test() function to prevent Google Sheets OAuth blocking
	// Test()

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)
	defer func() {
		signal.Stop(signalChan)
		cancel()
	}()

	port := app.Port
	logger.GetLogger().WithFields(map[string]interface{}{"port": port, "tls": app.TLSEnabled}).Info("Starting application")
	g.Go(func() error {
		httpServer = &http.Server{
			Addr:         fmt.Sprintf(":%d", port),
			Handler:      router,
			ReadTimeout:  0,
			WriteTimeout: 0,
		}
		// Keep reference for graceful shutdown
		// (re-use package-level httpServer variable if needed elsewhere)
		if app.TLSEnabled {
			cert := app.TLSCertFile
			key := app.TLSKeyFile
			if cert == "" || key == "" {
				logger.GetLogger().Error("TLS enabled but cert or key path empty; falling back to HTTP")
				if err := httpServer.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
					return err
				}
			} else {
				logger.GetLogger().WithFields(map[string]interface{}{"cert": cert, "key": key}).Info("Serving HTTPS")
				if err := httpServer.ListenAndServeTLS(cert, key); !errors.Is(err, http.ErrServerClosed) {
					return err
				}
			}
		} else {
			if err := httpServer.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
				return err
			}
		}
		return nil
	})

	select {
	case <-interrupt:
		logger.GetLogger().Info("Application shutdown requested")
		os.Exit(1)
	case <-ctx.Done():
		break
	}

	cancel()
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if httpServer != nil {
		_ = httpServer.Shutdown(shutdownCtx)
	}

	err = g.Wait()
	if err != nil {
		logger.GetLogger().WithField("error", err).Error("Server returned an error")
		os.Exit(2)
	}
}

func InitiateDatabase() (*sql.DB, error) {
	// Default/local: keep current behavior (MySQL native + PostgreSQL)
	db, err := persistence.NewNativeDb()
	if err != nil {
		logger.GetLogger().WithField("error", err).Error("Cannot connect to the local database")
		return nil, err
	}
	return db, nil
}

func InitiateGoroutine() {
	logger.GetLogger().Info("Initializing goroutines")

	for i := 0; i < 10; i++ {
		go func(id int) {
			logger.GetLogger().WithField("goroutine_id", id).Debug("Goroutine started")
		}(i)
	}
}

func Test() {
	file, err := filecsv.NewFile("cover.txt")
	if err != nil {
		logger.GetLogger().WithField("error", err).Error("Error while loading file")
	}

	validateCsv := filecsv.NewValidateCsv(file)
	logger.GetLogger().WithField("validateCsv", validateCsv).Info("Validate CSV initialized")

	googleSheet, err := googlesheet.NewGoogleSheet()
	if err != nil {
		logger.GetLogger().WithField("error", err).Error("Error while loading Google Sheet")
	}

	logger.GetLogger().WithField("googleSheet", googleSheet).Info("Google sheet initialized")
}
