package detection

import (
	"time"

	"github.com/Elder2003/nextlab-api/internal/config"
	tracing "github.com/Elder2003/nextlab-api/internal/tracing"
)

type SuspiciousSequence struct {
	EventA     string  `json:"event_a"`
	EventB     string  `json:"event_b"`
	GapSeconds float64 `json:"gap_seconds"`
	WeightBonus float64 `json:"weight_bonus"`
}

type TemporalMetrics struct {
	SessionDurationMin  float64              `json:"session_duration_min"`
	EventsPerMinute     float64              `json:"events_per_minute"`
	BurstWindows        int                  `json:"burst_windows"`
	MaxEventsInWindow   int                  `json:"max_events_in_window"`
	SuspiciousSequences []SuspiciousSequence `json:"suspicious_sequences"`
	// Baseline z-score fields (populated by detection service, not heuristic engine)
	BaselineAvailable     bool    `json:"baseline_available"`
	BaselineSessionsCount int     `json:"baseline_sessions_count,omitempty"`
	ZScoreTabBlurs        float64 `json:"z_score_tab_blurs,omitempty"`
	ZScoreClipboard       float64 `json:"z_score_clipboard,omitempty"`
}

const heartbeatGapThreshold = 30 * time.Second

type HeuristicEngine struct {
	TabWeight     int
	ClipWeight    int
	GapWeight     int
	HighThreshold int
}

func NewHeuristicEngine(cfg *config.Config) *HeuristicEngine {
	return &HeuristicEngine{
		TabWeight:     cfg.HeuristicTabWeight,
		ClipWeight:    cfg.HeuristicClipWeight,
		GapWeight:     cfg.HeuristicGapWeight,
		HighThreshold: cfg.HeuristicHighThreshold,
	}
}

// Score computes an additive suspicion score (0–100) and a list of flagged categories
// from the session's trace events.
func (h *HeuristicEngine) Score(events []tracing.TraceEvent) (score int, flagged []string) {
	var tabCount, clipCount, gapCount int

	// Count tab blur events
	for _, e := range events {
		switch e.EventType {
		case "tab_blur":
			tabCount++
		case "clipboard_copy", "clipboard_paste":
			clipCount++
		}
	}

	// Count heartbeat gaps > 30s by iterating in sequence order (already ordered ASC)
	var lastHeartbeat *time.Time
	for _, e := range events {
		if e.EventType == "heartbeat" {
			if lastHeartbeat != nil && e.ReceivedAt.Sub(*lastHeartbeat) > heartbeatGapThreshold {
				gapCount++
			}
			t := e.ReceivedAt
			lastHeartbeat = &t
		}
	}

	raw := tabCount*h.TabWeight + clipCount*h.ClipWeight + gapCount*h.GapWeight
	if raw > 100 {
		raw = 100
	}
	score = raw

	if tabCount > 0 {
		flagged = append(flagged, "tab_activity")
	}
	if clipCount > 0 {
		flagged = append(flagged, "clipboard_activity")
	}
	if gapCount > 0 {
		flagged = append(flagged, "heartbeat_gap")
	}
	if flagged == nil {
		flagged = []string{}
	}
	return score, flagged
}

// ScoreWithTemporal extends Score with temporal distribution metrics and suspicious sequence detection.
func (h *HeuristicEngine) ScoreWithTemporal(events []tracing.TraceEvent) (score int, flagged []string, temporal TemporalMetrics) {
	score, flagged = h.Score(events)
	if len(events) == 0 {
		temporal.SuspiciousSequences = []SuspiciousSequence{}
		return
	}

	durationMin := events[len(events)-1].ReceivedAt.Sub(events[0].ReceivedAt).Minutes()
	if durationMin < 0.25 {
		durationMin = 0.25
	}
	temporal.SessionDurationMin = durationMin
	temporal.EventsPerMinute = float64(len(events)) / durationMin

	// Burst windows: 60-second sliding windows with >5 non-heartbeat events
	windowStart := events[0].ReceivedAt
	windowCount := 0
	maxInWindow := 0
	for _, e := range events {
		if e.EventType == "heartbeat" {
			continue
		}
		if e.ReceivedAt.Sub(windowStart) > 60*time.Second {
			if windowCount > maxInWindow {
				maxInWindow = windowCount
			}
			if windowCount > 5 {
				temporal.BurstWindows++
			}
			windowStart = e.ReceivedAt
			windowCount = 0
		}
		windowCount++
	}
	// Final window
	if windowCount > maxInWindow {
		maxInWindow = windowCount
	}
	if windowCount > 5 {
		temporal.BurstWindows++
	}
	temporal.MaxEventsInWindow = maxInWindow

	// Suspicious sequences: tab_blur → clipboard_paste/copy within 5 seconds
	temporal.SuspiciousSequences = []SuspiciousSequence{}
	clipboardEvents := map[string]bool{"clipboard_paste": true, "clipboard_copy": true}
	for i := 0; i+1 < len(events); i++ {
		if events[i].EventType == "tab_blur" && clipboardEvents[events[i+1].EventType] {
			gap := events[i+1].ReceivedAt.Sub(events[i].ReceivedAt).Seconds()
			if gap < 5.0 {
				temporal.SuspiciousSequences = append(temporal.SuspiciousSequences, SuspiciousSequence{
					EventA:      "tab_blur",
					EventB:      events[i+1].EventType,
					GapSeconds:  gap,
					WeightBonus: float64(h.ClipWeight),
				})
				score += h.ClipWeight
				if score > 100 {
					score = 100
				}
			}
		}
	}

	return score, flagged, temporal
}
