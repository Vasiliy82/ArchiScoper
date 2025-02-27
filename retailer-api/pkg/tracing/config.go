package tracing

import "time"

type Config struct {
	ServiceName string
	ExporterURL string
	SampleRate  float64
	Environment string
	Timeout     time.Duration
}
