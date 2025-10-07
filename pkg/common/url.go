package common

import "strings"

// EnsureURLProtocol ensures the URL has a protocol scheme (defaults to https if missing)
func EnsureURLProtocol(url string) string {
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		return "https://" + url
	}
	return url
}