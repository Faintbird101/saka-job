package ingest

import "testing"

func TestNormalizeURL(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			// The case that motivated the whole normalized_url column: the
			// same posting served from a country subdomain.
			name: "country subdomain collapses to the apex",
			in:   "https://vn.linkedin.com/jobs/view/flutter-engineer-at-acme-4444856922",
			want: "https://linkedin.com/jobs/view/flutter-engineer-at-acme-4444856922",
		},
		{
			name: "www collapses to the same key",
			in:   "https://www.linkedin.com/jobs/view/flutter-engineer-at-acme-4444856922",
			want: "https://linkedin.com/jobs/view/flutter-engineer-at-acme-4444856922",
		},
		{
			name: "tracking query string is dropped",
			in:   "https://www.linkedin.com/jobs/view/x-123?trk=public_jobs&refId=abc",
			want: "https://linkedin.com/jobs/view/x-123",
		},
		{
			name: "fragment is dropped",
			in:   "https://linkedin.com/jobs/view/x-123#apply",
			want: "https://linkedin.com/jobs/view/x-123",
		},
		{
			name: "http is forced to https so scheme differences don't split the key",
			in:   "http://linkedin.com/jobs/view/x-123",
			want: "https://linkedin.com/jobs/view/x-123",
		},
		{
			name: "trailing slash and casing are normalised",
			in:   "https://LinkedIn.com/Jobs/View/X-123/",
			want: "https://linkedin.com/jobs/view/x-123",
		},
		{
			name: "non-linkedin host is left intact apart from www",
			in:   "https://www.greenhouse.io/acme/jobs/998877",
			want: "https://greenhouse.io/acme/jobs/998877",
		},
		{
			name: "deep subdomain that is not two letters is preserved",
			in:   "https://careers.acme.com/job/42",
			want: "https://careers.acme.com/job/42",
		},
		{
			name: "empty input yields empty key, never a collidable value",
			in:   "   ",
			want: "",
		},
		{
			name: "unparseable input degrades to a lowercased string",
			in:   "not a url",
			want: "not a url",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := NormalizeURL(tc.in); got != tc.want {
				t.Errorf("NormalizeURL(%q)\n got: %q\nwant: %q", tc.in, got, tc.want)
			}
		})
	}
}

// TestNormalizeURLCollapsesVariants is the property the dedup guard actually
// depends on: every subdomain variant of one posting must produce one key.
func TestNormalizeURLCollapsesVariants(t *testing.T) {
	variants := []string{
		"https://vn.linkedin.com/jobs/view/x-4444856922",
		"https://mx.linkedin.com/jobs/view/x-4444856922",
		"https://eg.linkedin.com/jobs/view/x-4444856922",
		"https://www.linkedin.com/jobs/view/x-4444856922?trk=guest",
		"http://linkedin.com/jobs/view/x-4444856922/",
	}

	first := NormalizeURL(variants[0])
	for _, v := range variants[1:] {
		if got := NormalizeURL(v); got != first {
			t.Errorf("variant %q normalised to %q, expected %q — dedup would let a duplicate through", v, got, first)
		}
	}
}

func TestExtractLinkedInID(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"https://vn.linkedin.com/jobs/view/flutter-engineer-at-acme-4444856922", "4444856922"},
		{"https://www.linkedin.com/jobs/view/x-4444856922/", "4444856922"},
		{"https://greenhouse.io/acme/jobs/998877", "998877"},
		{"https://example.com/jobs/apply", ""},
		// Too short to be a job id; must not be mistaken for one, or unrelated
		// postings would collide on the UNIQUE linkedin_id column.
		{"https://example.com/jobs/12", ""},
		{"", ""},
	}

	for _, tc := range tests {
		if got := ExtractLinkedInID(tc.in); got != tc.want {
			t.Errorf("ExtractLinkedInID(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
