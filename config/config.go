package config

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	DBHost                 string
	DBPort                 string
	DBUser                 string
	DBPassword             string
	DBName                 string
	DBSSLMode              string
	JWTSecret              string
	JWTExpireHours         int
	RefreshTokenExpireDays int
	Port                   string
	GinMode                string
	CORSAllowedOrigins     string
	CORSAllowedMethods     string
	APIHost                string
}

func LoadConfig() *Config {
	err := godotenv.Load()
	if err != nil {
		log.Println("Warning: .env file not found, using environment variables")
	}

	jwtExpireHours, _ := strconv.Atoi(getEnv("JWT_EXPIRE_HOURS", "24"))
	refreshTokenExpireDays, _ := strconv.Atoi(getEnv("REFRESH_TOKEN_EXPIRE_DAYS", "28"))

	return &Config{
		DBHost:                 getEnv("DB_HOST", "localhost"),
		DBPort:                 getEnv("DB_PORT", "5432"),
		DBUser:                 getEnv("DB_USER", "Nuxx"),
		DBPassword:             getEnv("DB_PASSWORD", "gajahku"),
		DBName:                 getEnv("DB_NAME", "livo-master"),
		DBSSLMode:              getEnv("DB_SSLMODE", "disable"),
		JWTSecret:              getEnv("JWT_SECRET", "your-secret-key"),
		JWTExpireHours:         jwtExpireHours,
		RefreshTokenExpireDays: refreshTokenExpireDays,
		Port:                   getEnv("SERVER_PORT", "8081"),
		GinMode:                getEnv("GIN_MODE", "debug"),
		CORSAllowedOrigins:     getEnv("CORS_ALLOWED_ORIGINS", "*"),
		CORSAllowedMethods:     getEnv("CORS_ALLOWED_METHODS", "GET,POST,PUT,DELETE,OPTIONS"),
		APIHost:                getEnv("API_HOST", "localhost"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
