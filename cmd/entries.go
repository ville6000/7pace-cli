package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"time"

	"github.com/ville6000/7pace-cli/internal/api"
)

// timeEntry is one time entry as printed by `toggl-cli history --json`.
type timeEntry struct {
	ID int `json:"id"`
	// Start carries the offset of the timezone toggl-cli is configured for,
	// so its date is the local day the entry belongs to.
	Start time.Time `json:"start"`
	// Duration is in seconds; for a running entry, the time elapsed so far.
	Duration    int      `json:"duration"`
	Running     bool     `json:"running"`
	Description string   `json:"description"`
	Project     string   `json:"project"`
	Tags        []string `json:"tags"`
}

// readEntries decodes the JSON array of time entries in r.
func readEntries(r io.Reader) ([]timeEntry, error) {
	var entries []timeEntry
	if err := json.NewDecoder(r).Decode(&entries); err != nil {
		if errors.Is(err, io.EOF) {
			return nil, errors.New("no input: pipe in time entries, e.g. `toggl-cli history --json | 7pace-cli sync`")
		}
		return nil, fmt.Errorf("invalid input, want the JSON from `toggl-cli history --json`: %w", err)
	}

	return entries, nil
}

var (
	workItemHashRe    = regexp.MustCompile(`#(\d+)`)
	workItemLeadingRe = regexp.MustCompile(`^\s*(\d+)`)
)

// parseWorkItemID extracts an Azure DevOps work item id from a time entry
// description. It prefers a `#1234` style reference (e.g. `AB#1234`) and falls
// back to a leading number (e.g. `1234 - do stuff`). Returns false when no id
// can be found.
func parseWorkItemID(description string) (int, bool) {
	if m := workItemHashRe.FindStringSubmatch(description); m != nil {
		id, err := strconv.Atoi(m[1])
		if err == nil {
			return id, true
		}
	}
	if m := workItemLeadingRe.FindStringSubmatch(description); m != nil {
		id, err := strconv.Atoi(m[1])
		if err == nil {
			return id, true
		}
	}
	return 0, false
}

// roundUpToMinute rounds a duration in seconds up to the nearest whole minute.
// Non-positive durations return 0.
func roundUpToMinute(seconds int) int {
	if seconds <= 0 {
		return 0
	}
	return ((seconds + 59) / 60) * 60
}

// aggregateEntries combines time entries that share the same description on
// the same day into a single entry, so a worklog never spans days. Days are
// calendar days in each entry's own offset, the timezone toggl-cli printed it
// in. Durations are summed and rounded up to the nearest minute, and the
// earliest Start is kept. First-seen order is preserved. Running entries and
// entries with a non-positive Duration are skipped.
func aggregateEntries(entries []timeEntry) []timeEntry {
	type groupKey struct {
		date        string
		description string
	}

	order := make([]groupKey, 0, len(entries))
	groups := make(map[groupKey]*timeEntry, len(entries))

	for _, entry := range entries {
		if entry.Running || entry.Duration <= 0 {
			continue
		}

		key := groupKey{
			date:        entry.Start.Format(time.DateOnly),
			description: entry.Description,
		}

		if g, ok := groups[key]; ok {
			g.Duration += entry.Duration
			if entry.Start.Before(g.Start) {
				g.Start = entry.Start
			}
			continue
		}

		combined := entry
		groups[key] = &combined
		order = append(order, key)
	}

	result := make([]timeEntry, 0, len(order))
	for _, key := range order {
		g := groups[key]
		g.Duration = roundUpToMinute(g.Duration)
		result = append(result, *g)
	}

	return result
}

// toWorkLog maps a time entry to a 7pace worklog, dated in the entry's own
// offset. The bool return reports whether a work item id was found in the
// description.
func toWorkLog(entry timeEntry, activityTypeID string) (api.WorkLog, bool) {
	id, ok := parseWorkItemID(entry.Description)

	workLog := api.WorkLog{
		Timestamp: entry.Start.Format(time.RFC3339),
		Length:    entry.Duration,
		Comment:   entry.Description,
	}

	if ok {
		workLog.WorkItemID = &id
	}

	if activityTypeID != "" {
		workLog.ActivityType = &api.ActivityRef{ID: activityTypeID}
	}

	return workLog, ok
}
