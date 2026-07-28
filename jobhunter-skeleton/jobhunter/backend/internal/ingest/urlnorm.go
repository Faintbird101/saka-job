package ingest

import (
	"net/url"
	"regexp"
	"strings"
)

// linkedinIDRe pulls the trailing numeric job id out of a LinkedIn job URL,
// e.g. ".../jobs/view/flutter-product-engineer-at-workbuddy-4444856922" -> 4444856922.
// This is the same value the API also gives us as linkedin_id; we extract it
// from the URL as a cross-check / for sources where we only have the link.
var linkedinIDRe = regexp.MustCompile(`(\d{6,})/?$`)

// NormalizeURL produces a stable dedup key for the normalized_url column.
//   - lowercased
//   - scheme forced to https
//   - query string and fragment dropped
//   - leading "www." and 2-letter country subdomains (vn./mx./eg.) collapsed,
//     so vn.linkedin.com and www.linkedin.com become linkedin.com
//   - trailing slash trimmed
func NormalizeURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		// Not a parseable URL; fall back to a lowercased, trimmed string.
		return strings.ToLower(strings.TrimRight(raw, "/"))
	}

	host := strings.ToLower(u.Host)
	// Strip a leading www. or a 2-letter country subdomain (e.g. vn., mx., eg.).
	parts := strings.Split(host, ".")
	if len(parts) > 2 {
		first := parts[0]
		if first == "www" || len(first) == 2 {
			host = strings.Join(parts[1:], ".")
		}
	}

	path := strings.TrimRight(strings.ToLower(u.Path), "/")
	return "https://" + host + path
}

// ExtractLinkedInID returns the trailing numeric id from a LinkedIn-style URL,
// or "" if none is present.
func ExtractLinkedInID(raw string) string {
	m := linkedinIDRe.FindStringSubmatch(strings.TrimRight(raw, "/"))
	if len(m) == 2 {
		return m[1]
	}
	return ""
}
