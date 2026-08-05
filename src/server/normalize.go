package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/url"
	"path"
	"strings"
)

var trackingParameters = []string{
	"gh_jid",
	"ref",
	"source",
	"utm_campaign",
	"utm_content",
	"utm_medium",
	"utm_source",
	"utm_term",
}

func canonicalizeURL(raw string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return "", fmt.Errorf("parse application URL: %w", err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", fmt.Errorf("application URL must use http or https")
	}
	if parsed.Hostname() == "" {
		return "", fmt.Errorf("application URL must include a host")
	}

	parsed.Fragment = ""
	parsed.Host = strings.ToLower(parsed.Host)
	parsed.Path = path.Clean(parsed.Path)
	if parsed.Path == "." {
		parsed.Path = "/"
	}
	parsed.Path = strings.TrimSuffix(parsed.Path, "/")
	if parsed.Path == "" {
		parsed.Path = "/"
	}

	query := parsed.Query()
	for _, parameter := range trackingParameters {
		query.Del(parameter)
	}
	parsed.RawQuery = query.Encode()
	return parsed.String(), nil
}

func derivedCanonicalKey(canonicalURL string) string {
	digest := sha256.Sum256([]byte(canonicalURL))
	return "url:" + hex.EncodeToString(digest[:12])
}

func normalizedCompanyKey(company string) string {
	return strings.ToLower(strings.Join(strings.Fields(company), "-"))
}
