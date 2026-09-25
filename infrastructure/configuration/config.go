package configuration

import (
	"fmt"
	"os"
	"strconv"

	"github.com/lamboktulus1379/issuing-ledger-service/infrastructure/logger"

	"github.com/spf13/viper"
)

type Config struct {
	Database    Database    `json:"database"`
	TulusTech   TulusTech   `json:"tulusTech"`
	App         App         `json:"app"`
	GoogleSheet GoogleSheet `json:"googleSheet"`
	Data        Data        `json:"data"`
	Pubsub      Pubsub      `json:"pubsub"`
	ServiceBus  ServiceBus  `json:"serviceBus"`
	RedisClient RedisClient `json:"redisClient"`
	Logger      Logger      `json:"logger"`
	YouTube     YouTube     `json:"youtube"`
	Share       Share       `json:"share"`
	OAuth       OAuth       `json:"oauth"`
}

type App struct {
	Port        int    `json:"port"`
	SecretKey   string `json:"secretKey"`
	TLSEnabled  bool   `json:"tlsEnabled"`
	TLSCertFile string `json:"tlsCertFile"`
	TLSKeyFile  string `json:"tlsKeyFile"`
}

type Database struct {
	Psql  Db `json:"psql"`
	MySql Db `json:"mysql"`
	Mongo Db `json:"mongo"`
	Mssql Db `json:"mssql"`
}

type GoogleSheet struct {
	Type                       int    `json:"type"`
	SpreadsheetId              string `json:"spreadsheetId"`
	SpreadsheetColumnReadRange string `json:"spreadsheetColumnReadRange"`
	SpreadsheetName            string `json:"spreadsheetName"`
	SpreadsheetDescription     string `json:"spreadsheetDescription"`
}

type Db struct {
	Name     string `json:"string"`
	Host     string `json:"host"`
	Port     string `json:"port"`
	User     string `json:"user"`
	Password string `json:"password"`
}

type TulusTech struct {
	Header Header `json:"header"`
	Host   string `json:"host"`
}

type Data struct {
	Source string `json:"source"`
}

type Header struct {
	Accept          string `json:"accept"`
	AcceptLanguage  string `json:"acceptLanguage"`
	Connection      string `json:"connection"`
	ContentType     string `json:"contentType"`
	Cookie          string `json:"cookie"`
	Origin          string `json:"origin"`
	Referer         string `json:"referer"`
	SecFetchDest    string `json:"secFetchDest"`
	SecFetchMode    string `json:"secFetchMode"`
	SectFetchSite   string `json:"secFetchSite"`
	UserAgent       string `json:"userAgent"`
	XRequestedWith  string `json:"xRequestedWith"`
	SecChUa         string `json:"secChUa"`
	SecChUaMobile   string `json:"secChUaMobile"`
	SecChUaPlatform string `json:"secChUaPlatform"`
}

type Pubsub struct {
	ProjectID string `json:"projectID"`
}

type ServiceBus struct {
	Namespace string `json:"namespace"`
}

type RedisClient struct {
	Host         string `json:"host"`
	Port         string `json:"port"`
	Password     string `json:"password"`
	DatabaseName string `json:"databaseName"`
	Username     string `json:"username"`
}

type Logger struct {
	Format string `json:"format"`
}

type YouTube struct {
	APIKey       string   `json:"apiKey"`
	ClientID     string   `json:"clientId"`
	ClientSecret string   `json:"clientSecret"`
	RedirectURI  string   `json:"redirectURI"`
	ChannelID    string   `json:"channelId"`
	Scopes       []string `json:"scopes"`
}

// Share holds share platform configuration
type Share struct {
	Platforms []string `json:"platforms"`
}

// OAuth holds third-party platform OAuth client credentials
type OAuth struct {
	Facebook OAuthClient `json:"facebook"`
	Twitter  OAuthClient `json:"twitter"`
}

type OAuthClient struct {
	ClientID     string `json:"clientId"`
	ClientSecret string `json:"clientSecret"`
	RedirectURI  string `json:"redirectURI"`
	// Additional fields (scopes, etc) can be added later
}

var C Config

