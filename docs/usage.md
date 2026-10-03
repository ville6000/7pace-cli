# Usage

## Syncing from toggl-cli

`7pace-cli sync` reads the time entries printed by `toggl-cli history --json`
and posts them as worklogs. Pick the date range on the toggl-cli side: today by
default, or `--day`, `--week`, `--month`, `--start` / `--end`.

```sh
toggl-cli history --week --json | 7pace-cli sync --dry-run   # preview
toggl-cli history --week --json | 7pace-cli sync             # post, after a confirmation prompt
toggl-cli history --day 2024-06-03 --json | 7pace-cli sync
```

- The Azure DevOps work item ID comes from the entry's description, such as
  `#1234 fix bug`, `AB#1234 ...` or a leading `1234 - ...`. Entries without one
  are skipped and listed.
- Entries with the same description on the same day become one worklog, with
  the time rounded up to the minute. Days and worklog times are in the timezone
  toggl-cli is configured for.
- Running entries are skipped.
- Posting the same entries again creates **duplicate worklogs**. Preview with
  `--dry-run` first.

The confirmation prompt is answered at the terminal, since stdin carries the
entries. Where there is no terminal (cron, CI), pass `--yes` to post without
asking. `sync` also reads a file instead of stdin:

```sh
toggl-cli history --week --json > week.json
7pace-cli sync week.json
```

## Adding a single worklog

```sh
7pace-cli add --work-item 1234 --duration 1h30m --comment "code review"
7pace-cli add --comment "planning" --duration 45m --date "2024-06-03 09:00"

# Short flags: -w work item, -d duration, -c comment, -D date
7pace-cli add -w 1234 -d 1h30m -c "code review"
```

A worklog needs a work item (`--work-item`) or a comment (`--comment`), and a
`--duration` such as `1h30m` or a number of seconds. `--date` takes
`YYYY-MM-DD` or `"YYYY-MM-DD HH:MM"` in the system timezone, defaulting to now.
`--activity-type` overrides the configured activity type.
