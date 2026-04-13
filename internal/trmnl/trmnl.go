package trmnl

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"time"

	"wastetrmnl/internal/config"
	"wastetrmnl/internal/schedule"
	"wastetrmnl/internal/timeutil"
)

const endpointBase = "https://trmnl.com/api/custom_plugins/"

// Entry is a date/hour pair for an optional szop/szot event.
type Entry struct {
	Hour         string `json:"hour"`
	DateReadable string `json:"dateReadable"`
}

type scheduleEntry struct {
	Type         string `json:"type"`
	DateReadable string `json:"date-readable"`
	Date         string `json:"date"`
}

type todayInfo struct {
	Name string `json:"name"`
	Date string `json:"date"`
}

type payload struct {
	Today     todayInfo                 `json:"today"`
	Tomorrow  []schedule.TomorrowPickup `json:"tomorrow"`
	Schedules []scheduleEntry           `json:"schedules"`
	Szop      *Entry                    `json:"szop,omitempty"`
	Szot      *Entry                    `json:"szot,omitempty"`
}

// Send builds the TRMNL payload from dashboard data and optional config, then
// POSTs it to the TRMNL custom-plugin endpoint derived from opts.TrmnlPluginUUID.
func Send(dashboard schedule.DashboardResponse, opts *config.Options, loc *time.Location) error {
	p := buildPayload(dashboard, opts, loc)
	body, err := json.Marshal(map[string]any{"merge_variables": p})
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}

	endpoint := endpointBase + opts.TrmnlPluginUUID
	req, err := http.NewRequest("POST", endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("non-2xx status: %d", resp.StatusCode)
	}
	return nil
}

// buildPayload converts a DashboardResponse + Options into the wire payload.
func buildPayload(dashboard schedule.DashboardResponse, opts *config.Options, loc *time.Location) payload {
	now := time.Now().In(loc)

	raw := map[string][]string{
		"tworzywa":  dashboard.Schedule.Tworzywa,
		"papier":    dashboard.Schedule.Papier,
		"szklo":     dashboard.Schedule.Szklo,
		"zmieszane": dashboard.Schedule.Zmieszane,
	}

	var schedules []scheduleEntry
	for typ, dates := range raw {
		if len(dates) > 0 {
			schedules = append(schedules, scheduleEntry{
				Type:         typ,
				DateReadable: timeutil.RelativeDays(dates[0], loc),
				Date:         dates[0],
			})
		} else {
			schedules = append(schedules, scheduleEntry{
				Type:         typ,
				DateReadable: "?",
				Date:         "?",
			})
		}
	}
	sort.Slice(schedules, func(i, j int) bool {
		return schedules[i].Date < schedules[j].Date
	})

	p := payload{
		Today: todayInfo{
			Name: timeutil.WeekdayNames[now.Weekday()],
			Date: now.Format("2006-01-02"),
		},
		Tomorrow:  dashboard.Tomorrow,
		Schedules: schedules,
	}

	if opts != nil {
		p.Szop = firstEntry(opts.Szop, loc)
		p.Szot = firstEntry(opts.Szot, loc)
	}

	return p
}

// firstEntry picks the nearest future entry from a list of config.Entry values.
func firstEntry(entries []config.Entry, loc *time.Location) *Entry {
	if len(entries) == 0 {
		return &Entry{DateReadable: "?"}
	}

	now := time.Now().In(loc)
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)

	var future []config.Entry
	for _, e := range entries {
		d, err := timeutil.ParseDate(e.Date, loc)
		if err != nil {
			continue
		}
		if !d.Before(today) {
			future = append(future, e)
		}
	}

	if len(future) == 0 {
		return &Entry{DateReadable: "?"}
	}

	nearest := future[0]
	nearestTime, err := timeutil.ParseDate(nearest.Date, loc)
	if err == nil {
		for _, e := range future[1:] {
			t, err := timeutil.ParseDate(e.Date, loc)
			if err != nil {
				continue
			}
			if t.Before(nearestTime) {
				nearest = e
				nearestTime = t
			}
		}
	}

	return &Entry{
		Hour:         nearest.Hour,
		DateReadable: timeutil.RelativeDays(nearest.Date, loc),
	}
}
