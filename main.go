package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/lamboktulus1379/issuing-ledger-service/domain/repository"
	"github.com/lamboktulus1379/issuing-ledger-service/infrastructure/cache"
	tulushost "github.com/lamboktulus1379/issuing-ledger-service/infrastructure/clients/tulustech"
	"github.com/lamboktulus1379/issuing-ledger-service/infrastructure/configuration"
	"github.com/lamboktulus1379/issuing-ledger-service/infrastructure/filecsv"
	"github.com/lamboktulus1379/issuing-ledger-service/infrastructure/googlesheet"
	"github.com/lamboktulus1379/issuing-ledger-service/infrastructure/logger"
	"github.com/lamboktulus1379/issuing-ledger-service/infrastructure/message"
	"github.com/lamboktulus1379/issuing-ledger-service/infrastructure/persistence"
	"github.com/lamboktulus1379/issuing-ledger-service/infrastructure/persistence/ledger"
	"github.com/lamboktulus1379/issuing-ledger-service/infrastructure/persistence/outbox"
	"github.com/lamboktulus1379/issuing-ledger-service/infrastructure/pubsub"
	"github.com/lamboktulus1379/issuing-ledger-service/infrastructure/servicebus"
	httpHandler "github.com/lamboktulus1379/issuing-ledger-service/interfaces/http"
	"github.com/lamboktulus1379/issuing-ledger-service/server"
	"github.com/lamboktulus1379/issuing-ledger-service/usecase"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/sdk/resource"

	"github.com/segmentio/kafka-go"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	"golang.org/x/sync/errgroup"
)

var httpServer *http.Server

func recoverPanic() {
	if err := recover(); err != nil {
		logger.GetLogger().WithField("error", err).Error("Application panic recovered")
	}
}

func initTracer() (func(context.Context) error, error) {
	ctx := context.Background()
	otel.SetErrorHandler(otel.ErrorHandlerFunc(func(err error) {
		fmt.Printf("OpenTelemetry internal error: %v\n", err)
	}))

	endpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	if endpoint == "" {
		endpoint = "localhost:4317"
	}
	endpoint = strings.TrimPrefix(strings.TrimPrefix(endpoint, "http://"), "https://")

	// Point the exporter to your local OTel Collector
	exporter, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithEndpoint(endpoint),
		otlptracegrpc.WithInsecure(),
	)
	if err != nil {
		return nil, err
	}

	// Identify your application in Grafana
	res := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceNameKey.String("ledger-service"),
	)

	// Register the global trace provider
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)
	otel.SetTracerProvider(tp)

	return tp.Shutdown, nil
}

func main() {
	// Initialize OpenTelemetry
	shutdown, err := initTracer()
	if err != nil {
		panic(err)
	}
	defer func() {
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()
		if err := shutdown(shutdownCtx); err != nil {
			fmt.Printf("OpenTelemetry shutdown error: %v\n", err)
		}
	}()
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
		return
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
	if err := startOutboxStrategy(ctx, g, mysqlDB); err != nil {
		logger.GetLogger().WithField("error", err).Error("Outbox strategy initialization failed")
	}

	ledgerRepository, err := ledger.NewMariaDBRepository(mysqlDB)
	if err != nil {
		logger.GetLogger().WithField("error", err).Error("Error while instantiate ledger repository")
		return
	}
	issuingUsecase, err := usecase.NewIssuingUsecase(ledgerRepository)
	if err != nil {
		logger.GetLogger().WithField("error", err).Error("Error while instantiate issuing usecase")
		return
	}

	userHandler := httpHandler.NewUserHandler(userUsecase)
	testHandler := httpHandler.NewTestHandler(testUsecase)

	idempotencyStore := cache.NewIdempotencyStore(redisClient)
	issuingHandler, err := httpHandler.NewIssuingHandler(idempotencyStore, issuingUsecase)
	if err != nil {
		logger.GetLogger().WithField("error", err).Error("Error while initiate issuing handler")
		return
	}

	router := server.InitiateRouter(userHandler, testHandler, userRepository, issuingHandler)

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

// startOutboxStrategy keeps infrastructure selection at the application
// composition root. Polling remains the default for local development and CI;
// production can select CDC so Debezium publishes committed outbox inserts
// from MariaDB's binlog without running a second publisher in the Go process.
func startOutboxStrategy(ctx context.Context, g *errgroup.Group, db *sql.DB) error {
	strategy := strings.ToLower(strings.TrimSpace(os.Getenv("OUTBOX_STRATEGY")))
	if strategy == "" {
		strategy = "polling"
	}

	switch strategy {
	case "cdc":
		logger.GetLogger().Info("Outbox strategy is CDC; Debezium handles event propagation")
		return nil
	case "polling":
		if db == nil {
			return errors.New("polling outbox strategy requires a database connection")
		}
	default:
		return fmt.Errorf("unsupported OUTBOX_STRATEGY %q: expected polling or cdc", strategy)
	}

	brokers := splitCSVEnv("KAFKA_BROKERS", "localhost:9092")
	pollInterval, err := durationEnv("OUTBOX_POLL_INTERVAL", 1*time.Second)
	if err != nil {
		return err
	}
	batchSize, err := positiveIntEnv("OUTBOX_BATCH_SIZE", 100)
	if err != nil {
		return err
	}

	writer := kafka.NewWriter(kafka.WriterConfig{
		Brokers:  brokers,
		Balancer: &kafka.LeastBytes{},
	})
	relay, err := message.NewOutboxRelay(db, &outbox.Repository{}, writer, pollInterval, batchSize)
	if err != nil {
		_ = writer.Close()
		return fmt.Errorf("failed to create polling outbox relay: %w", err)
	}

	g.Go(func() error {
		defer writer.Close()
		if err := relay.Start(ctx); err != nil {
			return fmt.Errorf("polling outbox relay stopped: %w", err)
		}
		return nil
	})
	logger.GetLogger().WithFields(map[string]interface{}{
		"strategy":      strategy,
		"kafka_brokers": brokers,
		"poll_interval": pollInterval.String(),
		"batch_size":    batchSize,
	}).Info("Polling outbox relay started")
	return nil
}

func splitCSVEnv(name, fallback string) []string {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		value = fallback
	}
	parts := strings.Split(value, ",")
	brokers := make([]string, 0, len(parts))
	for _, part := range parts {
		if broker := strings.TrimSpace(part); broker != "" {
			brokers = append(brokers, broker)
		}
	}
	return brokers
}

func durationEnv(name string, fallback time.Duration) (time.Duration, error) {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback, nil
	}
	duration, err := time.ParseDuration(value)
	if err != nil || duration <= 0 {
		return 0, fmt.Errorf("invalid %s %q: expected a positive duration", name, value)
	}
	return duration, nil
}

func positiveIntEnv(name string, fallback int) (int, error) {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback, nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return 0, fmt.Errorf("invalid %s %q: expected a positive integer", name, value)
	}
	return parsed, nil
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
