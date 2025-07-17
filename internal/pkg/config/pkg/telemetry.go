package pkg

// Telemetry defines configuration settings related to observability and service identity.
type Telemetry struct {
	// ServiceName is the name used to identify this service in telemetry systems (e.g., logs, traces, metrics).
	// Default: "blockwatch"
	ServiceName string `default:"blockwatch"`
}