func init() {
	LoadConfig()
	initDatabase(&C)
	initRedis(&C)
	initApp(&C)
	// Prefer https redirect URIs locally when TLS enabled
	if C.App.TLSEnabled {
		if C.YouTube.RedirectURI != "" && !hasHTTPS(C.YouTube.RedirectURI) {
			C.YouTube.RedirectURI = toHTTPSCallback(C.YouTube.RedirectURI)
		}
		if C.OAuth.Facebook.RedirectURI != "" && !hasHTTPS(C.OAuth.Facebook.RedirectURI) {
			C.OAuth.Facebook.RedirectURI = toHTTPSCallback(C.OAuth.Facebook.RedirectURI)
		}
	}
}

func initRedis(C *Config) {
	// Inside Docker, Redis must be addressed by its Compose service name. An
	// environment override prevents config.json's localhost default from being
	// used by a containerized application.
	if v := os.Getenv("REDIS_HOST"); v != "" {
		C.RedisClient.Host = v
	}
	if v := os.Getenv("REDIS_PORT"); v != "" {
		C.RedisClient.Port = v
	}
	if v := os.Getenv("REDIS_PASSWORD"); v != "" {
		C.RedisClient.Password = v
	}
	if v := os.Getenv("REDIS_USERNAME"); v != "" {
		C.RedisClient.Username = v
	}
}

func LoadConfig() {
	name := getConfig()
	viper.SetConfigName(name)
	viper.SetConfigType("json")
	viper.AddConfigPath(".")
	viper.AddConfigPath("../")
	viper.AddConfigPath("../../")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Config file not found; ignore error if desired
			logger.GetLogger().Warn("Config file not found")
		} else {
			// Config file was found but another error was produced
			logger.GetLogger().WithField("error", err).Error("Error reading config file")
		}
	}

	logger.GetLogger().WithField("config", name).Info("Config set up successfully")
	// Config file found and successfully parsed
	if err := viper.Unmarshal(&C); err != nil {
		logger.GetLogger().WithField("error", err).Error("Viper unable to decode into struct")
	}
}

func getConfig() string {
	name := "config"
	env := os.Getenv("ENV")
	if env != "" {
		name = fmt.Sprintf("%s-%s", name, env)
	}
	return name
}

func initDatabase(C *Config) {
	logger.GetLogger().WithField("Database", C.Database.Psql).Info("Database configuration")
	if C.Database.Psql.Name == "" {
		C.Database.Psql.Name = os.Getenv("DB_NAME")
	}
	if C.Database.Psql.Host == "" {
		C.Database.Psql.Host = os.Getenv("DB_HOST")
	}
	if C.Database.Psql.Password == "" {
		C.Database.Psql.Password = os.Getenv("DB_PASSWORD")
	}
	if C.Database.Psql.Port == "" {
		C.Database.Psql.Port = os.Getenv("DB_PORT")
	}

	// MySQL/MariaDB uses its own environment namespace. This is important when
	// the application runs in Docker: MYSQL_HOST must be the Compose service
	// name (for example, "mariadb"), never localhost.
	if C.Database.MySql.Name == "" {
		if v := os.Getenv("MYSQL_DATABASE"); v != "" {
			C.Database.MySql.Name = v
		} else if v := os.Getenv("MYSQL_DB_NAME"); v != "" {
			C.Database.MySql.Name = v
		}
	}
	if v := os.Getenv("MYSQL_HOST"); v != "" {
		C.Database.MySql.Host = v
	}
	if v := os.Getenv("MYSQL_PORT"); v != "" {
		C.Database.MySql.Port = v
	}
	if v := os.Getenv("MYSQL_USER"); v != "" {
		C.Database.MySql.User = v
	}
	if v := os.Getenv("MYSQL_PASSWORD"); v != "" {
		C.Database.MySql.Password = v
	} else if v := os.Getenv("MYSQL_ROOT_PASSWORD"); v != "" && C.Database.MySql.User == "root" {
		C.Database.MySql.Password = v
	}

	// Optional MSSQL config via environment variables (for Azure SQL in production)
	if C.Database.Mssql.Name == "" {
		if v := os.Getenv("MSSQL_DB_NAME"); v != "" {
			C.Database.Mssql.Name = v
		}
	}
	if C.Database.Mssql.Host == "" {
		if v := os.Getenv("MSSQL_HOST"); v != "" {
			C.Database.Mssql.Host = v
		}
	}
	if C.Database.Mssql.Password == "" {
		if v := os.Getenv("MSSQL_PASSWORD"); v != "" {
			C.Database.Mssql.Password = v
		}
	}
	if C.Database.Mssql.Port == "" {
		if v := os.Getenv("MSSQL_PORT"); v != "" {
			C.Database.Mssql.Port = v
		} else {
			C.Database.Mssql.Port = "1433"
		}
	}
	if C.Database.Mssql.User == "" {
		if v := os.Getenv("MSSQL_USER"); v != "" {
			C.Database.Mssql.User = v
		}
	}

	// Fill local/dev sensible defaults for MSSQL if still empty (matches Typing docker-compose)
	if C.Database.Mssql.Host == "" {
		C.Database.Mssql.Host = "localhost"
	}
	if C.Database.Mssql.Port == "" {
		C.Database.Mssql.Port = "1433"
	}
	// Default to SA user for local container only when nothing provided
	if C.Database.Mssql.User == "" {
		C.Database.Mssql.User = "sa"
	}
	if C.Database.Mssql.Password == "" {
		// Matches MSSQL_SA_PASSWORD in Typing docker-compose; safe for local dev only
		C.Database.Mssql.Password = "Toughpass1!"
	}
}

