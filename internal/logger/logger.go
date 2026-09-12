package logger

import (
	"os"
	"time"

	"github.com/rs/zerolog"
)

func NewLogger() zerolog.Logger {
	output := zerolog.ConsoleWriter{
		Out:        os.Stdout,
		TimeFormat: time.RFC3339,
	}
	return zerolog.New(output).With().Timestamp().Logger()
}

func NewSuspectLogger() zerolog.Logger {
	// Create logs directory if it doesn't exist
	if err := os.MkdirAll("logs", 0755); err != nil {
		// fallback to stdout if we can't create directory
		return NewLogger()
	}
	
	file, err := os.OpenFile("logs/suspect.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		// fallback to stdout
		return NewLogger()
	}
	
	return zerolog.New(file).With().Timestamp().Logger()
}
