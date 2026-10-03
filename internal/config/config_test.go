package config

import (
	"strings"
	"testing"

	"github.com/spf13/viper"
)

func TestLoad_ReadsAllSettings(t *testing.T) {
	v := viper.New()
	v.Set("base_url", "https://7pace.example")
	v.Set("domain", "CORP")
	v.Set("username", "user")
	v.Set("password", "secret")
	v.Set("activity_type_id", "activity-uuid")
	v.Set("insecure_skip_verify", true)

	got, err := Load(v)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	want := Config{
		BaseURL:         "https://7pace.example",
		Domain:          "CORP",
		Username:        "user",
		Password:        "secret",
		ActivityTypeID:  "activity-uuid",
		InsecureSkipTLS: true,
	}
	if got != want {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestLoad_RequiresBaseURLAndCredentials(t *testing.T) {
	tests := map[string]struct {
		set  map[string]string
		want string
	}{
		"no base url": {map[string]string{"username": "u", "password": "p"}, "base_url"},
		"no username": {map[string]string{"base_url": "https://x", "password": "p"}, "username"},
		"no password": {map[string]string{"base_url": "https://x", "username": "u"}, "password"},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			v := viper.New()
			for key, value := range tt.set {
				v.Set(key, value)
			}

			_, err := Load(v)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Errorf("error should mention %q, got %v", tt.want, err)
			}
		})
	}
}

func TestEnvVar(t *testing.T) {
	tests := map[string]string{
		"base_url": "SEVENPACE_CLI_BASE_URL",
		"password": "SEVENPACE_CLI_PASSWORD",
	}
	for key, want := range tests {
		if got := EnvVar(key); got != want {
			t.Errorf("EnvVar(%q) = %q, want %q", key, got, want)
		}
	}
}

func TestUseEnv_ReadsPrefixedVariables(t *testing.T) {
	t.Setenv("SEVENPACE_CLI_BASE_URL", "https://7pace.example")
	t.Setenv("SEVENPACE_CLI_USERNAME", "user")
	t.Setenv("SEVENPACE_CLI_PASSWORD", "secret")

	v := viper.New()
	UseEnv(v)

	cfg, err := Load(v)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.BaseURL != "https://7pace.example" || cfg.Username != "user" || cfg.Password != "secret" {
		t.Errorf("unexpected config from env: %+v", cfg)
	}
}

func TestMissingKeyErrorNamesTheEnvVar(t *testing.T) {
	_, err := Load(viper.New())
	if err == nil || !strings.Contains(err.Error(), "SEVENPACE_CLI_BASE_URL") {
		t.Errorf("error should name SEVENPACE_CLI_BASE_URL, got %v", err)
	}
}
