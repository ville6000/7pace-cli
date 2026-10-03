package cmd

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"runtime"
	"strconv"
	"strings"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/ville6000/7pace-cli/internal/api"
	"github.com/ville6000/7pace-cli/internal/config"
	"github.com/ville6000/7pace-cli/internal/output"
)

// plannedWorkLog pairs a built 7pace payload with the display columns used in
// the preview / result tables.
type plannedWorkLog struct {
	workItem string
	started  string
	duration string
	comment  string
	payload  api.WorkLog
}

func newSyncCmd(v *viper.Viper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sync [file]",
		Short: "Post time entries from toggl-cli to 7pace as worklogs",
		Long: "Read time entries printed by `toggl-cli history --json`, from stdin or a file, and\n" +
			"post them to 7pace as worklogs. Entries sharing the same description on the same day\n" +
			"are combined into a single worklog, with their durations summed and rounded up to the\n" +
			"nearest minute. The work item id is parsed from the description (e.g. \"#1234\" or a\n" +
			"leading number); entries without a work item id, and running entries, are skipped.\n" +
			"There is no de-duplication, so posting the same entries twice creates duplicate\n" +
			"worklogs — use --dry-run first to preview.",
		Example: "  toggl-cli history --week --json | 7pace-cli sync --dry-run\n" +
			"  toggl-cli history --week --json | 7pace-cli sync\n" +
			"  toggl-cli history --week --json > week.json && 7pace-cli sync week.json",
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			dryRun, err := cmd.Flags().GetBool("dry-run")
			if err != nil {
				return fmt.Errorf("failed to get dry-run flag: %w", err)
			}

			assumeYes, err := cmd.Flags().GetBool("yes")
			if err != nil {
				return fmt.Errorf("failed to get yes flag: %w", err)
			}

			cfg, err := config.Load(v)
			if err != nil {
				return err
			}

			in, answers, err := syncInput(cmd, args)
			if err != nil {
				return err
			}
			defer func() { _ = in.Close() }()

			timeEntries, err := readEntries(in)
			if err != nil {
				return err
			}

			// Combine entries sharing a description on the same local day into a
			// single worklog, summing their durations and rounding up to the
			// nearest minute. Running or zero-length entries are dropped here.
			entries := aggregateEntries(timeEntries)

			var planned []plannedWorkLog
			var skipped [][]any
			plannedSeconds := 0
			skippedSeconds := 0
			for _, entry := range entries {
				workLog, ok := toWorkLog(entry, cfg.ActivityTypeID)
				started := entry.Start.Format("2006-01-02 15:04")
				duration := output.FormatDuration(entry.Duration)

				if !ok {
					skipped = append(skipped, []any{"—", started, duration, entry.Description})
					skippedSeconds += entry.Duration
					continue
				}

				planned = append(planned, plannedWorkLog{
					workItem: strconv.Itoa(*workLog.WorkItemID),
					started:  started,
					duration: duration,
					comment:  entry.Description,
					payload:  workLog,
				})
				plannedSeconds += workLog.Length
			}

			if len(planned) == 0 && len(skipped) == 0 {
				return errors.New("no finished time entries in the input")
			}

			out := cmd.OutOrStdout()
			headers := []any{"Work Item", "Started At", "Duration", "Comment"}
			if len(planned) > 0 {
				rows := make([][]any, 0, len(planned))
				for _, p := range planned {
					rows = append(rows, []any{p.workItem, p.started, p.duration, p.comment})
				}
				output.RenderTable(out, "Worklogs to post", headers, rows, totalFooter(plannedSeconds))
				fmt.Fprintln(out)
			}
			if len(skipped) > 0 {
				output.RenderTable(out, "Skipped (no work item id)", headers, skipped, totalFooter(skippedSeconds))
				fmt.Fprintln(out)
			}

			if dryRun {
				fmt.Fprintf(out, "Dry run: %d worklog(s) (%s) would be posted, %d skipped.\n",
					len(planned), output.FormatDuration(plannedSeconds), len(skipped))
				return nil
			}

			if len(planned) == 0 {
				fmt.Fprintln(out, "Nothing to post.")
				return nil
			}

			if !assumeYes {
				prompt := fmt.Sprintf("Post %d worklog(s) (%s) to 7pace?", len(planned), output.FormatDuration(plannedSeconds))
				confirmed, err := confirm(out, answers, prompt)
				if err != nil {
					return err
				}
				if !confirmed {
					fmt.Fprintln(out, "Aborted.")
					return nil
				}
			}

			client := api.NewClient(cfg)
			posted := 0
			postedSeconds := 0
			var failures [][]any
			for _, p := range planned {
				if _, postErr := client.CreateWorkLog(ctx, p.payload); postErr != nil {
					failures = append(failures, []any{p.workItem, p.started, p.duration, postErr.Error()})
					continue
				}
				posted++
				postedSeconds += p.payload.Length
			}

			fmt.Fprintf(out, "Posted %d worklog(s) (%s), %d skipped, %d failed.\n",
				posted, output.FormatDuration(postedSeconds), len(skipped), len(failures))
			if len(failures) > 0 {
				output.RenderTable(out, "Failed", []any{"Work Item", "Started At", "Duration", "Error"}, failures, nil)
				return fmt.Errorf("%d worklog(s) failed to post", len(failures))
			}

			return nil
		},
	}

	cmd.Flags().Bool("dry-run", false, "Preview the worklogs without posting")
	cmd.Flags().BoolP("yes", "y", false, "Skip the confirmation prompt")

	return cmd
}

