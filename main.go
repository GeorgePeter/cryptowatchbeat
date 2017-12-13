package main

import (
	"os"

	"github.com/elastic/beats/libbeat/beat"

	"github.com/GeorgePeter/cryptowatchbeat/beater"
)

func main() {
	err := beat.Run("cryptowatchbeat", "", beater.New())
	if err != nil {
		os.Exit(1)
	}
}
