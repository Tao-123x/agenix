package agenix

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type OpenAIAnalyzeRequest struct {
	Skill   string
	Context string
}

type OpenAIAnalyzeResult struct {
	AnalysisSummary string   `json:"analysis_summary"`
	FailingTests    []string `json:"failing_tests"`
	LikelyRootCause string   `json:"likely_root_cause"`
	ChangedFiles    []string `json:"changed_files"`
}

type OpenAIAnalyzeClient struct {
	BaseURL string
	APIKey  string
	Model   string
	Client  *http.Client
}

func (c OpenAIAnalyzeClient) Analyze(request OpenAIAnalyzeRequest) (OpenAIAnalyzeResult, error) {
	if strings.TrimSpace(c.APIKey) == "" {
		return OpenAIAnalyzeResult{}, NewError(ErrDriverError, "missing OpenAI API key")
	}
	baseURL := strings.TrimSpace(c.BaseURL)
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	client := c.Client
	if client == nil {
		client = http.DefaultClient
	}

	body := struct {
		Model string `json:"model"`
		Input string `json:"input"`
	}{
		Model: c.Model,
		Input: request.Skill + "\n" + request.Context,
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return OpenAIAnalyzeResult{}, WrapError(ErrDriverError, "encode OpenAI request", err)
	}

	req, err := http.NewRequest(http.MethodPost, strings.TrimRight(baseURL, "/")+"/responses", bytes.NewReader(payload))
	if err != nil {
		return OpenAIAnalyzeResult{}, WrapError(ErrDriverError, "create OpenAI request", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.APIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return OpenAIAnalyzeResult{}, WrapError(ErrDriverError, "call OpenAI responses API", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return OpenAIAnalyzeResult{}, NewError(ErrDriverError, openAIResponseErrorMessage(resp))
	}

	var decoded struct {
		Output []struct {
			Content []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"content"`
		} `json:"output"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return OpenAIAnalyzeResult{}, WrapError(ErrDriverError, "decode OpenAI response", err)
	}

	for _, output := range decoded.Output {
		for _, content := range output.Content {
			if content.Type != "output_text" {
				continue
			}
			var result OpenAIAnalyzeResult
			if err := json.Unmarshal([]byte(content.Text), &result); err != nil {
				return OpenAIAnalyzeResult{}, WrapError(ErrDriverError, "decode OpenAI structured output", err)
			}
			return result, nil
		}
	}

	return OpenAIAnalyzeResult{}, NewError(ErrDriverError, "missing OpenAI structured output")
}

func openAIResponseErrorMessage(resp *http.Response) string {
	message := fmt.Sprintf("OpenAI responses API returned %s", resp.Status)

	body, err := io.ReadAll(resp.Body)
	if err != nil || len(body) == 0 {
		return message
	}

	var decoded struct {
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &decoded); err != nil {
		return message
	}

	apiMessage := strings.TrimSpace(decoded.Error.Message)
	if apiMessage != "" {
		message = message + ": " + apiMessage
	}

	if resp.StatusCode == http.StatusTooManyRequests {
		if retryAfter := parseRetryAfter(resp.Header.Get("Retry-After")); retryAfter != "" {
			message += " (retry after " + retryAfter + ")"
		}
	}

	return message
}

func parseRetryAfter(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}

	if seconds, err := strconv.Atoi(value); err == nil && seconds >= 0 {
		return fmt.Sprintf("%ds", seconds)
	}

	if retryAt, err := http.ParseTime(value); err == nil {
		delta := time.Until(retryAt)
		if delta < 0 {
			delta = 0
		}
		seconds := int(delta.Round(time.Second) / time.Second)
		if seconds < 0 {
			seconds = 0
		}
		return fmt.Sprintf("%ds", seconds)
	}

	return ""
}
