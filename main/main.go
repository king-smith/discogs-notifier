package main

import (
	"github.com/joho/godotenv"
	notifier "github.com/king-smith/discogs-notifier"
	log "github.com/sirupsen/logrus"
)

func main() {

	log.SetFormatter(&log.JSONFormatter{})

	err := godotenv.Load(".env")
	if err != nil {
		log.Fatal(err)
	}

	log.Info("Starting notifier")

	if err := notifier.RunNotifier(); err != nil {
		log.Errorf("Notifier failed: %v", err)
	}
}
