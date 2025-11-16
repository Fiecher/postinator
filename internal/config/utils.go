package config

import (
	"log"
	"os"
	"strconv"
)

func Load() *Config {
	token := os.Getenv("TOKEN")
	if token == "" {
		log.Fatal("TOKEN environment variable is required")
	}

	return &Config{
		BotToken:    token,
		AssetsDir:   getEnv("ASSETS_DIR", "./assets", parseString),
		TempDir:     getEnv("TEMP_DIR", "./temp", parseString),
		MaxFileSize: getEnv("MAX_FILE_SIZE", 10*1024*1024, parseInt),
	}
}

func getEnv[T any](key string, defaultValue T, parser func(string) (T, error)) T {
	val := os.Getenv(key)
	if val == "" {
		return defaultValue
	}

	parsed, err := parser(val)
	if err != nil {
		log.Printf("WARNING: invalid value for %s (%s). Using default: %v\n", key, val, defaultValue)
		return defaultValue
	}

	return parsed
}

func parseString(val string) (string, error) {
	return val, nil
}

func parseInt(val string) (int, error) {
	return strconv.Atoi(val)
}
