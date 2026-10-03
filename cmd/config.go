package cmd

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/term"
)

func newConfigCmd(v *viper.Viper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Create or update the config file",
		Long: "Prompt for the 7pace Timetracker settings and save them to the config file.\n" +
			"The Windows password is stored in plaintext, in a file readable only by you.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			out := cmd.OutOrStdout()
			in := cmd.InOrStdin()
			reader := bufio.NewReader(in)

			prompt := func(label string) (string, error) {
				fmt.Fprint(out, label)
				line, err := reader.ReadString('\n')
				if err != nil {
					return "", fmt.Errorf("error reading input: %w", err)
				}
				return strings.TrimSpace(line), nil
			}

			var s settings
			var err error
			if s.baseURL, err = prompt("7pace REST API base URL: "); err != nil {
				return err
			}
			if s.baseURL == "" {
				return errors.New("the base URL is required")
			}
			if s.domain, err = prompt("Windows domain (leave empty if none): "); err != nil {
				return err
			}
			if s.username, err = prompt("Windows username: "); err != nil {
				return err
			}
			fmt.Fprint(out, "Windows password (stored in plaintext): ")
			if s.password, err = readSecret(out, in, reader); err != nil {
				return err
			}
			if s.activityTypeID, err = prompt("Activity type UUID (optional): "); err != nil {
				return err
			}

			if err := writeConfig(v, s); err != nil {
				return fmt.Errorf("error saving configuration: %w", err)
			}

			fmt.Fprintln(out, "Configuration saved successfully!")
			return nil
		},
	}

	return cmd
}

// settings holds the values gathered during interactive configuration.
//
// Note: the password is stored in plaintext in the config file — this is the
// tradeoff of using NTLM credentials from config.
type settings struct {
	baseURL        string
	domain         string
	username       string
	password       string
	activityTypeID string
}

// Terminal access used by readSecret, replaced in tests.
var (
	isTerminal   = term.IsTerminal
	readPassword = term.ReadPassword
)

// readSecret reads a line without echoing it when in is a terminal, so a
// password doesn't end up on screen or in scrollback. Otherwise (piped input,
// tests) it reads the next line from reader like any other prompt.
func readSecret(out io.Writer, in io.Reader, reader *bufio.Reader) (string, error) {
	if f, ok := in.(*os.File); ok && isTerminal(int(f.Fd())) {
		secret, err := readPassword(int(f.Fd()))
		// The Enter that ended the input wasn't echoed either.
		fmt.Fprintln(out)
		if err != nil {
			return "", fmt.Errorf("error reading input: %w", err)
		}
		return strings.TrimSpace(string(secret)), nil
	}

	line, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("error reading input: %w", err)
	}
	return strings.TrimSpace(line), nil
}

func writeConfig(v *viper.Viper, s settings) error {
	configPath, err := ConfigPath()
	if err != nil {
		return fmt.Errorf("failed to get config path: %w", err)
	}

	// The directory does not exist yet on a fresh install, and viper only
	// creates the file. Keep it private: the file holds a plaintext password.
	if err := os.MkdirAll(filepath.Dir(configPath), 0o700); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	v.SetConfigFile(configPath)

	v.Set("base_url", s.baseURL)
	v.Set("domain", s.domain)
	v.Set("username", s.username)
	v.Set("password", s.password)
	v.Set("activity_type_id", s.activityTypeID)

	writeErr := v.WriteConfig()

	if writeErr != nil {
		if _, ok := errors.AsType[viper.ConfigFileNotFoundError](writeErr); !ok {
			return writeErr
		}

		if err := v.SafeWriteConfig(); err != nil {
			return fmt.Errorf("failed to create config file: %w", err)
		}
	}

	return restrictConfigFile(configPath)
}

// restrictConfigFile makes the config file readable by its owner only. It holds
// a plaintext password, and viper writes files with the process umask, which
// can leave them world-readable.
func restrictConfigFile(path string) error {
	if err := os.Chmod(path, 0o600); err != nil {
		return fmt.Errorf("failed to restrict config file permissions: %w", err)
	}

	return nil
}

// ConfigPath returns the config file to use: the first existing candidate,
// or the preferred XDG path for new installs.
func ConfigPath() (string, error) {
	candidates, err := configCandidates()
	if err != nil {
		return "", err
	}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}

	return candidates[0], nil
}

// configCandidates returns config file paths in priority order:
// XDG_CONFIG_HOME, then ~/.config/7pace-cli.
func configCandidates() ([]string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	candidates := []string{}

	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		candidates = append(candidates, filepath.Join(xdg, "7pace-cli", "config.yaml"))
	}

	candidates = append(candidates, filepath.Join(home, ".config", "7pace-cli", "config.yaml"))

	return candidates, nil
}
