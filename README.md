# 7pace-cli

A command line client for posting worklogs to an on-prem
[7pace Timetracker](https://www.7pace.com/): one at a time, or in bulk from your
[Toggl Track](https://toggl.com/track/) time entries via
[toggl-cli](https://github.com/ville6000/toggl-cli).

```console
$ toggl-cli history --week --json | 7pace-cli sync --dry-run
$ toggl-cli history --week --json | 7pace-cli sync
$ 7pace-cli add --work-item 1234 --duration 1h30m --comment "code review"
```

- [Installation](#installation)
- [Getting started](#getting-started)
- [Syncing from toggl-cli](#syncing-from-toggl-cli)
- [Adding a single worklog](#adding-a-single-worklog)
- [Configuration](#configuration)
- [Contributing](#contributing)
- [License](#license)

## Installation

### Homebrew

On macOS and Linux:

```sh
brew install ville6000/tap/7pace-cli
```

### Prebuilt binaries

Each [release](https://github.com/ville6000/7pace-cli/releases) has archives for
macOS, Linux and Windows on amd64 and arm64, plus a `checksums.txt`.

On macOS and Linux, set `PLATFORM` to `darwin_arm64` (Apple silicon),
`darwin_amd64` (Intel Mac), `linux_amd64` or `linux_arm64`:

```sh
PLATFORM=darwin_arm64
BASE=https://github.com/ville6000/7pace-cli/releases/latest/download
curl -fsSLO "$BASE/7pace-cli_$PLATFORM.tar.gz"
curl -fsSLO "$BASE/checksums.txt"
grep "7pace-cli_$PLATFORM.tar.gz" checksums.txt | shasum -a 256 -c
tar -xzf "7pace-cli_$PLATFORM.tar.gz" 7pace-cli
sudo mv 7pace-cli /usr/local/bin/
```

On Windows, download `7pace-cli_windows_amd64.zip` (or `_arm64`) from the
[latest release](https://github.com/ville6000/7pace-cli/releases/latest),
extract `7pace-cli.exe` and put it in a folder on your `PATH`.

The binaries aren't signed. On macOS, a binary downloaded with a browser rather
than `curl` is quarantined by Gatekeeper; allow it with
`xattr -d com.apple.quarantine 7pace-cli`.

### With Go

With Go 1.26 or newer:

```sh
go install github.com/ville6000/7pace-cli@latest
```

This installs `7pace-cli` into `$(go env GOPATH)/bin`.

## Getting started

Run the interactive setup, which asks for the 7pace REST API base URL, your
Windows credentials and an optional default activity type:

```sh
7pace-cli config
```

7pace's on-prem API uses NTLM (Windows) authentication, so your Windows
password is stored in the config file **in plaintext**; the file is readable
only by you. Reaching the server usually requires the corporate network or VPN.

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

Run `7pace-cli <command> --help` for all flags, and `7pace-cli --version` for
the installed version.

## Configuration

`7pace-cli` reads its settings from the first of these files that exists:

1. `$XDG_CONFIG_HOME/7pace-cli/config.yaml`
2. `~/.config/7pace-cli/config.yaml`

`7pace-cli config` writes that file, readable only by you. Commands accept
`--config <file>` to read a different file.

```yaml
base_url: https://timetracker.example.com:8090/api/YourCollection/rest  # required
domain: CORP                              # optional
username: <windows_username>              # required
password: <windows_password>              # required
activity_type_id: <activity_type_uuid>    # optional
insecure_skip_verify: false               # true only for self-signed certificates
```

### Environment variables

Any setting can also come from an environment variable named `SEVENPACE_CLI_`
followed by the key in upper case. Variables override the config file:

```sh
export SEVENPACE_CLI_PASSWORD=<your_windows_password>
```

### Moving from `toggl-cli 7pace`

The 7pace commands used to be part of toggl-cli. To keep your settings, copy
the keys under `sevenpace:` in your toggl-cli config to the top level of
`~/.config/7pace-cli/config.yaml`, or run `7pace-cli config`. Then:

| Before | Now |
| --- | --- |
| `toggl-cli 7pace sync --week` | `toggl-cli history --week --json \| 7pace-cli sync` |
| `toggl-cli 7pace sync --start 2024-06-03` | `toggl-cli history --day 2024-06-03 --json \| 7pace-cli sync` |
| `toggl-cli 7pace add ...` | `7pace-cli add ...` |

Note that `history --start` alone runs through today, while the old
`7pace sync --start` covered only that day; use `--day` for a single day.

## Contributing

Bug reports and pull requests are welcome. See
[CONTRIBUTING.md](CONTRIBUTING.md) for setting up a development environment,
running the tests and making a release.

## License

[MIT](LICENSE)