// openTerminal opens the controlling terminal, where the confirmation prompt
// is answered when stdin carries the time entries. Replaced in tests.
var openTerminal = func() (io.ReadCloser, error) {
	name := "/dev/tty"
	if runtime.GOOS == "windows" {
		name = "CONIN$"
	}

	return os.Open(name)
}

// syncInput returns where sync reads the time entries from, and where the
// confirmation prompt reads its answer from. With a file argument the entries
// come from the file and the answer from stdin. Otherwise the entries come
// from stdin, so the answer is read from the terminal; when there is none, the
// returned answers reader yields an error explaining --yes.
func syncInput(cmd *cobra.Command, args []string) (entries io.ReadCloser, answers func() (io.ReadCloser, error), err error) {
	stdin := cmd.InOrStdin()

	if len(args) == 1 && args[0] != "-" {
		f, err := os.Open(args[0])
		if err != nil {
			return nil, nil, fmt.Errorf("failed to open input: %w", err)
		}
		return f, func() (io.ReadCloser, error) { return io.NopCloser(stdin), nil }, nil
	}

	// Nothing piped in: reading would wait for JSON typed by hand.
	if f, ok := stdin.(*os.File); ok && isTerminal(int(f.Fd())) {
		return nil, nil, errors.New("no input: pipe in time entries, e.g. `toggl-cli history --json | 7pace-cli sync`, or pass a file")
	}

	answers = func() (io.ReadCloser, error) {
		tty, err := openTerminal()
		if err != nil {
			return nil, fmt.Errorf("can't ask for confirmation without a terminal (%w); pass --yes to post, or --dry-run to preview", err)
		}
		return tty, nil
	}

	return io.NopCloser(stdin), answers, nil
}

// totalFooter builds the footer row summing the Duration column of the sync
// tables, so the time about to be logged to 7pace is visible at a glance.
func totalFooter(seconds int) table.Row {
	return table.Row{"", "Total", output.FormatDuration(seconds), ""}
}

// confirm asks prompt and reports whether the answer, read from answers, was
// yes.
func confirm(out io.Writer, answers func() (io.ReadCloser, error), prompt string) (bool, error) {
	in, err := answers()
	if err != nil {
		return false, err
	}
	defer func() { _ = in.Close() }()

	fmt.Fprintf(out, "%s [y/N]: ", prompt)
	line, err := bufio.NewReader(in).ReadString('\n')
	if err != nil && line == "" {
		return false, nil
	}
	answer := strings.ToLower(strings.TrimSpace(line))
	return answer == "y" || answer == "yes", nil
}
