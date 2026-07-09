package catalog

import "strings"

// URLVariants returns the SSH and HTTPS forms of a git remote so a caller can
// try whichever the dev's auth supports — one baked binary then serves both
// SSH and HTTPS users. The input's own form is returned first (so the baked or
// typed protocol is tried before the alternate). For anything that is not a
// github-style HTTPS or scp-like SSH URL (file://, ssh://, local paths), it
// returns just the input, so non-github catalogs are never probed twice.
func URLVariants(url string) []string {
	// scp-like SSH: git@HOST:OWNER/REPO(.git)
	if strings.HasPrefix(url, "git@") {
		host, path, ok := strings.Cut(strings.TrimPrefix(url, "git@"), ":")
		if !ok || host == "" || path == "" {
			return []string{url}
		}
		return []string{url, "https://" + host + "/" + path}
	}
	// HTTPS: https://HOST/OWNER/REPO(.git)
	if strings.HasPrefix(url, "https://") {
		host, path, ok := strings.Cut(strings.TrimPrefix(url, "https://"), "/")
		if !ok || host == "" || path == "" {
			return []string{url}
		}
		return []string{url, "git@" + host + ":" + path}
	}
	return []string{url}
}
