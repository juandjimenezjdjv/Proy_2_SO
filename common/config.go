package common

import (
	"os"
	"strconv"
)

// Constantes del sistema
const (
	MAX_RETRIES = 3 // Número máximo de reintentos para tareas fallidas
)

// Config contiene la configuración compartida del sistema
type Config struct {
	// Configuración del Master
	MasterHost          string
	MasterPort          int
	HeartbeatSec        int
	HeartbeatTimeoutSec int

	// Configuración de Workers
	WorkerThreads int
	MaxRetries    int

	// Configuración de almacenamiento
	DataDir    string
	ResultsDir string

	// Configuración de logging
	LogLevel LogLevel
}

// LoadConfig carga la configuración desde variables de entorno con valores por defecto
func LoadConfig() *Config {
	return &Config{
		MasterHost:          getEnv("MASTER_HOST", "localhost"),
		MasterPort:          getEnvInt("MASTER_PORT", 8080),
		HeartbeatSec:        getEnvInt("HEARTBEAT_SEC", 2),
		HeartbeatTimeoutSec: getEnvInt("HEARTBEAT_TIMEOUT_SEC", 10),
		WorkerThreads:       getEnvInt("WORKER_THREADS", 4),
		MaxRetries:          getEnvInt("MAX_RETRIES", 3),
		DataDir:             getEnv("DATA_DIR", "./app/data"),
		ResultsDir:          getEnv("RESULTS_DIR", "./app/results"),
		LogLevel:            LogLevel(getEnv("LOG_LEVEL", "INFO")),
	}
}

// getEnv obtiene una variable de entorno o retorna un valor por defecto
func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

// getEnvInt obtiene una variable de entorno como entero o retorna un valor por defecto
func getEnvInt(key string, defaultValue int) int {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	intValue, err := strconv.Atoi(value)
	if err != nil {
		return defaultValue
	}
	return intValue
}
