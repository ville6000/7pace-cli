# Installation

## Homebrew

On macOS and Linux:

```sh
brew install ville6000/tap/7pace-cli
```

## Prebuilt binaries

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
gh attestation verify "7pace-cli_$PLATFORM.tar.gz" --repo ville6000/7pace-cli
tar -xzf "7pace-cli_$PLATFORM.tar.gz" 7pace-cli
sudo mv 7pace-cli /usr/local/bin/
```

On Windows, download `7pace-cli_windows_amd64.zip` (or `_arm64`) from the
[latest release](https://github.com/ville6000/7pace-cli/releases/latest),
extract `7pace-cli.exe` and put it in a folder on your `PATH`.

The `gh attestation verify` line (needs the [GitHub CLI](https://cli.github.com))
checks the archive was built by this repo's release workflow; skip it if you
don't have `gh`.

The binaries aren't code-signed. On macOS, a binary downloaded with a browser
rather than `curl` is quarantined by Gatekeeper; allow it with
`xattr -d com.apple.quarantine 7pace-cli`.

## With Go

With Go 1.26 or newer:

```sh
go install github.com/ville6000/7pace-cli@latest
```

This installs `7pace-cli` into `$(go env GOPATH)/bin`.

## Shell completion

`7pace-cli completion` prints a completion script for bash, zsh, fish or
PowerShell. For example, for zsh:

```sh
7pace-cli completion zsh > "${fpath[1]}/_7pace-cli"
```

See `7pace-cli completion <shell> --help` for each shell's instructions.
