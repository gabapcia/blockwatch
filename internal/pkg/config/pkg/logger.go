package pkg

type Logger struct {
	Level string `default:"INFO" validate:"oneof=DEBUG INFO WARN ERROR PANIC FATAL"`
}
