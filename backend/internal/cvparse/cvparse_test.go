package cvparse

import (
	"archive/zip"
	"bytes"
	"errors"
	"strings"
	"testing"
)

// buildDOCX assembles a minimal but structurally real .docx.
func buildDOCX(t *testing.T, paragraphs ...string) []byte {
	t.Helper()

	var body strings.Builder
	body.WriteString(`<?xml version="1.0" encoding="UTF-8"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:body>`)
	for _, p := range paragraphs {
		body.WriteString(`<w:p><w:r><w:t>` + p + `</w:t></w:r></w:p>`)
	}
	body.WriteString(`</w:body></w:document>`)

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	f, err := zw.Create("word/document.xml")
	if err != nil {
		t.Fatalf("create zip entry: %v", err)
	}
	if _, err := f.Write([]byte(body.String())); err != nil {
		t.Fatalf("write zip entry: %v", err)
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("close zip: %v", err)
	}
	return buf.Bytes()
}

func TestExtractDOCX(t *testing.T) {
	data := buildDOCX(t,
		"Victor Kinyua — Mobile Engineer",
		"Four years building Flutter apps in Dart.",
		"Backend in Go with PostgreSQL.")

	got, err := Extract("cv.docx", data)
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	for _, want := range []string{"Victor Kinyua", "Flutter", "Dart", "Go", "PostgreSQL"} {
		if !strings.Contains(got, want) {
			t.Errorf("extracted text is missing %q:\n%s", want, got)
		}
	}
}

// Paragraph boundaries must survive. Without them "…in Dart." and "Backend in
// Go" weld into one line, and a skill at a boundary stops matching.
func TestExtractDOCXKeepsParagraphsSeparate(t *testing.T) {
	got, err := Extract("cv.docx", buildDOCX(t, "Flutter", "Dart"))
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if strings.Contains(got, "FlutterDart") {
		t.Errorf("paragraphs were welded together: %q", got)
	}
	if !strings.Contains(got, "Flutter\nDart") {
		t.Errorf("expected a newline between paragraphs, got %q", got)
	}
}

// A .docx whose runs are split mid-word (Word does this constantly, e.g. after
// a spell-check correction) must still reassemble into the whole word.
func TestExtractDOCXJoinsSplitRuns(t *testing.T) {
	docx := `<?xml version="1.0" encoding="UTF-8"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:body>
<w:p><w:r><w:t>Flut</w:t></w:r><w:r><w:t>ter</w:t></w:r><w:r><w:t> and Dart</w:t></w:r></w:p>
</w:body></w:document>`

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	f, _ := zw.Create("word/document.xml")
	_, _ = f.Write([]byte(docx))
	_ = zw.Close()

	got, err := Extract("cv.docx", buf.Bytes())
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if !strings.Contains(got, "Flutter and Dart") {
		t.Errorf("split runs were not joined: %q", got)
	}
}

func TestExtractPlainText(t *testing.T) {
	got, err := Extract("cv.txt", []byte("Flutter developer.\r\n\r\n\r\n\r\nDart and Go."))
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if strings.Contains(got, "\r") {
		t.Error("carriage returns were not normalised")
	}
	if strings.Contains(got, "\n\n\n") {
		t.Errorf("blank-line runs were not collapsed: %q", got)
	}
}

// A file's content decides the parser, not its extension — a PDF saved as .txt
// and a .docx named .doc are both things people actually do.
func TestExtractSniffsContentOverExtension(t *testing.T) {
	// A zip named .doc is really a .docx.
	if _, err := Extract("cv.doc", buildDOCX(t, "Flutter")); err != nil {
		t.Errorf("a .docx named .doc should be parsed by content: %v", err)
	}

	// Whereas a genuine legacy .doc is refused with an actionable message.
	legacy := append([]byte{0xD0, 0xCF, 0x11, 0xE0}, make([]byte, 100)...)
	_, err := Extract("cv.doc", legacy)
	if !errors.Is(err, ErrUnsupported) {
		t.Errorf("legacy .doc should be ErrUnsupported, got %v", err)
	}
	if err != nil && !strings.Contains(err.Error(), ".docx") {
		t.Errorf("the error should say what to do instead, got %q", err)
	}
}

func TestExtractRejectsUnusableInput(t *testing.T) {
	cases := map[string]struct {
		name string
		data []byte
	}{
		"empty file":   {"cv.pdf", nil},
		"unknown type": {"cv.xlsx", []byte("some bytes here")},
		"zip but not docx": {"cv.docx", func() []byte {
			var buf bytes.Buffer
			zw := zip.NewWriter(&buf)
			f, _ := zw.Create("content.xml") // an .odt, say
			_, _ = f.Write([]byte("<x/>"))
			_ = zw.Close()
			return buf.Bytes()
		}()},
		"binary junk": {"cv.txt", []byte{0xff, 0xfe, 0x00, 0x01, 0x02}},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := Extract(tc.name, tc.data); err == nil {
				t.Error("expected an error")
			}
		})
	}
}

func TestExtractRejectsOversizedFile(t *testing.T) {
	_, err := Extract("cv.txt", make([]byte, MaxFileBytes+1))
	if err == nil || !strings.Contains(err.Error(), "limit") {
		t.Errorf("oversized file should be refused with a size message, got %v", err)
	}
}

// A whitespace-only document is not a usable CV, and saying so beats saving
// nothing and letting scoring fail later with a confusing message.
func TestExtractRejectsWhitespaceOnlyDocument(t *testing.T) {
	if _, err := Extract("cv.docx", buildDOCX(t, "   ", "\t")); !errors.Is(err, ErrNoText) {
		t.Errorf("want ErrNoText, got %v", err)
	}
}

func TestTidyStripsControlCharactersButKeepsLayout(t *testing.T) {
	got := tidy("Flutter\x00\x07 and\tDart\nGo")
	if strings.ContainsAny(got, "\x00\x07") {
		t.Errorf("control characters survived: %q", got)
	}
	if !strings.Contains(got, "\t") || !strings.Contains(got, "\n") {
		t.Errorf("tab/newline should be preserved: %q", got)
	}
}
