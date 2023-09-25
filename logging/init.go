package logging

import (
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/pkgerrors"
)

// init will instantiate the global logger and set up some global parameters from the zerolog package.
func init() {
	// Instantiate the global logger
	GlobalLogger = NewLogger(zerolog.Disabled)

	// Setup stack trace support and set the timestamp format to UNIX
	zerolog.ErrorStackMarshaler = pkgerrors.MarshalStack
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
}
