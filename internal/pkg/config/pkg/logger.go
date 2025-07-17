package pkg

// Logger defines configuration options for the application's logging system.
type Logger struct {
	// Level sets the minimum log level to output.
	//
	// Accepted values: "DEBUG", "INFO", "WARN", "ERROR", "PANIC", "FATAL".
	// Default: "INFO".
	Level string `default:"INFO" validate:"oneof=DEBUG INFO WARN ERROR PANIC FATAL"`
}
