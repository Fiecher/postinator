package config

import (
	"log"
	"os"
	"strconv"
)

func Load(logger *log.Logger) *Config {
	token := os.Getenv("TOKEN")
	if token == "" {
		logger.Fatal("TOKEN environment variable is required")
	}

	return &Config{
		BotToken:    token,
		AssetsDir:   getEnv(logger, "ASSETS_DIR", "./assets", parseString),
		TempDir:     getEnv(logger, "TEMP_DIR", "./temp", parseString),
		MaxFileSize: getEnv(logger, "MAX_FILE_SIZE", 10*1024*1024, parseInt),
	}
}

func getEnv[T any](logger *log.Logger, key string, defaultValue T, parser func(string) (T, error)) T {
	val := os.Getenv(key)
	if val == "" {
		return defaultValue
	}

	parsed, err := parser(val)
	if err != nil {
		logger.Printf("[WARN]: invalid value for %s (%s). Using default: %v\n", key, val, defaultValue)
		return defaultValue
	}

	return parsed
}

func parseString(val string) (string, error) {
	return val, nil
}

func parseInt(val string) (int64, error) {
	return strconv.ParseInt(val, 10, 64)
}
