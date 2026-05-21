package rag

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	pdfapi "github.com/pdfcpu/pdfcpu/pkg/api"
)

// ExtractText reads bytes and returns plain text based on file extension.
// Supported extensions: .pdf, .docx, .txt — returns an error for others.
func ExtractText(data []byte, ext string) (string, error) {
	switch strings.ToLower(ext) {
	case ".txt":
		return string(data), nil
	case ".docx":
		return extractDocx(data)
	case ".pdf":
		return extractPDF(data)
	default:
		return "", fmt.Errorf("unsupported file type")
	}
}

// extractPDF writes the PDF to a temp file, uses pdfcpu to extract content
// streams, then parses those streams for text operators.
func extractPDF(data []byte) (string, error) {
	tmpIn, err := os.CreateTemp("", "rag-*.pdf")
	if err != nil {
		return "", fmt.Errorf("create temp pdf: %w", err)
	}
	defer os.Remove(tmpIn.Name())
	if _, err := tmpIn.Write(data); err != nil {
		tmpIn.Close()
		return "", err
	}
	tmpIn.Close()

	tmpOut, err := os.MkdirTemp("", "rag-pdf-out-*")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(tmpOut)

	// nil configuration uses pdfcpu defaults (ValidationRelaxed)
	if err := pdfapi.ExtractContentFile(tmpIn.Name(), tmpOut, nil, nil); err != nil {
		return "", fmt.Errorf("pdfcpu extract: %w", err)
	}

	entries, err := os.ReadDir(tmpOut)
	if err != nil {
		return "", err
	}

	var sb strings.Builder
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(tmpOut, e.Name()))
		if err != nil {
			continue
		}
		sb.WriteString(parseContentStream(raw))
		sb.WriteRune('\n')
	}
	return normalizeSpaces(sb.String()), nil
}

// parseContentStream extracts visible text from a PDF content stream by
// finding all literal-string arguments to Tj, TJ, and ' operators.
func parseContentStream(data []byte) string {
	var sb strings.Builder
	i := 0
	for i < len(data) {
		if data[i] == '(' {
			s, n := readPDFLiteral(data[i:])
			sb.WriteString(s)
			sb.WriteString(" ")
			i += n
		} else {
			i++
		}
	}
	return sb.String()
}

// readPDFLiteral reads a PDF literal string starting at data[0]=='(' and
// returns the decoded content and the number of bytes consumed.
func readPDFLiteral(data []byte) (string, int) {
	var sb strings.Builder
	i := 1 // skip opening '('
	depth := 1
	for i < len(data) && depth > 0 {
		switch {
		case data[i] == '\\' && i+1 < len(data):
			switch data[i+1] {
			case 'n':
				sb.WriteByte('\n')
			case 'r':
				sb.WriteByte('\r')
			case 't':
				sb.WriteByte('\t')
			case '(':
				sb.WriteByte('(')
			case ')':
				sb.WriteByte(')')
			case '\\':
				sb.WriteByte('\\')
			default:
				sb.WriteByte(data[i+1])
			}
			i += 2
		case data[i] == '(':
			depth++
			sb.WriteByte('(')
			i++
		case data[i] == ')':
			depth--
			if depth > 0 {
				sb.WriteByte(')')
			}
			i++
		default:
			if data[i] >= 32 { // skip control chars
				sb.WriteByte(data[i])
			}
			i++
		}
	}
	return sb.String(), i
}

// extractDocx reads a .docx ZIP archive and extracts plain text from
// word/document.xml by collecting all <w:t> element contents.
func extractDocx(data []byte) (string, error) {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return "", fmt.Errorf("open docx zip: %w", err)
	}

	for _, f := range zr.File {
		if f.Name != "word/document.xml" {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return "", err
		}
		defer rc.Close()
		return parseWordXML(rc)
	}
	return "", fmt.Errorf("word/document.xml not found in docx archive")
}

// parseWordXML collects text from all <w:t> elements in an OOXML document body.
func parseWordXML(r io.Reader) (string, error) {
	decoder := xml.NewDecoder(r)
	var sb strings.Builder
	inText := false
	for {
		tok, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			break // return partial text on parse error
		}
		switch t := tok.(type) {
		case xml.StartElement:
			if t.Name.Local == "t" {
				inText = true
			}
		case xml.EndElement:
			if t.Name.Local == "t" {
				inText = false
				sb.WriteByte(' ')
			}
		case xml.CharData:
			if inText {
				sb.Write([]byte(t))
			}
		}
	}
	return strings.TrimSpace(sb.String()), nil
}

var multiSpaceRe = regexp.MustCompile(`\s{2,}`)

func normalizeSpaces(s string) string {
	return strings.TrimSpace(multiSpaceRe.ReplaceAllString(s, " "))
}

// FetchURL fetches a URL and returns its text content.
// For YouTube URLs it uses the timedtext API; for other URLs it strips HTML tags.
// Returns an error on network failure or non-2xx response.
func FetchURL(ctx context.Context, rawURL string) (string, error) {
	client := &http.Client{Timeout: 30 * time.Second}

	target := rawURL
	if isYouTubeURL(rawURL) {
		id := extractVideoID(rawURL)
		if id == "" {
			return "", fmt.Errorf("could not extract YouTube video ID from %s", rawURL)
		}
		target = "https://www.youtube.com/api/timedtext?v=" + id + "&lang=en&fmt=vtt"
	}

	req, err := http.NewRequestWithContext(ctx, "GET", target, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "nextlab-rag/1.0")

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("URL returned HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20)) // 2MB max
	if err != nil {
		return "", err
	}

	if isYouTubeURL(rawURL) {
		return stripVTTTimestamps(string(body)), nil
	}
	return stripHTML(string(body)), nil
}

func isYouTubeURL(u string) bool {
	return strings.Contains(u, "youtube.com/watch") || strings.Contains(u, "youtu.be/")
}

func extractVideoID(u string) string {
	if idx := strings.Index(u, "v="); idx >= 0 {
		id := u[idx+2:]
		if end := strings.IndexAny(id, "&# \t"); end >= 0 {
			id = id[:end]
		}
		return id
	}
	if idx := strings.LastIndex(u, "/"); idx >= 0 {
		id := u[idx+1:]
		if end := strings.IndexAny(id, "?& \t"); end >= 0 {
			id = id[:end]
		}
		return id
	}
	return ""
}

var htmlTagRe = regexp.MustCompile(`<[^>]+>`)
var vttTimestampRe = regexp.MustCompile(`\d{2}:\d{2}:\d{2}\.\d{3} --> \d{2}:\d{2}:\d{2}\.\d{3}[^\n]*\n`)

func stripHTML(s string) string {
	return normalizeSpaces(htmlTagRe.ReplaceAllString(s, " "))
}

func stripVTTTimestamps(s string) string {
	s = vttTimestampRe.ReplaceAllString(s, "")
	lines := strings.Split(s, "\n")
	var out []string
	for _, l := range lines {
		l = strings.TrimSpace(l)
		if l == "" || l == "WEBVTT" || strings.HasPrefix(l, "NOTE") {
			continue
		}
		out = append(out, l)
	}
	return strings.Join(out, " ")
}
