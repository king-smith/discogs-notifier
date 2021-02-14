package main

import (
	"os"

	"github.com/joho/godotenv"
	notifier "github.com/king-smith/discogs-notifier"
	log "github.com/sirupsen/logrus"
)

func main() {

	log.SetFormatter(&log.JSONFormatter{})

	// Check .env exists
	if _, err := os.Stat(".env"); err == nil {
		err := godotenv.Load(".env")
		if err != nil {
			log.Fatal(err)
		}

	} else if !os.IsNotExist(err) {
		log.Fatal(err)
	}

	if os.Getenv("VERBOSE") == "true" {
		log.SetLevel(log.DebugLevel)
	}

	log.Info("Starting notifier")

	if err := notifier.RunNotifier(); err != nil {
		log.Errorf("Notifier failed: %v", err)
	}
}
