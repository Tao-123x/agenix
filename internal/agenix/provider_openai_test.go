package agenix

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestOpenAIClientAnalyzeReturnsStructuredOutput(t *testing.T) {
	type requestBody struct {
		Model string `json:"model"`
		Input string `json:"input"`
	}

	handlerErr := make(chan error, 1)
	var captured requestBody
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		fail := func(format string, args ...any) {
			handlerErr <- fmt.Errorf(format, args...)
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte("request validation failed"))
		}
		if r.Method != http.MethodPost {
			fail("method = %q", r.Method)
			return
		}
		if r.URL.Path != "/responses" {
			fail("path = %q", r.URL.Path)
			return
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			fail("authorization = %q", got)
			return
		}
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			fail("content-type = %q", got)
			return
		}
		if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
			fail("decode request body: %v", err)
			return
		}
		if captured.Model != "gpt-5.4" {
			fail("model = %q", captured.Model)
			return
		}
		if captured.Input != "repo.analyze_test_failures\npytest output and relevant files" {
			fail("input = %q", captured.Input)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
  "output": [
    {
      "type": "message",
      "content": [
        {
          "type": "output_text",
          "text": "{\"analysis_summary\":\"The add helper subtracts instead of adding.\",\"failing_tests\":[\"test_mathlib.py::test_adds_numbers\"],\"likely_root_cause\":\"mathlib.add returns a - b.\",\"changed_files\":[]}"
        }
      ]
    }
  ]
}`))
	}))
	defer server.Close()

	client := OpenAIAnalyzeClient{
		BaseURL: server.URL,
		APIKey:  "test-key",
		Model:   "gpt-5.4",
	}

	result, err := client.Analyze(OpenAIAnalyzeRequest{
		Skill:   "repo.analyze_test_failures",
		Context: "pytest output and relevant files",
	})
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
	select {
	case err := <-handlerErr:
		if err != nil {
			t.Fatalf("handler validation failed: %v", err)
		}
	default:
	}
	if result.AnalysisSummary != "The add helper subtracts instead of adding." {
		t.Fatalf("analysis_summary = %q", result.AnalysisSummary)
	}
	if len(result.FailingTests) != 1 || result.FailingTests[0] != "test_mathlib.py::test_adds_numbers" {
		t.Fatalf("failing_tests = %#v", result.FailingTests)
	}
	if result.LikelyRootCause != "mathlib.add returns a - b." {
		t.Fatalf("likely_root_cause = %q", result.LikelyRootCause)
	}
	if len(result.ChangedFiles) != 0 {
		t.Fatalf("changed_files = %#v", result.ChangedFiles)
	}
	if captured.Model != "gpt-5.4" {
		t.Fatalf("captured model = %q", captured.Model)
	}
	if captured.Input != "repo.analyze_test_failures\npytest output and relevant files" {
		t.Fatalf("captured input = %q", captured.Input)
	}
}

func TestOpenAIClientAnalyzeReturnsStructuredAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":{"message":"invalid API key","type":"invalid_request_error","code":"invalid_api_key"}}`))
	}))
	defer server.Close()

	client := OpenAIAnalyzeClient{
		BaseURL: server.URL,
		APIKey:  "test-key",
		Model:   "gpt-5.4",
	}

	_, err := client.Analyze(OpenAIAnalyzeRequest{
		Skill:   "repo.analyze_test_failures",
		Context: "fixture context",
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if !IsErrorClass(err, ErrDriverError) {
		t.Fatalf("expected DriverError, got %v", err)
	}
	if got := err.Error(); got != "DriverError: OpenAI responses API returned 401 Unauthorized: invalid API key" {
		t.Fatalf("error = %q", got)
	}
}

func TestOpenAIClientAnalyzeIncludesRetryAfterForRateLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Retry-After", "120")
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"error":{"message":"rate limit exceeded","type":"rate_limit_error","code":"rate_limit_exceeded"}}`))
	}))
	defer server.Close()

	client := OpenAIAnalyzeClient{
		BaseURL: server.URL,
		APIKey:  "test-key",
		Model:   "gpt-5.4",
	}

	_, err := client.Analyze(OpenAIAnalyzeRequest{
		Skill:   "repo.analyze_test_failures",
		Context: "fixture context",
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if !IsErrorClass(err, ErrDriverError) {
		t.Fatalf("expected DriverError, got %v", err)
	}
	if got := err.Error(); got != "DriverError: OpenAI responses API returned 429 Too Many Requests: rate limit exceeded (retry after 120s)" {
		t.Fatalf("error = %q", got)
	}
}

func TestOpenAIClientAnalyzeIncludesHTTPDateRetryAfterForRateLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Retry-After", "Thu, 16 Apr 2026 01:00:00 GMT")
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"error":{"message":"","type":"rate_limit_error","code":"rate_limit_exceeded"}}`))
	}))
	defer server.Close()

	client := OpenAIAnalyzeClient{
		BaseURL: server.URL,
		APIKey:  "test-key",
		Model:   "gpt-5.4",
	}

	_, err := client.Analyze(OpenAIAnalyzeRequest{
		Skill:   "repo.analyze_test_failures",
		Context: "fixture context",
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if !IsErrorClass(err, ErrDriverError) {
		t.Fatalf("expected DriverError, got %v", err)
	}
	if got := err.Error(); got != "DriverError: OpenAI responses API returned 429 Too Many Requests (retry after 0s)" {
		t.Fatalf("error = %q", got)
	}
}

func TestOpenAIClientAnalyzeIgnoresRetryAfterForNonRateLimitErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Retry-After", "120")
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"error":{"message":"forbidden","type":"invalid_request_error","code":"forbidden"}}`))
	}))
	defer server.Close()

	client := OpenAIAnalyzeClient{
		BaseURL: server.URL,
		APIKey:  "test-key",
		Model:   "gpt-5.4",
	}

	_, err := client.Analyze(OpenAIAnalyzeRequest{
		Skill:   "repo.analyze_test_failures",
		Context: "fixture context",
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if !IsErrorClass(err, ErrDriverError) {
		t.Fatalf("expected DriverError, got %v", err)
	}
	if got := err.Error(); got != "DriverError: OpenAI responses API returned 403 Forbidden: forbidden" {
		t.Fatalf("error = %q", got)
	}
}

func TestOpenAIClientAnalyzeFallsBackToStatusOnlyForMalformedErrorBody(t *testing.T) {
	cases := []struct {
		name   string
		status int
		body   string
	}{
		{
			name:   "plain-text",
			status: http.StatusForbidden,
			body:   "forbidden",
		},
		{
			name:   "malformed-json",
			status: http.StatusInternalServerError,
			body:   "{not-json",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/plain")
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(tc.body))
			}))
			defer server.Close()

			client := OpenAIAnalyzeClient{
				BaseURL: server.URL,
				APIKey:  "test-key",
				Model:   "gpt-5.4",
			}

			_, err := client.Analyze(OpenAIAnalyzeRequest{
				Skill:   "repo.analyze_test_failures",
				Context: "fixture context",
			})
			if err == nil {
				t.Fatal("expected error")
			}
			if !IsErrorClass(err, ErrDriverError) {
				t.Fatalf("expected DriverError, got %v", err)
			}
			want := fmt.Sprintf("DriverError: OpenAI responses API returned %d %s", tc.status, http.StatusText(tc.status))
			if got := err.Error(); got != want {
				t.Fatalf("error = %q", got)
			}
		})
	}
}

func TestOpenAIClientAnalyzeRejectsOversizedSuccessfulResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"output":[{"content":[{"type":"output_text","text":"` + strings.Repeat("x", 128) + `"}]}]}`))
	}))
	defer server.Close()

	client := OpenAIAnalyzeClient{
		BaseURL:          server.URL,
		APIKey:           "test-key",
		Model:            "gpt-5.4",
		MaxResponseBytes: 64,
	}

	_, err := client.Analyze(OpenAIAnalyzeRequest{
		Skill:   "repo.analyze_test_failures",
		Context: "fixture context",
	})
	if err == nil {
		t.Fatal("expected oversized response error")
	}
	if !IsErrorClass(err, ErrDriverError) {
		t.Fatalf("expected DriverError, got %v", err)
	}
	if got := err.Error(); got != "DriverError: OpenAI response body exceeded 64 bytes" {
		t.Fatalf("error = %q", got)
	}
}