func initApp(C *Config) {
	// Prefer SECRET_KEY from environment for JWT verification; overrides config file when provided
	if v := os.Getenv("SECRET_KEY"); v != "" {
		C.App.SecretKey = v
	}
	// Port resolution order (env overrides config): APP_PORT -> PORT -> config -> default 10001
	if v := os.Getenv("APP_PORT"); v != "" {
		if p, err := strconv.Atoi(v); err == nil {
			C.App.Port = p
		}
	} else if v := os.Getenv("PORT"); v != "" {
		if p, err := strconv.Atoi(v); err == nil {
			C.App.Port = p
		}
	}
	if C.App.Port == 0 {
		C.App.Port = 10001
	}
	// Allow overriding TLS settings via env variables (both enable and disable)
	if v := os.Getenv("TLS_ENABLED"); v != "" {
		switch v {
		case "1", "true", "TRUE", "True":
			C.App.TLSEnabled = true
		case "0", "false", "FALSE", "False":
			C.App.TLSEnabled = false
		}
	}
	if C.App.TLSCertFile == "" {
		C.App.TLSCertFile = os.Getenv("TLS_CERT_FILE")
	}
	if C.App.TLSKeyFile == "" {
		C.App.TLSKeyFile = os.Getenv("TLS_KEY_FILE")
	}
	// Prefer local certs if TLS enabled and paths not provided
	if C.App.TLSEnabled {
		if C.App.TLSCertFile == "" {
			if _, err := os.Stat("certs/localhost.crt"); err == nil {
				C.App.TLSCertFile = "certs/localhost.crt"
			}
		}
		if C.App.TLSKeyFile == "" {
			if _, err := os.Stat("certs/localhost.key"); err == nil {
				C.App.TLSKeyFile = "certs/localhost.key"
			}
		}
	}
	if C.App.TLSEnabled {
		logger.GetLogger().WithFields(map[string]interface{}{"cert": C.App.TLSCertFile, "key": C.App.TLSKeyFile}).Info("TLS enabled via configuration")
	}
	if C.App.SecretKey == "" {
		logger.GetLogger().Warn("App.SecretKey not set; JWT authentication will fail. Provide SECRET_KEY via environment.")
	}
}

// helpers to coerce local callback to https
func hasHTTPS(u string) bool { return len(u) >= 8 && u[:8] == "https://" }
func toHTTPSCallback(u string) string {
	// simple swap for localhost callbacks
	if len(u) >= 7 && u[:7] == "http://" {
		return "https://" + u[7:]
	}
	return u
}
