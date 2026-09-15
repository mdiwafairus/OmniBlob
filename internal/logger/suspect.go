package logger

import (
	"os"

	"github.com/rs/zerolog"
)

func NewSuspectLogger() zerolog.Logger {
	return zerolog.New(os.Stdout).With().Timestamp().Str("type", "suspect").Logger()
}

