package pkg

import (
	"math/rand"
	"time"

	"github.com/rs/zerolog/log"
)

func retry(attempts int, sleep time.Duration, action string, f func() error) error {
	if err := f(); err != nil {
		if s, ok := err.(stop); ok {
			return s.error
		}

		if attempts--; attempts > 0 {
			log.Debug().Msgf("Attempt number %d to %s...", attempts, action)
			jitter := time.Duration(rand.Int63n(int64(sleep)))
			sleep = sleep + jitter/2

			time.Sleep(sleep)
			return retry(attempts, 2*sleep, action, f)
		}
		return err
	}
	return nil
}

type stop struct {
	error
}