func TestOpenAIClientAnalyzeFallsBackForOversizedErrorBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":{"message":"` + strings.Repeat("x", 128) + `"}}`))
	}))
	defer server.Close()

	client := OpenAIAnalyzeClient{
		BaseURL:          server.URL,
		APIKey:           "test-key",
		Model:            "gpt-5.4",
		MaxResponseBytes: 64,
	}

	_, err := client.Analyze(OpenAIAnalyzeRequest{
		Skill:   "repo.analyze_test_failures",
		Context: "fixture context",
	})
	if err == nil {
		t.Fatal("expected provider error")
	}
	if !IsErrorClass(err, ErrDriverError) {
		t.Fatalf("expected DriverError, got %v", err)
	}
	if got := err.Error(); got != "DriverError: OpenAI responses API returned 500 Internal Server Error" {
		t.Fatalf("error = %q", got)
	}
}

func TestOpenAIClientAnalyzeFailsWithoutAPIKey(t *testing.T) {
	client := OpenAIAnalyzeClient{}

	_, err := client.Analyze(OpenAIAnalyzeRequest{
		Skill:   "repo.analyze_test_failures",
		Context: "fixture context",
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if !IsErrorClass(err, ErrDriverError) {
		t.Fatalf("expected DriverError, got %v", err)
	}
}

func TestOpenAIClientAnalyzeTimesOutSlowProvider(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(50 * time.Millisecond)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"output":[]}`))
	}))
	defer server.Close()

	client := OpenAIAnalyzeClient{
		BaseURL: server.URL,
		APIKey:  "test-key",
		Model:   "gpt-5.4",
		Timeout: 5 * time.Millisecond,
	}

	_, err := client.Analyze(OpenAIAnalyzeRequest{
		Skill:   "repo.analyze_test_failures",
		Context: "fixture context",
	})
	if err == nil {
		t.Fatal("expected timeout")
	}
	if !IsErrorClass(err, ErrTimeout) {
		t.Fatalf("expected Timeout, got %v", err)
	}
}

func TestOpenAIClientAnalyzeRejectsMalformedResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
  "output": [
    {
      "type": "message",
      "content": [
        {
          "type": "output_text",
          "text": "not-json"
        }
      ]
    }
  ]
}`))
	}))
	defer server.Close()

	client := OpenAIAnalyzeClient{
		BaseURL: server.URL,
		APIKey:  "test-key",
		Model:   "gpt-5.4",
	}

	_, err := client.Analyze(OpenAIAnalyzeRequest{
		Skill:   "repo.analyze_test_failures",
		Context: strings.Repeat("x", 8),
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if !IsErrorClass(err, ErrDriverError) {
		t.Fatalf("expected DriverError, got %v", err)
	}
}
