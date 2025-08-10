package logutils

import (
	log "github.com/sirupsen/logrus"
)

// QuotesDisabled returns a new Logger that disables quotes.
func QuotesDisabled() *log.Logger {
	logger := log.New()
	logger.SetLevel(log.GetLevel())
	logger.SetFormatter(&log.TextFormatter{
		DisableQuote: true,
	})
	return logger
}
