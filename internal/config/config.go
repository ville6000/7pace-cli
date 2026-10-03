// Package config reads 7pace-cli settings from a loaded configuration.
package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

// EnvPrefix starts the name of every environment variable that sets a config
// key: password is read from SEVENPACE_CLI_PASSWORD.
const EnvPrefix = "SEVENPACE_CLI"

var envKeyReplacer = strings.NewReplacer(".", "_")

// UseEnv makes v read config keys from the environment as well. A variable
// outranks the config file, so credentials can be supplied without one.
func UseEnv(v *viper.Viper) {
	v.SetEnvPrefix(EnvPrefix)
	v.SetEnvKeyReplacer(envKeyReplacer)
	v.AutomaticEnv()
}

// EnvVar returns the environment variable that sets key.
func EnvVar(key string) string {
	return EnvPrefix + "_" + strings.ToUpper(envKeyReplacer.Replace(key))
}

// missing returns the error for required keys that are not set.
func missing(keys ...string) error {
	vars := make([]string, len(keys))
	for i, key := range keys {
		vars[i] = EnvVar(key)
	}

	return fmt.Errorf("missing %s in config (or %s), please run '7pace-cli config'",
		strings.Join(keys, " or "), strings.Join(vars, " / "))
}

// Config holds the settings for talking to an on-prem 7pace Timetracker
// instance using NTLM (Windows) authentication.
type Config struct {
	BaseURL         string
	Domain          string
	Username        string
	Password        string
	ActivityTypeID  string
	InsecureSkipTLS bool
}

// Load reads the configuration from v. Base URL, username and password are
// required; domain and activity type are optional.
func Load(v *viper.Viper) (Config, error) {
	cfg := Config{
		BaseURL:         v.GetString("base_url"),
		Domain:          v.GetString("domain"),
		Username:        v.GetString("username"),
		Password:        v.GetString("password"),
		ActivityTypeID:  v.GetString("activity_type_id"),
		InsecureSkipTLS: v.GetBool("insecure_skip_verify"),
	}

	if cfg.BaseURL == "" {
		return Config{}, missing("base_url")
	}
	if cfg.Username == "" || cfg.Password == "" {
		return Config{}, missing("username", "password")
	}

	return cfg, nil
}
