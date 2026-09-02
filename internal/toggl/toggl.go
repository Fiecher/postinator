package toggl

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image/color"
	"net/http"
	"postinator/internal/config"
	"sort"
	"strings"
	"time"
)

type StatItem struct {
	Label    string
	Duration string
	Color    color.RGBA
}

type ProjectSummary struct {
	UserID         int `json:"user_id"`
	ProjectID      int `json:"project_id"`
	TrackedSeconds int `json:"tracked_seconds"`
}

type ProjectInfo struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type Client struct {
	apiToken    string
	workspaceID int
	httpClient  *http.Client
	cache       map[int]string
}

func NewClient(token string, workspaceID int) *Client {
	return &Client{
		apiToken:    token,
		workspaceID: workspaceID,
		httpClient:  &http.Client{Timeout: 15 * time.Second},
		cache:       make(map[int]string),
	}
}

func (c *Client) GetStats(ctx context.Context, start, end time.Time, mappings []config.ProjectMapping, otherMapping config.ProjectMapping) ([]StatItem, error) {
	if err := c.refreshProjectCache(ctx); err != nil {
		return nil, fmt.Errorf("toggl cache refresh failed: %w", err)
	}

	rawData, err := c.fetchSummary(ctx, start, end)
	if err != nil {
		return nil, fmt.Errorf("toggl fetch summary failed: %w", err)
	}

	return c.aggregate(rawData, mappings, otherMapping), nil
}

func (c *Client) aggregate(rawData []ProjectSummary, mappings []config.ProjectMapping, otherCfg config.ProjectMapping) []StatItem {
	aggregated := make(map[string]int)
	colorMap := make(map[string]color.RGBA)
	nameToDisplay := make(map[string]string)

	for _, m := range mappings {
		rgba := ParseHexColor(m.Color)
		colorMap[m.DisplayName] = rgba
		for _, tn := range m.TogglNames {
			nameToDisplay[strings.ToLower(tn)] = m.DisplayName
		}
	}

	for _, d := range rawData {
		originalName := strings.ToLower(c.cache[d.ProjectID])
		if dispName, ok := nameToDisplay[originalName]; ok {
			aggregated[dispName] += d.TrackedSeconds
		} else {
			aggregated[originalName] += d.TrackedSeconds
			if _, exists := colorMap[originalName]; !exists {
				colorMap[originalName] = color.RGBA{R: 130, G: 130, B: 130, A: 255}
			}
		}
	}

	type entry struct {
		name string
		sec  int
	}
	var entries []entry
	for k, v := range aggregated {
		if v > 0 {
			entries = append(entries, entry{k, v})
		}
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].sec > entries[j].sec
	})

	var results []StatItem
	const limit = 6

	if len(entries) <= limit {
		for _, e := range entries {
			results = append(results, c.buildItem(e.name, e.sec, colorMap[e.name]))
		}
	} else {
		for i := 0; i < 5; i++ {
			results = append(results, c.buildItem(entries[i].name, entries[i].sec, colorMap[entries[i].name]))
		}
		otherSec := 0
		for i := 5; i < len(entries); i++ {
			otherSec += entries[i].sec
		}
		results = append(results, c.buildItem(otherCfg.DisplayName, otherSec, ParseHexColor(otherCfg.Color)))
	}

	return results
}

func (c *Client) fetchSummary(ctx context.Context, start, end time.Time) ([]ProjectSummary, error) {
	url := fmt.Sprintf("https://api.track.toggl.com/reports/api/v3/workspace/%d/projects/summary", c.workspaceID)
	payload := map[string]string{
		"start_date": start.Format("2006-01-02"),
		"end_date":   end.Format("2006-01-02"),
	}
	jsonData, _ := json.Marshal(payload)

	req, _ := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	req.SetBasicAuth(c.apiToken, "api_token")
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var data []ProjectSummary
	err = json.NewDecoder(resp.Body).Decode(&data)
	return data, err
}

func (c *Client) refreshProjectCache(ctx context.Context) error {
	url := fmt.Sprintf("https://api.track.toggl.com/api/v9/workspaces/%d/projects", c.workspaceID)
	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	req.SetBasicAuth(c.apiToken, "api_token")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var projects []ProjectInfo
	if err := json.NewDecoder(resp.Body).Decode(&projects); err != nil {
		return err
	}

	for _, p := range projects {
		c.cache[p.ID] = p.Name
	}
	return nil
}

