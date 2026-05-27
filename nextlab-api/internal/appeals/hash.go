package appeals

import (
	"crypto/sha256"
	"fmt"
	"strconv"
	"strings"
)

// ComputeHash returns the canonical SHA-256 event hash for an audit event.
// canonical = occurred_at_iso8601|event_type|sequence|payload_json|prev_hash_or_empty
func ComputeHash(occurredAt, eventType string, sequence int, payloadJSON, prevHash string) string {
	canonical := strings.Join([]string{
		occurredAt,
		eventType,
		strconv.Itoa(sequence),
		payloadJSON,
		prevHash,
	}, "|")
	sum := sha256.Sum256([]byte(canonical))
	return fmt.Sprintf("%x", sum)
}
