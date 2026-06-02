package tracing

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

var (
	ErrSessionClosed       = errors.New("session_closed")
	ErrTraceNotApplicable  = errors.New("trace_not_applicable")
	ErrNotEnrolled         = errors.New("not_enrolled")
	ErrForbidden           = errors.New("forbidden")
	ErrSessionNotFound     = errors.New("session_not_found")
)

const redisJobStream = "nextlab:sessions:jobs"

type Service struct {
	repo *Repository
	rdb  *redis.Client
}

func NewService(repo *Repository, rdb *redis.Client) *Service {
	return &Service{repo: repo, rdb: rdb}
}

// ─── US1 — Session lifecycle ──────────────────────────────────────────────────

// OpenSession creates or retrieves an active exam session.
// Returns (session, isNew, error). isNew is true when the session was just created.
func (s *Service) OpenSession(ctx context.Context, studentID, actividadID uuid.UUID) (*ExamSession, bool, error) {
	traceEnabled, enrolled, err := s.repo.GetActividadTraceInfo(ctx, actividadID, studentID)
	if err != nil {
		return nil, false, err
	}
	if !enrolled {
		return nil, false, ErrNotEnrolled
	}
	if !traceEnabled {
		return nil, false, ErrTraceNotApplicable
	}

	if err := s.repo.InsertSessionIdempotent(ctx, studentID, actividadID); err != nil {
		return nil, false, err
	}

	session, err := s.repo.GetSessionByStudentActivity(ctx, studentID, actividadID)
	if err != nil {
		return nil, false, err
	}
	if session == nil {
		return nil, false, ErrSessionNotFound
	}
	if session.Status != "active" {
		return nil, false, ErrSessionClosed
	}

	// Treat the session as new if it was started within the last 2 seconds
	isNew := time.Since(session.StartedAt) < 2*time.Second
	return session, isNew, nil
}

// ─── US2 — Event capture ──────────────────────────────────────────────────────

// AppendEvent stores an event with a chained SHA-256 hash.
// The hash is computed inside the repository TX using the DB-assigned timestamp.
func (s *Service) AppendEvent(ctx context.Context, sessionID uuid.UUID, eventType string, metadata map[string]any) (*TraceAck, error) {
	// Strip clipboard content before storing (FR-008)
	if eventType == "clipboard_copy" || eventType == "clipboard_paste" {
		delete(metadata, "content")
		delete(metadata, "text")
		delete(metadata, "data")
	}

	canonicalMeta := canonicalJSON(metadata)
	metaBytes, _ := json.Marshal(metadata)
	if string(metaBytes) == "null" {
		metaBytes = []byte("{}")
	}

	hashFn := func(seq int, prevHash string, receivedAt time.Time) string {
		return computeEventHash(sessionID, seq, eventType, receivedAt.UnixNano(), canonicalMeta, prevHash)
	}

	result, err := s.repo.AppendEvent(ctx, sessionID, eventType, metaBytes, hashFn)
	if err != nil {
		return nil, err
	}

	return &TraceAck{
		Type:        "ack",
		SequenceNum: result.SequenceNum,
		ReceivedAt:  result.ReceivedAt,
	}, nil
}

// ─── US3 — Session completion + job enqueueing ────────────────────────────────

// CloseSessionWithSubmission closes the session as completed and enqueues an async job.
// Silent no-op if no active session exists (student used raw API without session).
func (s *Service) CloseSessionWithSubmission(ctx context.Context, studentID, actividadID, submissionID uuid.UUID) {
	session, err := s.repo.GetSessionByStudentActivity(ctx, studentID, actividadID)
	if err != nil || session == nil || session.Status != "active" {
		return
	}

	if err := s.repo.CloseSession(ctx, session.ID, &submissionID, "completed"); err != nil {
		slog.ErrorContext(ctx, "failed to close session", "session_id", session.ID, "err", err)
		return
	}
	// Append session_close trace event
	_, _ = s.AppendEvent(ctx, session.ID, "session_close", nil)

	s.EnqueueJobForSession(ctx, session.ID, submissionID)
}

// EnqueueJobForSession inserts a worker_job row then XADDs to Redis Streams.
// On XADD failure the job row stays in 'pending' and is recovered by the worker sweep.
func (s *Service) EnqueueJobForSession(ctx context.Context, sessionID, submissionID uuid.UUID) {
	jobID, err := s.repo.InsertWorkerJob(ctx, sessionID, submissionID)
	if err != nil {
		slog.ErrorContext(ctx, "failed to insert worker_job", "session_id", sessionID, "err", err)
		return
	}

	streamID, err := s.rdb.XAdd(ctx, &redis.XAddArgs{
		Stream: redisJobStream,
		Values: map[string]any{
			"job_id":        jobID.String(),
			"session_id":    sessionID.String(),
			"submission_id": submissionID.String(),
		},
	}).Result()
	if err != nil {
		slog.ErrorContext(ctx, "xadd failed; job will be re-enqueued by sweep",
			"job_id", jobID, "err", err)
		return
	}

	if err := s.repo.SetJobStreamID(ctx, jobID, streamID); err != nil {
		slog.ErrorContext(ctx, "failed to set job stream_id", "job_id", jobID, "err", err)
	}
}