func (c *Client) buildItem(name string, sec int, clr color.RGBA) StatItem {
	dur := time.Duration(sec) * time.Second
	return StatItem{
		Label:    name,
		Duration: fmt.Sprintf("%02d:%02d", int(dur.Hours()), int(dur.Minutes())%60),
		Color:    clr,
	}
}

func ParseHexColor(s string) color.RGBA {
	var r, g, b uint8
	if len(s) != 7 || s[0] != '#' {
		return color.RGBA{R: 128, G: 128, B: 128, A: 255}
	}
	fmt.Sscanf(s, "#%02x%02x%02x", &r, &g, &b)
	return color.RGBA{R: r, G: g, B: b, A: 255}
}

type seasonRange struct {
	startMonth time.Month
	endMonth   time.Month
	wraps      bool
}

func (c *Client) ParseDates(caption string) (time.Time, time.Time, error) {
	if strings.TrimSpace(caption) == "" {
		return time.Time{}, time.Time{}, fmt.Errorf("empty title")
	}

	months := map[string]time.Month{
		"ЯНВАРЬ": 1, "ФЕВРАЛЬ": 2, "МАРТ": 3, "АПРЕЛЬ": 4, "МАЙ": 5, "ИЮНЬ": 6,
		"ИЮЛЬ": 7, "АВГУСТ": 8, "СЕНТЯБРЬ": 9, "ОКТЯБРЬ": 10, "НОЯБРЬ": 11, "ДЕКАБРЬ": 12,
	}

	seasons := map[string]seasonRange{
		"ВЕСНА": {startMonth: 3, endMonth: 5},
		"ЛЕТО":  {startMonth: 6, endMonth: 8},
		"ОСЕНЬ": {startMonth: 9, endMonth: 11},
		"ЗИМА":  {startMonth: 12, endMonth: 2, wraps: true},
	}

	normalized := strings.ReplaceAll(strings.ToUpper(caption), "-", " - ")
	words := strings.Fields(normalized)
	now := time.Now()

	var foundMonths []time.Month
	var foundSeason string
	var targetYear int
	yearFound := false

	for _, word := range words {
		if word == "-" {
			continue
		}
		if m, ok := months[word]; ok {
			foundMonths = append(foundMonths, m)
			continue
		}
		if _, ok := seasons[word]; ok {
			foundSeason = word
			continue
		}

		var y int
		if n, _ := fmt.Sscanf(word, "%d", &y); n > 0 && y > 2000 {
			targetYear = y
			yearFound = true
		}
	}

	if !yearFound {
		targetYear = now.Year()
	}

	// Season, e.g. "ЛЕТО 2026" or "ЗИМА 2043".
	if foundSeason != "" {
		rng := seasons[foundSeason]
		startYear := targetYear
		if rng.wraps {
			startYear = targetYear - 1
		}
		start := time.Date(startYear, rng.startMonth, 1, 0, 0, 0, 0, time.UTC)
		end := time.Date(targetYear, rng.endMonth+1, 1, 0, 0, 0, 0, time.UTC).AddDate(0, 0, -1)
		return start, end, nil
	}

	if len(foundMonths) >= 2 {
		startMonth := foundMonths[0]
		endMonth := foundMonths[len(foundMonths)-1]
		endYear := targetYear
		if endMonth < startMonth {
			endYear = targetYear + 1
		}
		start := time.Date(targetYear, startMonth, 1, 0, 0, 0, 0, time.UTC)
		end := time.Date(endYear, endMonth+1, 1, 0, 0, 0, 0, time.UTC).AddDate(0, 0, -1)
		return start, end, nil
	}

	if len(foundMonths) == 1 {
		start := time.Date(targetYear, foundMonths[0], 1, 0, 0, 0, 0, time.UTC)
		end := start.AddDate(0, 1, -1)
		return start, end, nil
	}

	if yearFound {
		start := time.Date(targetYear, 1, 1, 0, 0, 0, 0, time.UTC)
		end := time.Date(targetYear, 12, 31, 23, 59, 59, 0, time.UTC)
		return start, end, nil
	}

	start := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 1, -1)
	return start, end, nil
}
