package cmd

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ville6000/7pace-cli/internal/api"
)

// syncInputJSON is `toggl-cli history --json` output for the sync tests: two
// entries that share a description and carry a work item id, one without an
// id, and a running one.
const syncInputJSON = `[
  {"id": 1, "start": "2024-03-04T10:00:00+09:00", "duration": 1500, "running": false, "description": "#1234 review", "project": "Alpha", "tags": []},
  {"id": 2, "start": "2024-03-04T14:00:00+09:00", "duration": 100, "running": false, "description": "#1234 review", "project": "Alpha", "tags": []},
  {"id": 3, "start": "2024-03-04T15:00:00+09:00", "duration": 900, "running": false, "description": "no work item", "project": "Alpha", "tags": []},
  {"id": 4, "start": "2024-03-04T16:00:00+09:00", "duration": 600, "running": true, "description": "#99 still going", "project": "Alpha", "tags": []}
]`

func TestSync_DryRunPostsNothing(t *testing.T) {
	stub := newAPIStub(t)
	v := setupCLITest(t, stub)

	out, _, err := executeCommandWithInput(t, v, syncInputJSON, "sync", "--dry-run")
	if err != nil {
		t.Fatalf("sync --dry-run: %v", err)
	}

	for _, want := range []string{
		"Worklogs to post",
		"1234",
		"Skipped (no work item id)",
		"no work item",
		"Dry run: 1 worklog(s) (00:27:00) would be posted, 1 skipped.",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "still going") {
		t.Errorf("running entry listed:\n%s", out)
	}

	if posted := stub.requestsFor(http.MethodPost, workLogPath); len(posted) != 0 {
		t.Errorf("--dry-run posted %d worklog(s), want 0", len(posted))
	}
}

// Each preview table footers its Duration column, so the time about to be
// logged to 7pace is visible before confirming the post.
func TestSync_TablesTotalTheDurationColumn(t *testing.T) {
	v := setupCLITest(t, newAPIStub(t))

	out, _, err := executeCommandWithInput(t, v, syncInputJSON, "sync", "--dry-run")
	if err != nil {
		t.Fatalf("sync --dry-run: %v", err)
	}

	planned, skipped, found := strings.Cut(out, "Skipped (no work item id)")
	if !found {
		t.Fatalf("output missing the skipped table:\n%s", out)
	}

	// The single row plus the footer: 1500 + 100 seconds, rounded up to the
	// next whole minute.
	if got := strings.Count(planned, "00:27:00"); got != 2 {
		t.Errorf("worklogs table has %d occurrences of 00:27:00, want 2 (row + total):\n%s", got, planned)
	}
	if got := strings.Count(skipped, "00:15:00"); got != 2 {
		t.Errorf("skipped table has %d occurrences of 00:15:00, want 2 (row + total):\n%s", got, skipped)
	}
}

func TestSync_AggregatesEntriesSharingADescription(t *testing.T) {
	stub := newAPIStub(t)
	v := setupCLITest(t, stub)
	stub.respondToWorkLogs(http.StatusOK, api.WorkLog{})

	out, _, err := executeCommandWithInput(t, v, syncInputJSON, "sync", "--yes")
	if err != nil {
		t.Fatalf("sync: %v", err)
	}

	var posted api.WorkLog
	stub.onlyRequestFor(http.MethodPost, workLogPath).decodeBody(t, &posted)

	if posted.WorkItemID == nil || *posted.WorkItemID != 1234 {
		t.Errorf("work item id: got %v, want 1234", posted.WorkItemID)
	}
	// 1500 + 100 seconds, rounded up to the next whole minute.
	if want := 1620; posted.Length != want {
		t.Errorf("length: got %d, want %d", posted.Length, want)
	}
	// The earliest start of the group, in the input's own offset.
	if want := "2024-03-04T10:00:00+09:00"; posted.Timestamp != want {
		t.Errorf("timestamp: got %q, want %q", posted.Timestamp, want)
	}
	if posted.Comment != "#1234 review" {
		t.Errorf("comment: got %q, want %q", posted.Comment, "#1234 review")
	}
	if posted.ActivityType == nil || posted.ActivityType.ID != "activity-uuid" {
		t.Errorf("activity type: got %v, want the configured uuid", posted.ActivityType)
	}

	if want := "Posted 1 worklog(s) (00:27:00), 1 skipped, 0 failed."; !strings.Contains(out, want) {
		t.Errorf("output missing %q:\n%s", want, out)
	}
}

