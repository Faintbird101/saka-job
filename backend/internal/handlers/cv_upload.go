package handlers

import (
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/yourname/jobhunter/backend/internal/cvparse"
)

// UploadCV handles POST /profile/cv — a multipart upload of a PDF, DOCX, or
// text CV.
//
// It extracts the text and returns it, but does NOT save. That is deliberate:
// PDF extraction can mangle a heavily designed CV into unusable soup, and
// silently overwriting your profile with that would break scoring in a way
// that is hard to notice. The editor shows you the text and you press Save.
func (h *Handler) UploadCV(w http.ResponseWriter, r *http.Request) {
	// Bound the request before parsing, so an oversized upload is refused
	// rather than buffered.
	r.Body = http.MaxBytesReader(w, r.Body, cvparse.MaxFileBytes+1<<20)

	if err := r.ParseMultipartForm(cvparse.MaxFileBytes); err != nil {
		h.badRequest(w, r, "could not read the upload: "+err.Error())
		return
	}
	defer func() {
		if r.MultipartForm != nil {
			_ = r.MultipartForm.RemoveAll()
		}
	}()

	file, header, err := r.FormFile("file")
	if err != nil {
		h.badRequest(w, r, `no file in the request (expected a multipart field named "file")`)
		return
	}
	defer file.Close()

	data, err := io.ReadAll(io.LimitReader(file, cvparse.MaxFileBytes+1))
	if err != nil {
		h.badRequest(w, r, "could not read the file: "+err.Error())
		return
	}

	text, err := cvparse.Extract(header.Filename, data)
	if err != nil {
		// These are all user-fixable — wrong format, a scanned PDF — so the
		// message goes back verbatim rather than being flattened to a generic
		// 400. writeError would hide exactly the detail that helps.
		switch {
		case errors.Is(err, cvparse.ErrUnsupported), errors.Is(err, cvparse.ErrNoText):
			h.badRequest(w, r, err.Error())
		default:
			h.badRequest(w, r, fmt.Sprintf("could not extract text from %s: %v", header.Filename, err))
		}
		return
	}

	h.log.Info("cv extracted",
		"filename", header.Filename, "bytes", len(data), "chars", len(text))

	h.writeJSON(w, r, http.StatusOK, map[string]any{
		"filename": header.Filename,
		"chars":    len(text),
		"text":     text,
	})
}
