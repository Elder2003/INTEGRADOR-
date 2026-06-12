package mldetection

import (
	"fmt"
	"sync/atomic"
	"unsafe"
)

// ONNXRuntime wraps the ONNX inference session.
// In production (Linux + CGO), swap the stub methods below with the
// real yalue/onnxruntime_go calls. The interface is stable.
type ONNXRuntime struct {
	ptr          unsafe.Pointer // holds *ort.Session when CGO is available
	featureNames []string
	filePath     string
	active       atomic.Bool
}

// Load creates a runtime from an ONNX model file.
// Stub implementation: stores metadata only; actual inference returns neutral score.
func Load(filePath string, featureNames []string) (*ONNXRuntime, error) {
	r := &ONNXRuntime{
		featureNames: featureNames,
		filePath:     filePath,
	}
	r.active.Store(true)
	return r, nil
}

// Predict runs inference and returns fraud probability (0.0–1.0).
// Stub: returns 0.5 (neutral) until the CGO runtime is linked in.
func (r *ONNXRuntime) Predict(tabBlursRate, clipboardRate, heartbeatGapAvg, durationMin, burstWindows, eventsPerMin float64) (float64, error) {
	if r == nil || !r.active.Load() {
		return 0, fmt.Errorf("mldetection: no model loaded")
	}
	// Neutral score — replace with onnxruntime_go inference when CGO is available.
	return 0.5, nil
}

// HotSwap atomically replaces the active model with a new one.
func (r *ONNXRuntime) HotSwap(newFilePath string, newFeatureNames []string) error {
	r.filePath = newFilePath
	r.featureNames = newFeatureNames
	r.active.Store(true)
	return nil
}

// FilePath returns the current model file path.
func (r *ONNXRuntime) FilePath() string {
	return r.filePath
}
