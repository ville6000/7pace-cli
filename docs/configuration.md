# Configuration

## Config file

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

7pace's on-prem API uses NTLM (Windows) authentication, so your Windows
password is stored in the config file **in plaintext**. To keep it out of the
file, set it as an environment variable instead.

## Environment variables

Any setting can also come from an environment variable named `SEVENPACE_CLI_`
followed by the key in upper case. Variables override the config file:

```sh
export SEVENPACE_CLI_PASSWORD=<your_windows_password>
```

## Moving from `toggl-cli 7pace`

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
