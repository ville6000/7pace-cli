package cmd

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/spf13/viper"
)

const workLogPath = "/workLogs"

// stubRequest is one request a command made against the stub server.
type stubRequest struct {
	Method string
	Path   string
	Body   string
}

// decodeBody unmarshals a recorded JSON request body into v.
func (r stubRequest) decodeBody(t *testing.T, v any) {
	t.Helper()
	if err := json.Unmarshal([]byte(r.Body), v); err != nil {
		t.Fatalf("decode %s %s body %q: %v", r.Method, r.Path, r.Body, err)
	}
}

// apiStub is a stand-in for the 7pace API. Routes are registered per
// "METHOD /path"; a request to any other route fails the test instead of
// quietly returning a 404 that the command would report as some other error.
type apiStub struct {
	t      *testing.T
	server *httptest.Server

	mu       sync.Mutex
	routes   map[string]http.HandlerFunc
	requests []stubRequest
}

func newAPIStub(t *testing.T) *apiStub {
	t.Helper()
	stub := &apiStub{t: t, routes: make(map[string]http.HandlerFunc)}
	stub.server = httptest.NewServer(stub)
	t.Cleanup(stub.server.Close)

	return stub
}

func (s *apiStub) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		s.t.Errorf("read request body: %v", err)
	}

	s.mu.Lock()
	s.requests = append(s.requests, stubRequest{Method: r.Method, Path: r.URL.Path, Body: string(body)})
	handler, ok := s.routes[r.Method+" "+r.URL.Path]
	s.mu.Unlock()

	if !ok {
		s.t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		w.WriteHeader(http.StatusNotFound)
		return
	}

	handler(w, r)
}

// respondToWorkLogs registers a canned JSON response for posting a worklog.
func (s *apiStub) respondToWorkLogs(status int, body any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.routes[http.MethodPost+" "+workLogPath] = func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		if body == nil {
			return
		}
		if err := json.NewEncoder(w).Encode(body); err != nil {
			s.t.Errorf("encode worklog response: %v", err)
		}
	}
}

// requestsFor returns the recorded requests for one route, in order.
func (s *apiStub) requestsFor(method, path string) []stubRequest {
	s.mu.Lock()
	defer s.mu.Unlock()

	var matching []stubRequest
	for _, req := range s.requests {
		if req.Method == method && req.Path == path {
			matching = append(matching, req)
		}
	}

	return matching
}

// onlyRequestFor returns the single recorded request for a route, failing the
// test when the command made none or more than one.
func (s *apiStub) onlyRequestFor(method, path string) stubRequest {
	s.t.Helper()
	matching := s.requestsFor(method, path)
	if len(matching) != 1 {
		s.t.Fatalf("%s %s: got %d requests, want 1", method, path, len(matching))
	}

	return matching[0]
}

// setupCLITest isolates a command run from the developer's real config, and
// returns the configuration to run commands with, pointing the client at stub.
func setupCLITest(t *testing.T, stub *apiStub) *viper.Viper {
	t.Helper()

	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, "config"))

	v := viper.New()
	v.Set("username", "user")
	v.Set("password", "secret")
	v.Set("activity_type_id", "activity-uuid")
	if stub != nil {
		v.Set("base_url", stub.server.URL)
	}

	return v
}

// stubTerminal makes the sync confirmation prompt read its answer from input,
// as if typed at the terminal, and reports whether the terminal was opened.
func stubTerminal(t *testing.T, input string) *bool {
	t.Helper()

	orig := openTerminal
	t.Cleanup(func() { openTerminal = orig })

	opened := new(bool)
	openTerminal = func() (io.ReadCloser, error) {
		*opened = true
		return io.NopCloser(strings.NewReader(input)), nil
	}

	return opened
}

// executeCommand runs the CLI the way main() does — through a full command
// tree, so flag parsing, config lookup and output rendering are all exercised
// — and returns what the command wrote to stdout and stderr.
func executeCommand(t *testing.T, v *viper.Viper, args ...string) (stdout, stderr string, err error) {
	t.Helper()
	return executeCommandWithInput(t, v, "", args...)
}

// executeCommandWithInput is executeCommand with stdin wired to input.
func executeCommandWithInput(t *testing.T, v *viper.Viper, input string, args ...string) (stdout, stderr string, err error) {
	t.Helper()

	root := newRootCmd(v)

	var outBuf, errBuf bytes.Buffer
	root.SetOut(&outBuf)
	root.SetErr(&errBuf)
	root.SetIn(bytes.NewBufferString(input))
	root.SetArgs(args)

	err = root.ExecuteContext(t.Context())

	return outBuf.String(), errBuf.String(), err
}
