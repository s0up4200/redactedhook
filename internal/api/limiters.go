package api

import (
	"fmt"
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

func getLimiter(indexer string) (*rate.Limiter, error) {
	switch indexer {
	case "redacted":
		return redactedLimiter, nil
	case "ops":
		return orpheusLimiter, nil
	default:
		err := fmt.Errorf("invalid indexer: %s", indexer)
		log.Error().Err(err).Msg("Failed to get rate limiter")
		return nil, err
	}
}
