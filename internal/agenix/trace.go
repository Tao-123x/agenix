package agenix

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"os"
	"time"
)

type Trace struct {
	RunID        string       `json:"run_id"`
	Skill        string       `json:"skill"`
	ManifestPath string       `json:"manifest_path,omitempty"`
	ModelProfile string       `json:"model_profile"`
	StartedAt    time.Time    `json:"started_at"`
	Policy       Permissions  `json:"policy"`
	Events       []TraceEvent `json:"events"`
	Final        TraceFinal   `json:"final"`
}

type TraceEvent struct {
	Type       string      `json:"type"`
	Name       string      `json:"name"`
	Request    interface{} `json:"request,omitempty"`
	Result     interface{} `json:"result,omitempty"`
	Error      interface{} `json:"error,omitempty"`
	DurationMS int64       `json:"duration_ms,omitempty"`
	Status     string      `json:"status,omitempty"`
	Stdout     string      `json:"stdout,omitempty"`
	Stderr     string      `json:"stderr,omitempty"`
	ExitCode   int         `json:"exit_code,omitempty"`
}

type TraceFinal struct {
	Status string      `json:"status"`
	Output interface{} `json:"output,omitempty"`
	Error  string      `json:"error,omitempty"`
}

func NewTrace(skill, modelProfile string, permissions Permissions) *Trace {
	return &Trace{
		RunID:        newRunID(),
		Skill:        skill,
		ModelProfile: modelProfile,
		StartedAt:    time.Now().UTC(),
		Policy:       permissions,
		Events:       []TraceEvent{},
	}
}

func (t *Trace) AddToolEvent(name string, request, result interface{}, err error, durationMS int64) {
	event := TraceEvent{Type: "tool_call", Name: name, Request: request, Result: result, DurationMS: durationMS}
	if err != nil {
		event.Error = map[string]string{"class": ErrorClass(err), "message": err.Error()}
	}
	t.Events = append(t.Events, event)
}

func (t *Trace) AddVerifierEvent(name, verifierType, status, stdout, stderr string, exitCode int) {
	t.Events = append(t.Events, TraceEvent{
		Type:     "verifier",
		Name:     name,
		Request:  map[string]string{"type": verifierType},
		Status:   status,
		Stdout:   truncate(stdout),
		Stderr:   truncate(stderr),
		ExitCode: exitCode,
	})
}

func (t *Trace) AddAdapterEvent(name, status string, request, result interface{}, err error) {
	event := TraceEvent{Type: "adapter", Name: name, Request: request, Result: result, Status: status}
	if err != nil {
		event.Error = map[string]string{"class": ErrorClass(err), "message": err.Error()}
	}
	t.Events = append(t.Events, event)
}

func (t *Trace) SetFinal(status string, output interface{}, message string) {
	t.Final = TraceFinal{Status: status, Output: output, Error: message}
}

func WriteTrace(path string, trace *Trace) error {
	if err := ensureParent(path); err != nil {
		return WrapError(ErrDriverError, "create trace directory", err)
	}
	raw, err := json.MarshalIndent(trace, "", "  ")
	if err != nil {
		return WrapError(ErrDriverError, "encode trace", err)
	}
	if err := os.WriteFile(path, append(raw, '\n'), 0o600); err != nil {
		return WrapError(ErrDriverError, "write trace", err)
	}
	return nil
}

func ReadTrace(path string) (*Trace, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, WrapError(ErrNotFound, "read trace", err)
	}
	var trace Trace
	if err := json.Unmarshal(raw, &trace); err != nil {
		return nil, WrapError(ErrInvalidInput, "decode trace", err)
	}
	if err := ValidateTrace(trace); err != nil {
		return nil, err
	}
	return &trace, nil
}

func newRunID() string {
	var bytes [16]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return hex.EncodeToString([]byte(time.Now().UTC().Format("20060102150405.000000000")))
	}
	return hex.EncodeToString(bytes[:])
}

func truncate(value string) string {
	const limit = 4000
	if len(value) <= limit {
		return value
	}
	return value[:limit] + "...<truncated>"
}
