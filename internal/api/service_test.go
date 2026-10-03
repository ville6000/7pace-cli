package api

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// newTestClient creates a Client pointing at the test server,
// bypassing the NTLM negotiator (which is only exercised against a real server).
func newTestClient(t *testing.T, handler http.Handler) *Client {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return &Client{
		BaseURL:    server.URL,
		HTTPClient: server.Client(),
		Domain:     "CORP",
		Username:   "user",
		Password:   "secret",
	}
}

func TestCreateWorkLog_PostsToEndpoint(t *testing.T) {
	var capturedMethod, capturedPath, capturedQuery, capturedAuth string
	var capturedBody WorkLog
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedMethod = r.Method
		capturedPath = r.URL.Path
		capturedQuery = r.URL.RawQuery
		capturedAuth = r.Header.Get("Authorization")

		body, _ := io.ReadAll(r.Body)
		if err := json.Unmarshal(body, &capturedBody); err != nil {
			t.Errorf("unmarshal body: %v", err)
		}

		if err := json.NewEncoder(w).Encode(capturedBody); err != nil {
			t.Errorf("encode: %v", err)
		}
	})
	client := newTestClient(t, handler)

	workItem := 1234
	input := WorkLog{
		Timestamp:  "2024-01-02T10:00:00Z",
		Length:     3600,
		WorkItemID: &workItem,
		Comment:    "#1234 do stuff",
	}

	got, err := client.CreateWorkLog(t.Context(), input)
	if err != nil {
		t.Fatalf("CreateWorkLog: %v", err)
	}

	if capturedMethod != http.MethodPost {
		t.Errorf("method: got %s, want POST", capturedMethod)
	}
	if capturedPath != "/workLogs" {
		t.Errorf("path: got %q, want /workLogs", capturedPath)
	}
	if capturedQuery != "api-version=3.0" {
		t.Errorf("query: got %q, want api-version=3.0", capturedQuery)
	}
	if capturedAuth == "" {
		t.Error("expected Basic auth header to be set for NTLM negotiation")
	}
	if capturedBody.Length != 3600 || capturedBody.WorkItemID == nil || *capturedBody.WorkItemID != 1234 {
		t.Errorf("unexpected body: %+v", capturedBody)
	}
	if got.Comment != "#1234 do stuff" {
		t.Errorf("response comment: got %q", got.Comment)
	}
}

func TestCreateWorkLog_HTTPError(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))

	if _, err := client.CreateWorkLog(t.Context(), WorkLog{Length: 3600}); err == nil {
		t.Error("expected error for HTTP 400")
	}
}

func TestCreateWorkLog_UnauthorizedAddsCredentialHint(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("WWW-Authenticate", "NTLM")
		w.WriteHeader(http.StatusUnauthorized)
	}))

	_, err := client.CreateWorkLog(t.Context(), WorkLog{Length: 3600})
	if err == nil {
		t.Fatal("expected error for HTTP 401")
	}

	for _, want := range []string{"401", `"NTLM"`, "domain/username/password"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error should contain %q, got %q", want, err)
		}
	}
}

func TestCreateWorkLog_ErrorIncludesResponseBody(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "work item not found", http.StatusBadRequest)
	}))

	_, err := client.CreateWorkLog(t.Context(), WorkLog{Length: 3600})

	var statusErr *statusError
	if !errors.As(err, &statusErr) || statusErr.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected *statusError with 400, got %v", err)
	}
	if !strings.Contains(err.Error(), "work item not found") {
		t.Errorf("error should include the response body, got %q", err)
	}
}

func TestCreateWorkLog_StopsWhenContextIsCancelled(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	if _, err := client.CreateWorkLog(ctx, WorkLog{Length: 3600}); !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}

func TestStatusError_Message(t *testing.T) {
	tests := []struct {
		name string
		err  statusError
		want string
	}{
		{"without body", statusError{Status: "400 Bad Request"}, "request failed: 400 Bad Request"},
		{"with body", statusError{Status: "400 Bad Request", Body: "bad work item"}, "request failed: 400 Bad Request: bad work item"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.want {
				t.Errorf("Error() = %q, want %q", got, tt.want)
			}
		})
	}
}
