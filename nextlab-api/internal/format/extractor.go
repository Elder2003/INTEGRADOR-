package format

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"io"
	"os"
	"path/filepath"
	"strings"

	pdfapi "github.com/pdfcpu/pdfcpu/pkg/api"
)

// ParseFile detects the file type via magic bytes and extracts structural metadata.
// A *ParseError is returned for known failure modes (password, unsupported, empty, corrupt).
// System errors are returned as ordinary errors.
func ParseFile(data []byte) (FileInfo, error) {
	if len(data) == 0 {
		return FileInfo{}, &ParseError{Reason: "empty_file"}
	}

	switch detectType(data) {
	case "pdf":
		return parsePDF(data)
	case "docx":
		return parseDocx(data)
	default:
		return FileInfo{FileType: "unsupported"}, &ParseError{Reason: "unsupported_file_type"}
	}
}

func detectType(data []byte) string {
	if len(data) < 4 {
		return "unsupported"
	}
	if bytes.HasPrefix(data, []byte("%PDF")) {
		return "pdf"
	}
	if bytes.HasPrefix(data, []byte("PK\x03\x04")) {
		zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
		if err != nil {
			return "unsupported"
		}
		for _, f := range zr.File {
			if f.Name == "[Content_Types].xml" {
				return "docx"
			}
		}
	}
	return "unsupported"
}

func parsePDF(data []byte) (FileInfo, error) {
	// Page count via pdfcpu (in-memory reader)
	pageCount, err := pdfapi.PageCount(bytes.NewReader(data), nil)
	if err != nil {
		msg := strings.ToLower(err.Error())
		if strings.Contains(msg, "password") || strings.Contains(msg, "encrypt") {
			return FileInfo{FileType: "pdf"}, &ParseError{Reason: "password_protected"}
		}
		return FileInfo{FileType: "pdf"}, &ParseError{Reason: "corrupt_file"}
	}

	// Text extraction for word/char counts
	tmpIn, err := os.CreateTemp("", "fmt-*.pdf")
	if err != nil {
		return FileInfo{FileType: "pdf", PageCount: pageCount}, nil
	}
	defer os.Remove(tmpIn.Name())
	if _, err := tmpIn.Write(data); err != nil {
		tmpIn.Close()
		return FileInfo{FileType: "pdf", PageCount: pageCount}, nil
	}
	tmpIn.Close()

	tmpOut, err := os.MkdirTemp("", "fmt-pdf-out-*")
	if err != nil {
		return FileInfo{FileType: "pdf", PageCount: pageCount}, nil
	}
	defer os.RemoveAll(tmpOut)

	var wordCount, charCount int
	if err := pdfapi.ExtractContentFile(tmpIn.Name(), tmpOut, nil, nil); err == nil {
		entries, _ := os.ReadDir(tmpOut)
		var sb strings.Builder
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			raw, err := os.ReadFile(filepath.Join(tmpOut, e.Name()))
			if err == nil {
				sb.WriteString(parsePDFContentStream(raw))
				sb.WriteRune(' ')
			}
		}
		text := strings.TrimSpace(sb.String())
		wordCount = len(strings.Fields(text))
		charCount = len([]rune(strings.ReplaceAll(text, " ", "")))
	}

	return FileInfo{
		FileType:  "pdf",
		PageCount: pageCount,
		WordCount: wordCount,
		CharCount: charCount,
	}, nil
}

// parsePDFContentStream extracts visible text from a PDF content stream by
// finding all literal-string arguments to text operators.
func parsePDFContentStream(data []byte) string {
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

func readPDFLiteral(data []byte) (string, int) {
	var sb strings.Builder
	i := 1
	depth := 1
	for i < len(data) && depth > 0 {
		switch {
		case data[i] == '\\' && i+1 < len(data):
			switch data[i+1] {
			case 'n':
				sb.WriteByte('\n')
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
			if data[i] >= 32 {
				sb.WriteByte(data[i])
			}
			i++
		}
	}
	return sb.String(), i
}

type docxAppProps struct {
	XMLName    xml.Name `xml:"Properties"`
	Pages      int      `xml:"Pages"`
	Words      int      `xml:"Words"`
	Characters int      `xml:"Characters"`
}

type docxCoreProps struct {
	Creator string `xml:"creator"`
	Created string `xml:"created"`
}

func parseDocx(data []byte) (FileInfo, error) {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return FileInfo{FileType: "docx"}, &ParseError{Reason: "corrupt_file"}
	}

	info := FileInfo{FileType: "docx", DocMetadata: map[string]any{}}

	for _, f := range zr.File {
		switch f.Name {
		case "docProps/app.xml":
			parseDocxApp(f, &info)
		case "docProps/core.xml":
			parseDocxCore(f, &info)
		}
	}

	// Fallback: count words from document text when app.xml had no word count
	if info.WordCount == 0 {
		for _, f := range zr.File {
			if f.Name == "word/document.xml" {
				parseDocxWordCount(f, &info)
				break
			}
		}
	}

	return info, nil
}

func parseDocxApp(f *zip.File, info *FileInfo) {
	rc, err := f.Open()
	if err != nil {
		return
	}
	defer rc.Close()
	var props docxAppProps
	if err := xml.NewDecoder(rc).Decode(&props); err != nil {
		return
	}
	info.PageCount = props.Pages
	info.WordCount = props.Words
	info.CharCount = props.Characters
}

func parseDocxCore(f *zip.File, info *FileInfo) {
	rc, err := f.Open()
	if err != nil {
		return
	}
	defer rc.Close()
	var props docxCoreProps
	dec := xml.NewDecoder(rc)
	dec.DefaultSpace = ""
	if err := dec.Decode(&props); err != nil {
		return
	}
	if info.DocMetadata == nil {
		info.DocMetadata = map[string]any{}
	}
	if props.Creator != "" {
		info.DocMetadata["author"] = props.Creator
	}
	if props.Created != "" {
		info.DocMetadata["created_at"] = props.Created
	}
}

func parseDocxWordCount(f *zip.File, info *FileInfo) {
	rc, err := f.Open()
	if err != nil {
		return
	}
	defer rc.Close()
	dec := xml.NewDecoder(rc)
	var sb strings.Builder
	inText := false
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			break
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
	text := strings.TrimSpace(sb.String())
	info.WordCount = len(strings.Fields(text))
	info.CharCount = len([]rune(strings.ReplaceAll(text, " ", "")))
}