// CloseSessionAbandoned marks a session as abandoned (connection closed without session_close).
func (s *Service) CloseSessionAbandoned(ctx context.Context, sessionID uuid.UUID) {
	if err := s.repo.CloseSession(ctx, sessionID, nil, "abandoned"); err != nil {
		slog.ErrorContext(ctx, "failed to mark session abandoned", "session_id", sessionID, "err", err)
	}
}

// CloseSessionDeadline marks a session as deadline_closed (called by worker sweep).
func (s *Service) CloseSessionDeadline(ctx context.Context, sessionID uuid.UUID) error {
	return s.repo.CloseSession(ctx, sessionID, nil, "deadline_closed")
}

// CloseSessionTimeout marks a session as timed_out (called by worker sweep).
func (s *Service) CloseSessionTimeout(ctx context.Context, sessionID uuid.UUID) error {
	return s.repo.CloseSession(ctx, sessionID, nil, "timed_out")
}

// GetOrphanedJobs returns jobs stuck in pending/queued for >60 seconds.
func (s *Service) GetOrphanedJobs(ctx context.Context) ([]WorkerJob, error) {
	return s.repo.GetOrphanedJobs(ctx)
}

// MarkJobDeadIfExhausted looks up the job by ID and marks it dead if attempt_count >= max_retries.
// Returns true if the job was marked dead, false if it should still be retried.
func (s *Service) MarkJobDeadIfExhausted(ctx context.Context, jobID uuid.UUID, errMsg string) (bool, error) {
	var attemptCount, maxRetries int
	err := s.repo.db.QueryRow(ctx,
		`SELECT attempt_count, max_retries FROM worker_jobs WHERE id = $1`, jobID,
	).Scan(&attemptCount, &maxRetries)
	if err != nil {
		return false, err
	}
	if attemptCount >= maxRetries {
		return true, s.repo.MarkJobDead(ctx, jobID, errMsg)
	}
	return false, nil
}

// GetDeadlinedSessions returns active sessions whose activity deadline has passed.
func (s *Service) GetDeadlinedSessions(ctx context.Context) ([]ExamSession, error) {
	return s.repo.GetDeadlinedSessions(ctx)
}

// GetTimedOutSessions returns active sessions with no heartbeat for >5 minutes.
func (s *Service) GetTimedOutSessions(ctx context.Context) ([]ExamSession, error) {
	return s.repo.GetTimedOutSessions(ctx)
}

// ─── US4 — Hash chain verification ───────────────────────────────────────────

// VerifyChain recomputes the full SHA-256 chain and reports whether it is intact.
func (s *Service) VerifyChain(ctx context.Context, sessionID uuid.UUID) (valid bool, eventCount int, firstInvalid int) {
	events, err := s.repo.GetAllEvents(ctx, sessionID)
	if err != nil || len(events) == 0 {
		return true, 0, 0
	}

	prevHash := fmt.Sprintf("%x", sha256.Sum256([]byte(sessionID.String())))
	for _, e := range events {
		meta := canonicalJSON(e.Metadata)
		computed := computeEventHash(sessionID, e.SequenceNum, e.EventType, e.ReceivedAt.UnixNano(), meta, prevHash)
		if computed != e.EventHash {
			return false, len(events), e.SequenceNum
		}
		prevHash = e.EventHash
	}
	return true, len(events), 0
}

// GetTraceBySubmission retrieves the session and events for a submission, role-scoped.
func (s *Service) GetTraceBySubmission(ctx context.Context, submissionID, callerID uuid.UUID, role string) (*ExamSession, []TraceEvent, error) {
	return s.repo.GetTraceBySubmission(ctx, submissionID, callerID, role)
}

// ─── Hash helpers ─────────────────────────────────────────────────────────────

// computeEventHash computes the SHA-256 event hash per the chain formula.
func computeEventHash(sessionID uuid.UUID, seq int, eventType string, receivedAtNano int64, canonicalMeta string, prevHash string) string {
	raw := fmt.Sprintf("%s|%d|%s|%d|%s|%s", sessionID, seq, eventType, receivedAtNano, canonicalMeta, prevHash)
	return fmt.Sprintf("%x", sha256.Sum256([]byte(raw)))
}

// canonicalJSON serializes a map with sorted keys and no whitespace.
// Returns "{}" for nil or empty maps.
func canonicalJSON(m map[string]any) string {
	if len(m) == 0 {
		return "{}"
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	b := make([]byte, 0, 64)
	b = append(b, '{')
	for i, k := range keys {
		kb, _ := json.Marshal(k)
		vb, _ := json.Marshal(m[k])
		b = append(b, kb...)
		b = append(b, ':')
		b = append(b, vb...)
		if i < len(keys)-1 {
			b = append(b, ',')
		}
	}
	b = append(b, '}')
	return string(b)
}

// jsonUnmarshal is a helper alias used in repository.go scan.
func jsonUnmarshal(data []byte, v any) error {
	return json.Unmarshal(data, v)
}

// jsonMarshal is a helper alias used in repository.go batch insert.
func jsonMarshal(v any) ([]byte, error) {
	return json.Marshal(v)
}
