package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	log "github.com/sirupsen/logrus"
)

var logger = log.New()

func init() {
	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		log.Info("Failed get current working directory")
		log.Fatal(err)
	}
	layout := "2006-01-02"
	env := os.Getenv("ENV")
	fmt.Println("ENV", env)
	formatTime := time.Now().Format(layout)
	// Prefer stdout for non-local environments (better with systemd/docker).
	// Allow overriding via LOG_TO_FILE=true to force file logging.
	logToFile := os.Getenv("LOG_TO_FILE") == "true"
	if env == "stage" || env == "prod" || env == "" {
		if logToFile {
			// Ensure logs directory exists
			logsDir := filepath.Join(cwd, "logs")
			if mkErr := os.MkdirAll(logsDir, 0o755); mkErr != nil {
				log.Warnf("Failed to create logs directory %s: %v, falling back to stdout", logsDir, mkErr)
				logger.Out = os.Stdout
			} else {
				filePath := filepath.Join(logsDir, fmt.Sprintf("%s%s.log", formatTime, env))
				f, openErr := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o666)
				if openErr != nil {
					log.Warnf("Failed to open log file %s: %v, falling back to stdout", filePath, openErr)
					logger.Out = os.Stdout
				} else {
					logger.Out = f
				}
			}
		} else {
			// Default to stdout for prod/stage unless explicitly overridden.
			logger.Out = os.Stdout
		}
	} else {
		// For any other envs, default to stdout.
		logger.Out = os.Stdout
	}

	logger.Formatter = &log.JSONFormatter{
		TimestampFormat: time.RFC3339Nano,
	}
	logger.SetLevel(log.DebugLevel)
}

func GetLogger() *log.Entry {
	// The API for setting attributes is a little different than the package level
	// exported logger. See Godoc.
	// log.Out = os.Stdout

	// You could set this to any `io.Writer` such as a file
	// file, err := os.OpenFile("log.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	// if err == nil {
	//  log.Out = file
	// } else {
	//  log.Info("Failed to log to file, using default stderr")
	// }
	function, file, line, _ := runtime.Caller(1)

	functionObject := runtime.FuncForPC(function)
	entry := logger.WithFields(log.Fields{
		"requestId": time.Now().UnixNano() / int64(time.Millisecond),
		"size":      10,
		"function":  functionObject.Name(),
		"file":      file,
		"line":      line,
	})

	return entry
}
