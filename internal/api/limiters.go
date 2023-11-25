package api

import (
	"time"

	"github.com/rs/zerolog/log"
	"golang.org/x/time/rate"
)

var (
	redactedLimiter *rate.Limiter
	orpheusLimiter  *rate.Limiter
)

func init() {
	redactedLimiter = rate.NewLimiter(rate.Every(10*time.Second), 10)
	orpheusLimiter = rate.NewLimiter(rate.Every(10*time.Second), 5)
}

// returns a rate limiter based on the provided indexer string.
func getLimiter(indexer string) *rate.Limiter {
	switch indexer {
	case "redacted":
		return redactedLimiter
	case "ops":
		return orpheusLimiter
	default:
		log.Error().Msgf("Invalid indexer: %s", indexer)
		return nil
	}
}
