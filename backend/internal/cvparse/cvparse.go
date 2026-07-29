// Package cvparse extracts plain text from an uploaded CV.
//
// Everything happens in-process with no external service: PDF via a pure-Go
// reader, DOCX via the standard library (it is a zip of XML), and plain text
// passed through. That keeps CV upload working regardless of whether scoring
// is configured, and means your CV — which is personal data — is never sent to
// a third party just to be read.
package cvparse

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/ledongthuc/pdf"
)

// MaxFileBytes caps an upload. A CV is a few pages; anything far larger is a
// mistake or an attack, and both deserve a clear rejection rather than a
// container OOM.
const MaxFileBytes = 10 << 20 // 10 MiB

// ErrUnsupported is returned for a file type we cannot read.
var ErrUnsupported = errors.New("unsupported file type")

// ErrNoText means the file parsed but yielded nothing useful — most often a
// scanned PDF, which is images of text with no text layer.
var ErrNoText = errors.New("no extractable text found")

// Extract reads a CV and returns its plain text.
//
// filename is used only to pick a parser; the content is what is trusted.
func Extract(filename string, data []byte) (string, error) {
	if len(data) == 0 {
		return "", errors.New("file is empty")
	}
	if len(data) > MaxFileBytes {
		return "", fmt.Errorf("file is %d bytes, over the %d byte limit", len(data), MaxFileBytes)
	}

	var (
		text string
		err  error
	)

	// Sniff the content first — a .doc extension on a real .docx, or a PDF
	// saved as .txt, are both common enough to be worth handling.
	switch {
	case bytes.HasPrefix(data, []byte("%PDF")):
		text, err = extractPDF(data)
	case bytes.HasPrefix(data, []byte("PK\x03\x04")):
		// A zip: could be .docx, .odt, or something else entirely.
		text, err = extractDOCX(data)
	default:
		switch strings.ToLower(filepath.Ext(filename)) {
		case ".txt", ".md", ".text", "":
			if !utf8.Valid(data) {
				return "", fmt.Errorf("%w: file is not valid UTF-8 text", ErrUnsupported)
			}
			text = string(data)
		case ".doc":
			// Legacy binary .doc is a different format entirely (OLE compound
			// file) and is not worth a dependency. Saving as .docx or PDF is
			// one menu click.
			return "", fmt.Errorf("%w: legacy .doc is not supported — re-save as .docx or PDF", ErrUnsupported)
		default:
			return "", fmt.Errorf("%w: %s (supported: PDF, DOCX, TXT)", ErrUnsupported, filepath.Ext(filename))
		}
	}
	if err != nil {
		return "", err
	}

	text = tidy(text)
	if strings.TrimSpace(text) == "" {
		return "", ErrNoText
	}
	return text, nil
}

// extractPDF pulls the text layer out of a PDF.
func extractPDF(data []byte) (string, error) {
	r, err := pdf.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return "", fmt.Errorf("could not read PDF: %w", err)
	}

	var out strings.Builder
	// Page-by-page rather than GetPlainText on the whole file: a single
	// malformed page is common in CVs exported from design tools, and this way
	// it costs that page rather than the entire document.
	for i := 1; i <= r.NumPage(); i++ {
		page := r.Page(i)
		if page.V.IsNull() {
			continue
		}
		content, err := page.GetPlainText(nil)
		if err != nil {
			continue
		}
		out.WriteString(content)
		out.WriteString("\n")
	}

	if strings.TrimSpace(out.String()) == "" {
		return "", fmt.Errorf("%w: the PDF has no text layer, which usually means it is a scan or an export of images. Copy the text out and paste it instead", ErrNoText)
	}
	return out.String(), nil
}

// docxBody is the subset of word/document.xml we care about.
//
// Decoding the XML properly rather than stripping tags with a regex matters
// here: <w:p> paragraph boundaries are what keep "Flutter" and "Dart" on
// separate lines instead of being welded into "FlutterDart", which would stop
// the scorer matching either.
type docxBody struct {
	Paragraphs []struct {
		Runs []struct {
			Text string `xml:"t"`
		} `xml:"r"`
	} `xml:"body>p"`
}

// extractDOCX reads a .docx, which is a zip containing word/document.xml.
func extractDOCX(data []byte) (string, error) {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return "", fmt.Errorf("could not read DOCX: %w", err)
	}

	var doc *zip.File
	for _, f := range zr.File {
		if f.Name == "word/document.xml" {
			doc = f
			break
		}
	}
	if doc == nil {
		// A zip without word/document.xml is some other format — .odt and
		// .pages both look like zips too.
		return "", fmt.Errorf("%w: the file is a zip but not a Word document (no word/document.xml)", ErrUnsupported)
	}

	rc, err := doc.Open()
	if err != nil {
		return "", fmt.Errorf("open document.xml: %w", err)
	}
	defer rc.Close()

	xmlBytes, err := io.ReadAll(io.LimitReader(rc, MaxFileBytes))
	if err != nil {
		return "", fmt.Errorf("read document.xml: %w", err)
	}

	var body docxBody
	if err := xml.Unmarshal(xmlBytes, &body); err != nil {
		return "", fmt.Errorf("parse document.xml: %w", err)
	}

	var out strings.Builder
	for _, p := range body.Paragraphs {
		for _, r := range p.Runs {
			out.WriteString(r.Text)
		}
		out.WriteString("\n")
	}
	return out.String(), nil
}

var multiBlank = regexp.MustCompile(`\n{3,}`)

// tidy normalises extracted text: consistent line endings, no control
// characters, no runs of blank lines, no trailing spaces.
//
// PDF extraction in particular produces a lot of incidental whitespace, and a
// tidy CV is easier to eyeball in the editor to confirm the upload worked.
func tidy(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")

	// Strip control characters that survive some PDF exports; keep tab and
	// newline, which carry layout.
	s = strings.Map(func(r rune) rune {
		if r == '\n' || r == '\t' {
			return r
		}
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, s)

	lines := strings.Split(s, "\n")
	for i, l := range lines {
		lines[i] = strings.TrimRight(l, " \t")
	}
	s = strings.Join(lines, "\n")

	return strings.TrimSpace(multiBlank.ReplaceAllString(s, "\n\n"))
}
