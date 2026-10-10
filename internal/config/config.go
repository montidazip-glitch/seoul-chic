package config

import (
	"fmt"
	"log"
	"net/url"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

// Config holds all application configuration loaded from environment variables.
type Config struct {
	Port          string
	DBDSN         string
	SessionSecret string
}

// Load reads the .env file (if present) and returns a populated Config.
func Load() *Config {
	// Silently ignore missing .env — in production, vars are set externally.
	if err := godotenv.Load(); err != nil {
		log.Println("[config] No .env file found, reading from environment")
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = getEnv("SERVER_PORT", "8080")
	}

	rawDSN := os.Getenv("DATABASE_URL")
	if rawDSN == "" {
		rawDSN = os.Getenv("MYSQL_URL")
	}
	if rawDSN == "" {
		rawDSN = getEnv("DB_DSN", "root:@tcp(127.0.0.1:3306)/kfry_shop?parseTime=true&charset=utf8mb4")
	}

	cfg := &Config{
		Port:          port,
		DBDSN:         normalizeDSN(rawDSN),
		SessionSecret: getEnv("SESSION_SECRET", "kfry-secret-change-in-production-32b"),
	}
	return cfg
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// normalizeDSN converts standard mysql:// URL formats (common on cloud hosts) to Go MySQL DSN.
func normalizeDSN(raw string) string {
	raw = strings.TrimSpace(raw)
	if strings.HasPrefix(raw, "mysql://") {
		u, err := url.Parse(raw)
		if err == nil {
			user := ""
			pass := ""
			if u.User != nil {
				user = u.User.Username()
				pass, _ = u.User.Password()
			}
			host := u.Host
			if !strings.Contains(host, ":") {
				host = host + ":3306"
			}
			dbName := strings.TrimPrefix(u.Path, "/")
			query := u.RawQuery
			if query == "" {
				query = "parseTime=true&charset=utf8mb4"
			} else {
				if !strings.Contains(query, "parseTime") {
					query += "&parseTime=true"
				}
				if !strings.Contains(query, "charset") {
					query += "&charset=utf8mb4"
				}
			}
			auth := user
			if pass != "" {
				auth = user + ":" + pass
			}
			return fmt.Sprintf("%s@tcp(%s)/%s?%s", auth, host, dbName, query)
		}
	}
	return raw
}

