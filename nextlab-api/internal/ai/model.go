package ai

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// ErrModeUnchanged is returned by SetMode when the requested mode is already active.
var ErrModeUnchanged = errors.New("mode unchanged")

// RouterConfig is a single append-only row in router_configs.
// Current mode = most recent row ordered by changed_at DESC.
type RouterConfig struct {
	ID        uuid.UUID  `db:"id"`
	Mode      string     `db:"mode"`
	PrevMode  *string    `db:"prev_mode"`
	ChangedBy *uuid.UUID `db:"changed_by"`
	ChangedAt time.Time  `db:"changed_at"`
}

// AIRequestLog is an immutable audit record for one routing dispatch.
// Prompt and completion content are NEVER stored (FR-011).
type AIRequestLog struct {
	ID             uuid.UUID `db:"id"`
	RequestID      uuid.UUID `db:"request_id"`
	Mode           string    `db:"mode"`
	BackendUsed    string    `db:"backend_used"`
	FallbackUsed   bool      `db:"fallback_used"`
	FallbackReason *string   `db:"fallback_reason"`
	LatencyMs      int       `db:"latency_ms"`
	Status         string    `db:"status"`
	ErrorDetail    *string   `db:"error_detail"`
	LoggedAt       time.Time `db:"logged_at"`
}

// ChatMessage is a single message in an OpenAI-compatible chat request.
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatRequest is the body sent to an OpenAI-compatible /v1/chat/completions endpoint.
type ChatRequest struct {
	Model     string        `json:"model"`
	Messages  []ChatMessage `json:"messages"`
	MaxTokens int           `json:"max_tokens,omitempty"`
}

// ChatChoice is one completion candidate.
type ChatChoice struct {
	Index        int         `json:"index"`
	Message      ChatMessage `json:"message"`
	FinishReason string      `json:"finish_reason"`
}

// ChatUsage holds token accounting returned by the backend.
type ChatUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// ChatResponse is the OpenAI-compatible response returned to the caller.
type ChatResponse struct {
	ID      string       `json:"id"`
	Object  string       `json:"object"`
	Created int64        `json:"created"`
	Model   string       `json:"model"`
	Choices []ChatChoice `json:"choices"`
	Usage   *ChatUsage   `json:"usage,omitempty"`
}

// DispatchRequest is the router's internal representation of a completion request.
type DispatchRequest struct {
	RequestID uuid.UUID
	Model     string
	Messages  []ChatMessage
	MaxTokens int
}

func (d DispatchRequest) toChatRequest(defaultModel string) ChatRequest {
	model := d.Model
	if model == "" {
		model = defaultModel
	}
	maxTokens := d.MaxTokens
	if maxTokens == 0 {
		maxTokens = 512
	}
	return ChatRequest{
		Model:     model,
		Messages:  d.Messages,
		MaxTokens: maxTokens,
	}
}

func ptr[T any](v T) *T { return &v }