// With the entries piped in on stdin, the answer to the prompt comes from the
// terminal.
func TestSync_PipedInputAsksTheTerminal(t *testing.T) {
	stub := newAPIStub(t)
	v := setupCLITest(t, stub)
	stub.respondToWorkLogs(http.StatusOK, api.WorkLog{})
	opened := stubTerminal(t, "y\n")

	out, _, err := executeCommandWithInput(t, v, syncInputJSON, "sync")
	if err != nil {
		t.Fatalf("sync: %v", err)
	}

	if !*opened {
		t.Error("the confirmation was not read from the terminal")
	}
	if !strings.Contains(out, "Post 1 worklog(s) (00:27:00) to 7pace? [y/N]: ") {
		t.Errorf("output missing the confirmation prompt:\n%s", out)
	}
	if posted := stub.requestsFor(http.MethodPost, workLogPath); len(posted) != 1 {
		t.Errorf("confirmed sync posted %d worklog(s), want 1", len(posted))
	}
}

func TestSync_AbortsWhenConfirmationIsDeclined(t *testing.T) {
	stub := newAPIStub(t)
	v := setupCLITest(t, stub)
	stubTerminal(t, "n\n")

	out, _, err := executeCommandWithInput(t, v, syncInputJSON, "sync")
	if err != nil {
		t.Fatalf("sync: %v", err)
	}

	if !strings.Contains(out, "Aborted.") {
		t.Errorf("output missing the abort notice:\n%s", out)
	}
	if posted := stub.requestsFor(http.MethodPost, workLogPath); len(posted) != 0 {
		t.Errorf("declined sync posted %d worklog(s), want 0", len(posted))
	}
}

// Without a terminal to ask (cron, CI), sync refuses to post rather than
// guessing, and points at --yes.
func TestSync_WithoutATerminalRequiresYes(t *testing.T) {
	stub := newAPIStub(t)
	v := setupCLITest(t, stub)

	orig := openTerminal
	t.Cleanup(func() { openTerminal = orig })
	openTerminal = func() (io.ReadCloser, error) { return nil, os.ErrNotExist }

	_, _, err := executeCommandWithInput(t, v, syncInputJSON, "sync")
	if err == nil || !strings.Contains(err.Error(), "--yes") {
		t.Fatalf("expected an error pointing at --yes, got %v", err)
	}
	if posted := stub.requestsFor(http.MethodPost, workLogPath); len(posted) != 0 {
		t.Errorf("unconfirmed sync posted %d worklog(s), want 0", len(posted))
	}
}

// With a file argument, stdin is free, so the prompt reads its answer there.
func TestSync_ReadsAFileAndAsksOnStdin(t *testing.T) {
	stub := newAPIStub(t)
	v := setupCLITest(t, stub)
	stub.respondToWorkLogs(http.StatusOK, api.WorkLog{})
	opened := stubTerminal(t, "")

	path := filepath.Join(t.TempDir(), "entries.json")
	if err := os.WriteFile(path, []byte(syncInputJSON), 0o600); err != nil {
		t.Fatalf("write input: %v", err)
	}

	if _, _, err := executeCommandWithInput(t, v, "y\n", "sync", path); err != nil {
		t.Fatalf("sync: %v", err)
	}

	if *opened {
		t.Error("the terminal was opened although stdin was free")
	}
	if posted := stub.requestsFor(http.MethodPost, workLogPath); len(posted) != 1 {
		t.Errorf("confirmed sync posted %d worklog(s), want 1", len(posted))
	}
}

// The same task worked on two days is posted as one worklog per day, each
// dated on its own day, not merged into the first day.
func TestSync_PostsOneWorklogPerDay(t *testing.T) {
	stub := newAPIStub(t)
	v := setupCLITest(t, stub)
	stub.respondToWorkLogs(http.StatusOK, api.WorkLog{})

	input := `[
	  {"id": 1, "start": "2024-03-04T10:00:00+09:00", "duration": 3600, "description": "#1234 review"},
	  {"id": 2, "start": "2024-03-05T10:00:00+09:00", "duration": 7200, "description": "#1234 review"}
	]`
	if _, _, err := executeCommandWithInput(t, v, input, "sync", "--yes"); err != nil {
		t.Fatalf("sync: %v", err)
	}

	posted := stub.requestsFor(http.MethodPost, workLogPath)
	if len(posted) != 2 {
		t.Fatalf("posted %d worklog(s), want one per day (2)", len(posted))
	}

	want := []struct {
		date   string
		length int
	}{
		{"2024-03-04", 3600},
		{"2024-03-05", 7200},
	}
	for i, req := range posted {
		var workLog api.WorkLog
		req.decodeBody(t, &workLog)

		if !strings.HasPrefix(workLog.Timestamp, want[i].date) {
			t.Errorf("worklog %d timestamp %q, want it dated %s", i, workLog.Timestamp, want[i].date)
		}
		if workLog.Length != want[i].length {
			t.Errorf("worklog %d length %d, want %d", i, workLog.Length, want[i].length)
		}
	}
}

