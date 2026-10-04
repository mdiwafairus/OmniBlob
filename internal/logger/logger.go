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
	writer := NewDailyWriter("logs", "suspect")
	return zerolog.New(writer).With().Timestamp().Logger()
}
