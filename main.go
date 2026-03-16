package main

import (
	"log"
	"time"
	_ "time/tzdata"

	"wastetrmnl/internal/config"
	"wastetrmnl/internal/schedule"
	"wastetrmnl/internal/timeutil"
	"wastetrmnl/internal/trmnl"
)

const (
	maxRetries = 3
	retryDelay = 30 * time.Second
)

var loc *time.Location

func run(opts *config.Options) error {
	s, err := schedule.Fetch(opts)
	if err != nil {
		return err
	}

	dashboard := schedule.BuildDashboard(*s, loc)

	for attempt := 1; attempt <= maxRetries; attempt++ {
		err = trmnl.Send(dashboard, opts, loc)
		if err == nil {
			log.Printf("successfully sent to TRMNL on attempt %d", attempt)
			return nil
		}
		log.Printf("attempt %d/%d failed: %v", attempt, maxRetries, err)
		if attempt < maxRetries {
			log.Printf("retrying in %s...", retryDelay)
			time.Sleep(retryDelay)
		}
	}

	return err
}

func main() {
	var err error
	loc, err = time.LoadLocation("Europe/Warsaw")
	if err != nil {
		log.Fatalf("failed to load timezone: %v", err)
	}

	opts, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load options: %v", err)
	}
	for _, e := range opts.Szop {
		log.Printf("Szop - date: %s, hour: %s", e.Date, e.Hour)
	}
	for _, e := range opts.Szot {
		log.Printf("Szot - date: %s, hour: %s", e.Date, e.Hour)
	}

	log.Println("running initial send...")
	if err := run(opts); err != nil {
		log.Printf("initial run failed: %v", err)
	}

	for {
		wait := timeutil.DurationUntilNextRun(loc)
		log.Printf("next run scheduled in %s (at 00:05)", wait.Round(time.Second))
		time.Sleep(wait)

		log.Println("running scheduled send...")
		if err := run(opts); err != nil {
			log.Printf("scheduled run failed (all retries exhausted): %v", err)
		}
	}
}
