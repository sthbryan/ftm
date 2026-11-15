package process

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/sthbryan/ftm/internal/providers"
)

func isolateHome(t *testing.T) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("APPDATA", filepath.Join(home, "AppData", "Roaming"))
}

func TestCompareReleaseByVersion(t *testing.T) {
	rel := &providers.Release{Version: "2026.11.0"}

	tests := []struct {
		name       string
		info       ProviderBinary
		wantLatest string
		wantUpdate bool
	}{
		{"older bundled", ProviderBinary{Installed: true, Source: SourceBundled, Version: "2024.6.1"}, "2026.11.0", true},
		{"current bundled", ProviderBinary{Installed: true, Source: SourceBundled, Version: "2026.11.0"}, "2026.11.0", false},
		{"older on PATH still reports", ProviderBinary{Installed: true, Source: SourcePath, Version: "2025.1.0"}, "2026.11.0", true},
		{"unreadable bundled copy is replaced", ProviderBinary{Installed: true, Source: SourceBundled}, "2026.11.0", true},
		{"unreadable copy on PATH is left alone", ProviderBinary{Installed: true, Source: SourcePath}, "2026.11.0", false},
		{"not installed", ProviderBinary{Source: SourceNone}, "2026.11.0", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			latest, update := compareRelease("cloudflared", tt.info, rel)
			if latest != tt.wantLatest || update != tt.wantUpdate {
				t.Errorf("compareRelease = (%q, %v), want (%q, %v)", latest, update, tt.wantLatest, tt.wantUpdate)
			}
		})
	}
}

func TestCompareReleaseByETag(t *testing.T) {
	isolateHome(t)

	installed := ProviderBinary{Installed: true, Source: SourceBundled, Version: "2.3.9"}
	rel := &providers.Release{Version: "2.4.0", ETag: `"one"`}

	// Without a recorded ETag the npm version decides, once.
	if _, update := compareRelease("tunnelmole", installed, rel); !update {
		t.Error("no recorded ETag and an older version: want an update")
	}

	if err := providers.RecordInstall("tunnelmole", providers.InstallRecord{ETag: `"one"`}); err != nil {
		t.Fatal(err)
	}

	// The download has not changed, so the copy on disk is the latest even
	// though npm reports a higher number.
	latest, update := compareRelease("tunnelmole", installed, rel)
	if update || latest != "2.3.9" {
		t.Errorf("same ETag: got (%q, %v), want (2.3.9, false)", latest, update)
	}

	rel.ETag = `"two"`
	if _, update := compareRelease("tunnelmole", installed, rel); !update {
		t.Error("changed ETag: want an update")
	}
}

func TestLatestEntryFreshness(t *testing.T) {
	now := time.Now()

	if (latestEntry{}).fresh(now) {
		t.Error("an empty entry is fresh")
	}
	if !(latestEntry{checkedAt: now.Add(-time.Hour)}).fresh(now) {
		t.Error("a one-hour-old answer is stale")
	}
	if (latestEntry{checkedAt: now.Add(-7 * time.Hour)}).fresh(now) {
		t.Error("a seven-hour-old answer is fresh")
	}
	if (latestEntry{checkedAt: now.Add(-20 * time.Minute), err: providers.ErrChecksumMismatch}).fresh(now) {
		t.Error("a failed lookup is not retried after 15 minutes")
	}
}

func TestUpdateProviderRefusesSystemProviders(t *testing.T) {
	m := NewManager()
	if _, err := m.UpdateProvider(t.Context(), "serveo", nil); err != ErrNotUpdatable {
		t.Errorf("UpdateProvider(serveo) = %v, want ErrNotUpdatable", err)
	}
	if _, err := m.UpdateProvider(t.Context(), "nope", nil); err != ErrUnknownProvider {
		t.Errorf("UpdateProvider(nope) = %v, want ErrUnknownProvider", err)
	}
}
