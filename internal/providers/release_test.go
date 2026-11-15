package providers

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestGitHubReleaseAsset(t *testing.T) {
	rel := &GitHubRelease{
		TagName: "2026.11.0",
		HTMLURL: "https://github.com/cloudflare/cloudflared/releases/tag/2026.11.0",
		Assets: []GitHubAsset{
			{Name: "cloudflared-linux-amd64", BrowserDownloadURL: "https://example.test/linux", Digest: "sha256:ABCDEF"},
			{Name: "cloudflared-darwin-arm64.tgz", BrowserDownloadURL: "https://example.test/darwin"},
		},
	}

	got, err := rel.Release("cloudflared-linux-amd64")
	if err != nil {
		t.Fatal(err)
	}
	if got.Version != "2026.11.0" || got.Asset.URL != "https://example.test/linux" || got.Asset.SHA256 != "abcdef" {
		t.Errorf("Release() = %+v", got)
	}

	if a, _ := rel.Asset("cloudflared-darwin-arm64.tgz"); a.SHA256 != "" {
		t.Errorf("asset without a digest got SHA256 %q", a.SHA256)
	}

	if _, err := rel.Asset("missing"); err == nil {
		t.Error("Asset(missing) returned no error")
	}
}

func TestDigestHex(t *testing.T) {
	if got := digestHex("sha256:AbC"); got != "abc" {
		t.Errorf("digestHex = %q", got)
	}
	if got := digestHex("sha512:abc"); got != "" {
		t.Errorf("digestHex accepted sha512: %q", got)
	}
	if got := digestHex(""); got != "" {
		t.Errorf("digestHex(\"\") = %q", got)
	}
}

func TestDownloadVerifiesChecksum(t *testing.T) {
	body := []byte("pretend this is cloudflared")
	sum := sha256.Sum256(body)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("ETag", `"v1"`)
		_, _ = w.Write(body)
	}))
	defer srv.Close()

	dir := t.TempDir()

	etag, err := Download(context.Background(), srv.URL, filepath.Join(dir, "ok"), hex.EncodeToString(sum[:]), nil, "test")
	if err != nil {
		t.Fatalf("Download with the right digest: %v", err)
	}
	if etag != `"v1"` {
		t.Errorf("etag = %q", etag)
	}

	_, err = Download(context.Background(), srv.URL, filepath.Join(dir, "bad"), "00", nil, "test")
	if !errors.Is(err, ErrChecksumMismatch) {
		t.Fatalf("Download with a wrong digest = %v, want ErrChecksumMismatch", err)
	}
}

func TestReplaceBinary(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "bin", "tool")
	if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dest, []byte("old"), 0755); err != nil {
		t.Fatal(err)
	}

	src := filepath.Join(dir, "new")
	if err := os.WriteFile(src, []byte("new"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := ReplaceBinary(src, dest); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(dest)
	if err != nil || string(data) != "new" {
		t.Fatalf("dest = %q, %v; want new", data, err)
	}
	if _, err := os.Stat(src); !os.IsNotExist(err) {
		t.Error("the staged file was left behind")
	}
	if _, err := os.Stat(dest + ".old"); !os.IsNotExist(err) {
		t.Error("the backup was left behind")
	}
}

func TestBundledPathFallsBackToLegacy(t *testing.T) {
	root := t.TempDir()
	current := filepath.Join(root, "new", "bin")
	legacy := filepath.Join(root, "legacy", "bin")

	if got, want := bundledPath(current, legacy, "bore"), filepath.Join(current, "bore"); got != want {
		t.Errorf("nothing installed: got %q, want %q", got, want)
	}

	if err := os.MkdirAll(legacy, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(legacy, "bore"), []byte("x"), 0755); err != nil {
		t.Fatal(err)
	}
	if got, want := bundledPath(current, legacy, "bore"), filepath.Join(legacy, "bore"); got != want {
		t.Errorf("legacy only: got %q, want %q", got, want)
	}

	if err := os.MkdirAll(current, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(current, "bore"), []byte("y"), 0755); err != nil {
		t.Fatal(err)
	}
	if got, want := bundledPath(current, legacy, "bore"), filepath.Join(current, "bore"); got != want {
		t.Errorf("both: got %q, want %q", got, want)
	}
}

func TestMigrateBinDir(t *testing.T) {
	root := t.TempDir()
	legacy := filepath.Join(root, "legacy")
	current := filepath.Join(root, "current")

	if err := os.MkdirAll(legacy, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(current, 0755); err != nil {
		t.Fatal(err)
	}
	for name, body := range map[string]string{"cloudflared": "old-cf", "bore": "old-bore"} {
		if err := os.WriteFile(filepath.Join(legacy, name), []byte(body), 0755); err != nil {
			t.Fatal(err)
		}
	}
	// A newer copy already in place must win.
	if err := os.WriteFile(filepath.Join(current, "bore"), []byte("new-bore"), 0755); err != nil {
		t.Fatal(err)
	}

	if err := migrateBinDir(legacy, current); err != nil {
		t.Fatal(err)
	}

	if data, _ := os.ReadFile(filepath.Join(current, "cloudflared")); string(data) != "old-cf" {
		t.Errorf("cloudflared was not moved: %q", data)
	}
	if _, err := os.Stat(filepath.Join(legacy, "cloudflared")); !os.IsNotExist(err) {
		t.Error("cloudflared is still in the legacy dir")
	}
	if data, _ := os.ReadFile(filepath.Join(current, "bore")); string(data) != "new-bore" {
		t.Errorf("bore was overwritten: %q", data)
	}

	if err := migrateBinDir(filepath.Join(root, "missing"), current); err != nil {
		t.Errorf("missing legacy dir: %v", err)
	}
}

func TestInstallRecordRoundTrip(t *testing.T) {
	isolateConfigDir(t)

	if _, ok := LookupInstall("tunnelmole"); ok {
		t.Fatal("found a record before any install")
	}

	bin := filepath.Join(t.TempDir(), "tmole")
	if err := os.WriteFile(bin, []byte("v1"), 0755); err != nil {
		t.Fatal(err)
	}

	rel := &Release{Version: "2.4.0", ETag: `"abc"`, Asset: Asset{URL: "https://example.test/tmole"}}
	if err := RecordRelease("tunnelmole", bin, rel, ""); err != nil {
		t.Fatal(err)
	}

	rec, ok := LookupInstall("tunnelmole")
	if !ok || rec.Version != "2.4.0" || rec.ETag != `"abc"` || rec.InstalledAt.IsZero() {
		t.Fatalf("LookupInstall = %+v, %v", rec, ok)
	}
	if !rec.Matches(bin) {
		t.Error("the record does not match the file it was written for")
	}

	// Someone swaps the binary behind ftm's back.
	if err := os.WriteFile(bin, []byte("replaced"), 0755); err != nil {
		t.Fatal(err)
	}
	if rec.Matches(bin) {
		t.Error("the record still matches a replaced binary")
	}
}

func isolateConfigDir(t *testing.T) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("APPDATA", filepath.Join(home, "AppData", "Roaming"))
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
}
