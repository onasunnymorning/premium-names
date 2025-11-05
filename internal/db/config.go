package db

import (
	"fmt"
	"os"
)

// Config holds PostgreSQL connection parameters.
type Config struct {
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
	SSLMode  string // disable, require, verify-ca, verify-full
	// If provided, DSN takes precedence over other fields.
	DSN string
}

// FromEnv loads configuration from environment variables.
// DB_DSN overrides individual fields if set.
func FromEnv() Config {
	cfg := Config{
		Host:     getEnv("DB_HOST", "localhost"),
		Port:     getEnvInt("DB_PORT", 5432),
		User:     getEnv("DB_USER", "postgres"),
		Password: getEnv("DB_PASSWORD", ""),
		DBName:   getEnv("DB_NAME", "postgres"),
		SSLMode:  getEnv("DB_SSLMODE", "disable"),
		DSN:      os.Getenv("DB_DSN"),
	}
	return cfg
}

func (c Config) ConnString() string {
	if c.DSN != "" {
		return c.DSN
	}
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		c.User, urlEncode(c.Password), c.Host, c.Port, c.DBName, c.SSLMode,
	)
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getEnvInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		var n int
		_, _ = fmt.Sscanf(v, "%d", &n)
		if n != 0 {
			return n
		}
	}
	return def
}

// urlEncode performs a minimal percent-encoding for passwords inside DSN.
func urlEncode(s string) string {
	// Only encode % and @ and : to be safe. This is minimal; users can pass DSN directly if needed.
	replacer := map[rune]string{
		'%': "%25",
		'@': "%40",
		':': "%3A",
	}
	out := make([]rune, 0, len(s))
	for _, r := range s {
		if enc, ok := replacer[r]; ok {
			for _, er := range enc {
				out = append(out, er)
			}
			continue
		}
		out = append(out, r)
	}
	return string(out)
}