func TestSync_ReportsFailedWorklogs(t *testing.T) {
	stub := newAPIStub(t)
	v := setupCLITest(t, stub)
	stub.respondToWorkLogs(http.StatusInternalServerError, nil)

	out, _, err := executeCommandWithInput(t, v, syncInputJSON, "sync", "--yes")
	if err == nil {
		t.Fatal("expected an error when a worklog fails to post")
	}
	if !strings.Contains(err.Error(), "1 worklog(s) failed to post") {
		t.Errorf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "Posted 0 worklog(s) (00:00:00), 1 skipped, 1 failed.") {
		t.Errorf("output missing the failure summary:\n%s", out)
	}
}

func TestSync_EmptyInputIsAnError(t *testing.T) {
	v := setupCLITest(t, newAPIStub(t))

	_, _, err := executeCommandWithInput(t, v, "[]\n", "sync", "--dry-run")
	if err == nil || !strings.Contains(err.Error(), "no finished time entries") {
		t.Errorf("expected a no-entries error, got %v", err)
	}
}

func TestAdd_PostsASingleWorklog(t *testing.T) {
	stub := newAPIStub(t)
	v := setupCLITest(t, stub)
	stub.respondToWorkLogs(http.StatusOK, api.WorkLog{})

	out, _, err := executeCommand(t, v,
		"add",
		"--work-item", "99",
		"--comment", "manual entry",
		"--duration", "1h30m",
		"--date", "2024-03-04 13:15",
	)
	if err != nil {
		t.Fatalf("add: %v", err)
	}

	var posted api.WorkLog
	stub.onlyRequestFor(http.MethodPost, workLogPath).decodeBody(t, &posted)

	if posted.WorkItemID == nil || *posted.WorkItemID != 99 {
		t.Errorf("work item id: got %v, want 99", posted.WorkItemID)
	}
	if want := 5400; posted.Length != want {
		t.Errorf("length: got %d, want %d", posted.Length, want)
	}
	local, _ := time.ParseInLocation("2006-01-02 15:04", "2024-03-04 13:15", time.Local)
	if want := local.Format(time.RFC3339); posted.Timestamp != want {
		t.Errorf("timestamp: got %q, want %q", posted.Timestamp, want)
	}
	if !strings.Contains(out, "Posted worklog: 01:30:00 for 2024-03-04 13:15") {
		t.Errorf("unexpected output:\n%s", out)
	}
}

func TestAdd_RequiresWorkItemOrComment(t *testing.T) {
	stub := newAPIStub(t)
	v := setupCLITest(t, stub)

	_, _, err := executeCommand(t, v, "add", "--duration", "30m")
	if err == nil {
		t.Fatal("expected an error without --work-item or --comment")
	}
	if !strings.Contains(err.Error(), "either --work-item or --comment") {
		t.Errorf("unexpected error: %v", err)
	}
	if posted := stub.requestsFor(http.MethodPost, workLogPath); len(posted) != 0 {
		t.Errorf("invalid worklog posted %d time(s), want 0", len(posted))
	}
}

func TestConfigCommand_WritesTheConfigFile(t *testing.T) {
	v := setupCLITest(t, nil)

	input := strings.Join([]string{
		"https://7pace.example",
		"CORP",
		"jdoe",
		"hunter2",
		"activity-uuid",
		"",
	}, "\n")

	out, _, err := executeCommandWithInput(t, v, input, "config")
	if err != nil {
		t.Fatalf("config: %v", err)
	}
	if !strings.Contains(out, "Configuration saved successfully!") {
		t.Errorf("unexpected output:\n%s", out)
	}

	configPath := filepath.Join(os.Getenv("XDG_CONFIG_HOME"), "7pace-cli", "config.yaml")
	written, err := os.ReadFile(configPath) // #nosec G304 - path built from the test's own temp dir
	if err != nil {
		t.Fatalf("read written config: %v", err)
	}
	for _, want := range []string{"https://7pace.example", "CORP", "jdoe", "hunter2", "activity-uuid"} {
		if !strings.Contains(string(written), want) {
			t.Errorf("config file missing %q:\n%s", want, written)
		}
	}

	info, err := os.Stat(configPath)
	if err != nil {
		t.Fatalf("stat config: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("config file permissions %o, want 600", perm)
	}
}

func TestConfigCommand_RequiresTheBaseURL(t *testing.T) {
	v := setupCLITest(t, nil)

	if _, _, err := executeCommandWithInput(t, v, "\n", "config"); err == nil {
		t.Error("expected an error for an empty base URL")
	}
}
