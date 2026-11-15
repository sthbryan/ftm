package providers

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

// versionPattern matches the first dotted version in a --version banner:
//
//	cloudflared version 2025.8.1 (built 2025-08-13-1219 UTC)
//	bore-cli 0.6.0
//	Pinggy CLI version: 0.6.7
//	2.4.0                                 (tmole)
var versionPattern = regexp.MustCompile(`\bv?(\d+\.\d+(?:\.\d+){0,2})\b`)

// ParseVersion pulls the version number out of a binary's --version output.
// It returns "" when the output holds none.
func ParseVersion(output string) string {
	for _, line := range strings.Split(StripANSI(output), "\n") {
		if m := versionPattern.FindStringSubmatch(line); m != nil {
			return m[1]
		}
	}
	return ""
}

// NormalizeVersion trims the "v" prefix release tags carry.
func NormalizeVersion(v string) string {
	v = strings.TrimSpace(v)
	v = strings.TrimPrefix(v, "v")
	return strings.TrimPrefix(v, "V")
}

// CompareVersions compares two dotted versions numerically, segment by
// segment, so it orders both semver (0.6.0) and cloudflared's calendar
// versions (2025.8.1). A missing segment counts as 0 and a pre-release or build
// suffix is ignored. It returns -1, 0 or 1.
func CompareVersions(a, b string) int {
	pa, pb := versionSegments(a), versionSegments(b)
	n := len(pa)
	if len(pb) > n {
		n = len(pb)
	}

	for i := 0; i < n; i++ {
		var x, y int
		if i < len(pa) {
			x = pa[i]
		}
		if i < len(pb) {
			y = pb[i]
		}
		if x < y {
			return -1
		}
		if x > y {
			return 1
		}
	}
	return 0
}

func versionSegments(v string) []int {
	v = NormalizeVersion(v)
	v = strings.SplitN(v, "-", 2)[0]
	v = strings.SplitN(v, "+", 2)[0]
	if v == "" {
		return nil
	}

	parts := strings.Split(v, ".")
	out := make([]int, len(parts))
	for i, p := range parts {
		out[i], _ = strconv.Atoi(p)
	}
	return out
}

// IsNewer reports whether latest is a strictly newer version than installed.
// Unknown versions never count as an update.
func IsNewer(latest, installed string) bool {
	if latest == "" || installed == "" {
		return false
	}
	return CompareVersions(latest, installed) > 0
}

const versionTimeout = 15 * time.Second

type versionCacheEntry struct {
	size    int64
	modTime time.Time
	version string
}

var (
	versionCacheMu sync.Mutex
	versionCache   = map[string]versionCacheEntry{}
)

// DetectVersion runs `path --version` and parses the result. The answer is
// cached until the file changes, since some providers (tmole, pinggy) are
// packaged Node apps that take a second to start.
func DetectVersion(ctx context.Context, path string) (string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return "", err
	}

	versionCacheMu.Lock()
	cached, ok := versionCache[path]
	versionCacheMu.Unlock()
	if ok && cached.size == info.Size() && cached.modTime.Equal(info.ModTime()) {
		return cached.version, nil
	}

	ctx, cancel := context.WithTimeout(ctx, versionTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, path, "--version")
	hideWindow(cmd)
	out, runErr := cmd.CombinedOutput()

	version := ParseVersion(string(out))
	if version == "" {
		if runErr != nil {
			return "", fmt.Errorf("%s --version: %w", path, runErr)
		}
		return "", fmt.Errorf("%s --version printed no version", path)
	}

	versionCacheMu.Lock()
	versionCache[path] = versionCacheEntry{size: info.Size(), modTime: info.ModTime(), version: version}
	versionCacheMu.Unlock()

	return version, nil
}
