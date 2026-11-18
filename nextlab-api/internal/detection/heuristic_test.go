package detection

import (
	"testing"
	"time"

	"github.com/Elder2003/nextlab-api/internal/config"
	tracing "github.com/Elder2003/nextlab-api/internal/tracing"
)

func TestHeuristicEngineScoreFlagsTabClipboardAndHeartbeatGap(t *testing.T) {
	engine := NewHeuristicEngine(&config.Config{
		HeuristicTabWeight:     10,
		HeuristicClipWeight:    25,
		HeuristicGapWeight:     15,
		HeuristicHighThreshold: 70,
	})
	now := time.Date(2026, 9, 23, 10, 0, 0, 0, time.UTC)

	score, flagged := engine.Score([]tracing.TraceEvent{
		{EventType: "heartbeat", ReceivedAt: now},
		{EventType: "tab_blur", ReceivedAt: now.Add(5 * time.Second)},
		{EventType: "clipboard_paste", ReceivedAt: now.Add(10 * time.Second)},
		{EventType: "heartbeat", ReceivedAt: now.Add(45 * time.Second)},
	})

	if score != 50 {
		t.Fatalf("score = %d, want 50", score)
	}
	wantFlags := []string{"tab_activity", "clipboard_activity", "heartbeat_gap"}
	if len(flagged) != len(wantFlags) {
		t.Fatalf("flagged = %v, want %v", flagged, wantFlags)
	}
	for i, want := range wantFlags {
		if flagged[i] != want {
			t.Fatalf("flagged[%d] = %q, want %q", i, flagged[i], want)
		}
	}
}

func TestHeuristicEngineScoreWithTemporalDetectsSuspiciousSequence(t *testing.T) {
	engine := NewHeuristicEngine(&config.Config{
		HeuristicTabWeight:     10,
		HeuristicClipWeight:    25,
		HeuristicGapWeight:     15,
		HeuristicHighThreshold: 70,
	})
	now := time.Date(2026, 9, 23, 10, 0, 0, 0, time.UTC)

	score, _, temporal := engine.ScoreWithTemporal([]tracing.TraceEvent{
		{EventType: "heartbeat", ReceivedAt: now},
		{EventType: "tab_blur", ReceivedAt: now.Add(10 * time.Second)},
		{EventType: "clipboard_copy", ReceivedAt: now.Add(12 * time.Second)},
	})

	if score != 60 {
		t.Fatalf("score = %d, want 60", score)
	}
	if len(temporal.SuspiciousSequences) != 1 {
		t.Fatalf("suspicious sequences = %d, want 1", len(temporal.SuspiciousSequences))
	}
	seq := temporal.SuspiciousSequences[0]
	if seq.EventA != "tab_blur" || seq.EventB != "clipboard_copy" {
		t.Fatalf("sequence = %s -> %s, want tab_blur -> clipboard_copy", seq.EventA, seq.EventB)
	}
	if seq.GapSeconds != 2 {
		t.Fatalf("gap seconds = %v, want 2", seq.GapSeconds)
	}
}
