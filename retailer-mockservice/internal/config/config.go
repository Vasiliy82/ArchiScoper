package config

import (
	"math/rand"
	"os"
	"strconv"
)

type EndpointConfig struct {
	Method       string
	Name         string
	ResponseBody string
	DurationMin  int
	DurationRnd  int
	Error400Prob float64
	Error500Prob float64
}

type Config struct {
	DomainName        string
	ServiceName       string
	ServiceVersion    string
	ListenAddress     string
	ExporterURL       string
	ServiceInstanceID string
	Endpoint          EndpointConfig
	ReverseEndpoint   EndpointConfig
}

func LoadConfig() Config {
	return Config{
		DomainName:        getEnv("DOMAIN_NAME", "default_domain"),
		ServiceName:       getEnv("SERVICE_NAME", "mock_service"),
		ServiceVersion:    getEnv("SERVICE_VERSION", "1.0.0"),
		ExporterURL:       getEnv("OTEL_EXPORTER_URL", "http://localhost:4317"),
		ListenAddress:     getEnv("LISTEN_ADDRESS", ":8080"),
		ServiceInstanceID: getEnv("SERVICE_INSTANCE_ID", generateInstanceID()),
		Endpoint: EndpointConfig{
			Method:       getEnv("ENDPOINT_METHOD", "GET"),
			Name:         getEnv("ENDPOINT_NAME", "test"),
			ResponseBody: getEnv("RESPONSE_BODY", `{"message":"ok"}`),
			DurationMin:  getEnvAsInt("DURATION_MIN", 200),
			DurationRnd:  getEnvAsInt("DURATION_MAX", 500) - getEnvAsInt("DURATION_MIN", 200),
			Error400Prob: getEnvAsFloat("ERROR_400_PROBABILITY", 0.1),
			Error500Prob: getEnvAsFloat("ERROR_500_PROBABILITY", 0.05),
		},
		ReverseEndpoint: EndpointConfig{
			Method:       getEnv("REVERSE_ENDPOINT_METHOD", "GET"),
			Name:         getEnv("REVERSE_ENDPOINT_NAME", "test"),
			ResponseBody: getEnv("REVERSE_RESPONSE_BODY", `{"message":"ok"}`),
			DurationMin:  getEnvAsInt("REVERSE_DURATION_MIN", 200),
			DurationRnd:  getEnvAsInt("REVERSE_DURATION_MAX", 500) - getEnvAsInt("REVERSE_DURATION_MIN", 200),
			Error400Prob: getEnvAsFloat("REVERSE_ERROR_400_PROBABILITY", 0),
			Error500Prob: getEnvAsFloat("REVERSE_ERROR_500_PROBABILITY", 0),
		},
	}
}

func generateInstanceID() string {
	return strconv.Itoa(rand.Intn(100000))
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

func getEnvAsFloat(key string, defaultValue float64) float64 {
	valueStr := getEnv(key, "")
	if value, err := strconv.ParseFloat(valueStr, 64); err == nil {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int64) int {
	valueStr := getEnv(key, "")
	if value, err := strconv.ParseInt(valueStr, 0, 32); err == nil {
		return int(value)
	}
	return int(defaultValue)
}
