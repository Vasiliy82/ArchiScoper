package tracing

import "time"

type TraceConfig struct {
	ExporterURL string
	SampleRate  float64
	Timeout     time.Duration
}

type AppInfo struct {
	Environment       string
	DomainName        string
	ServiceName       string
	ServiceVersion    string
	ServiceInstanceID string
}
