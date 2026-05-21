package rag

import "strings"

// ChunkText splits text into overlapping chunks using a byte/4 token approximation.
// tokenWindow and overlap are approximate token counts (1 token ≈ 4 bytes).
func ChunkText(text string, tokenWindow, overlap int) []string {
	if len(text) == 0 {
		return nil
	}

	byteWindow := tokenWindow * 4
	byteOverlap := overlap * 4

	if byteWindow <= 0 {
		byteWindow = 2048
	}
	if byteOverlap < 0 || byteOverlap >= byteWindow {
		byteOverlap = 0
	}

	if len(text) <= byteWindow {
		return []string{strings.TrimSpace(text)}
	}

	var chunks []string
	start := 0
	for start < len(text) {
		end := start + byteWindow
		if end >= len(text) {
			chunks = append(chunks, strings.TrimSpace(text[start:]))
			break
		}

		// snap backward to the nearest whitespace to avoid splitting mid-word
		snap := end
		for snap > start && snap > end-100 && text[snap] != ' ' && text[snap] != '\n' && text[snap] != '\t' {
			snap--
		}
		if snap > start {
			end = snap
		}

		chunks = append(chunks, strings.TrimSpace(text[start:end]))

		step := byteWindow - byteOverlap
		if step <= 0 {
			step = byteWindow
		}
		start += step
	}

	return chunks
}
