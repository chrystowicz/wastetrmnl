package schedule

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
	"wastetrmnl/internal/config"

	"golang.org/x/net/html"

	"wastetrmnl/internal/timeutil"
)

// Schedule holds the upcoming collection dates for each waste category.
type Schedule struct {
	Tworzywa  []string `json:"tworzywa"`
	Papier    []string `json:"papier"`
	Szklo     []string `json:"szklo"`
	Zmieszane []string `json:"zmieszane"`
}

// TomorrowPickup describes a single collection event happening tomorrow.
type TomorrowPickup struct {
	Date string `json:"date"`
	Type string `json:"type"`
}

// DashboardResponse is the top-level payload used by other packages.
type DashboardResponse struct {
	Today    string           `json:"today"`
	Schedule Schedule         `json:"schedule"`
	Tomorrow []TomorrowPickup `json:"tomorrow"`
}

type rawResponse struct {
	WiadomoscRWD string `json:"wiadomoscRWD"`
}

const (
	scheduleURL = "https://ekosystem.wroc.pl/wp-admin/admin-ajax.php"
)

// Fetch retrieves the raw HTML schedule from ekosystem.wroc.pl.
func Fetch(opts *config.Options) (*Schedule, error) {
	formData := url.Values{
		"action":    {"waste_disposal_form_get_schedule_direct"},
		"id_numeru": {opts.IdNumeru},
		"id_ulicy":  {opts.IdUlicy},
	}

	req, err := http.NewRequest("POST", scheduleURL, strings.NewReader(formData.Encode()))
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	var raw rawResponse
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("decode JSON: %w", err)
	}

	s := Parse(raw.WiadomoscRWD)
	return &s, nil
}

// Parse extracts a Schedule from an HTML table string.
func Parse(htmlStr string) Schedule {
	doc, err := html.Parse(strings.NewReader(htmlStr))
	if err != nil {
		panic(fmt.Sprintf("schedule: failed to parse HTML: %v", err))
	}

	var headers []string
	var columns [][]string
	var colIndex int

	var walkHeaders func(*html.Node)
	walkHeaders = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "th" && n.FirstChild != nil {
			headers = append(headers, n.FirstChild.Data)
			columns = append(columns, []string{})
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walkHeaders(c)
		}
	}

	var walkRows func(*html.Node)
	walkRows = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "tr" {
			colIndex = 0
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				if c.Type == html.ElementNode && c.Data == "td" {
					if colIndex < len(columns) && c.FirstChild != nil {
						columns[colIndex] = append(columns[colIndex], c.FirstChild.Data)
					}
					colIndex++
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walkRows(c)
		}
	}

	walkHeaders(doc)
	walkRows(doc)

	var s Schedule
	for i, header := range headers {
		switch strings.ToLower(header) {
		case "tworzywa":
			s.Tworzywa = columns[i]
		case "papier":
			s.Papier = columns[i]
		case "szkło", "szklo":
			s.Szklo = columns[i]
		case "zmieszane":
			s.Zmieszane = columns[i]
		}
	}
	return s
}

// TomorrowPickups returns which categories are collected tomorrow.
// Always returns at least one entry; uses type "none" when nothing is scheduled.
func TomorrowPickups(s Schedule, loc *time.Location) []TomorrowPickup {
	tomorrow := time.Now().In(loc).AddDate(0, 0, 1).Format("2006-01-02")

	categories := map[string][]string{
		"tworzywa":  s.Tworzywa,
		"papier":    s.Papier,
		"szklo":     s.Szklo,
		"zmieszane": s.Zmieszane,
	}

	var pickups []TomorrowPickup
	for category, dates := range categories {
		for _, date := range dates {
			if date == tomorrow {
				pickups = append(pickups, TomorrowPickup{Date: date, Type: category})
				break
			}
		}
	}

	if len(pickups) == 0 {
		pickups = append(pickups, TomorrowPickup{Date: tomorrow, Type: "none"})
	}
	return pickups
}

// BuildDashboard assembles a DashboardResponse from a fetched schedule.
func BuildDashboard(s Schedule, loc *time.Location) DashboardResponse {
	return DashboardResponse{
		Today:    timeutil.TodayString(loc),
		Schedule: s,
		Tomorrow: TomorrowPickups(s, loc),
	}
}
