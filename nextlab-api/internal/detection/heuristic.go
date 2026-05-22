package detection

import (
	"time"

	tracing "github.com/Elder2003/nextlab-api/internal/tracing"
	"github.com/Elder2003/nextlab-api/internal/config"
)

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
