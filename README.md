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

## Installation

```sh
brew install ville6000/tap/7pace-cli
```

Prebuilt binaries for macOS, Linux and Windows, `go install` and shell
completion are covered in [docs/install.md](docs/install.md).

## Getting started

1. Run the interactive setup, which asks for the 7pace REST API base URL, your
   Windows credentials and an optional default activity type:

   ```sh
   7pace-cli config
   ```

   7pace's on-prem API uses NTLM (Windows) authentication, so your Windows
   password is stored in the config file **in plaintext**; the file is
   readable only by you. Reaching the server usually requires the corporate
   network or VPN.
2. Preview a week of Toggl entries, then post them:

   ```sh
   toggl-cli history --week --json | 7pace-cli sync --dry-run
   toggl-cli history --week --json | 7pace-cli sync
   ```

   The work item ID comes from each entry's description, such as `#1234 fix
   bug`. Posting the same entries again creates **duplicate worklogs**.

## Usage

| Command | Description |
| --- | --- |
| `7pace-cli sync [file]` | Post toggl-cli entries from stdin or a file as worklogs |
| `7pace-cli add` | Post a single worklog (`-w` work item, `-d` duration, `-c` comment) |
| `7pace-cli config` | Create or update the config file |

Run `7pace-cli <command> --help` for all flags, and `7pace-cli --version` for
the installed version.

## Documentation

- [Installation](docs/install.md): binaries, `go install`, shell completion
- [Usage](docs/usage.md): how `sync` maps entries to worklogs, adding single
  worklogs
- [Configuration](docs/configuration.md): config file, environment variables,
  moving from `toggl-cli 7pace`

## Contributing

Bug reports and pull requests are welcome. See
[CONTRIBUTING.md](CONTRIBUTING.md) for setting up a development environment,
running the tests and making a release.

## License

[MIT](LICENSE)
