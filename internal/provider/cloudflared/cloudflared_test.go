package cloudflared

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseURL(t *testing.T) {
	p := &CloudflaredProvider{}

	tests := []struct {
		name string
		line string
		want string
	}{
		{
			name: "quick tunnel banner",
			line: "2026-08-03T20:11:02Z INF |  https://ntsc-project-router-giving.trycloudflare.com                    |",
			want: "https://ntsc-project-router-giving.trycloudflare.com",
		},
		{
			name: "failed request line is not a url announcement",
			line: `2026-08-03T20:12:10Z ERR Request failed error="dial tcp [::1]:30000: connect: connection refused" connIndex=0 dest=https://ntsc-project-router-giving.trycloudflare.com/favicon.ico ip=198.41.200.113 type=http`,
			want: "",
		},
		{
			name: "registration line without a tunnel host",
			line: "2026-08-03T20:11:01Z INF Requesting new quick Tunnel on trycloudflare.com...",
			want: "",
		},
		{
			name: "unrelated host",
			line: "2026-08-03T20:11:01Z INF see https://developers.cloudflare.com/argo-tunnel",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := p.ParseURL(tt.line); got != tt.want {
				t.Errorf("ParseURL() = %q, want %q", got, tt.want)
			}
		})
	}
}

// A cloudflared sitting in the working directory is never picked up: ftm only
// runs one from PATH, its own bin dir or a known install location.
func TestFindBinaryIgnoresWorkingDirectory(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("AppData", filepath.Join(home, "appdata"))
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("PATH", "")
	t.Setenv("ProgramFiles", filepath.Join(home, "programs"))

	work := t.TempDir()
	t.Chdir(work)
	p := New().(*CloudflaredProvider)
	if err := os.WriteFile(filepath.Join(work, p.BinaryName()), []byte("x"), 0o755); err != nil {
		t.Fatal(err)
	}

	got := p.FindBinary()
	if got == "" {
		return
	}
	abs, err := filepath.Abs(got)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Dir(abs) == work || filepath.Dir(got) == "." {
		t.Fatalf("FindBinary() = %q, picked up the copy in the working directory", got)
	}
}

func TestQuickTunnelIgnoresTheUsersOwnConfig(t *testing.T) {
	args := quickTunnelArgs("/tmp/ftm-cloudflared.yml", 51234)
	want := []string{"tunnel", "--config", "/tmp/ftm-cloudflared.yml", "--url", "http://127.0.0.1:51234"}
	if strings.Join(args, " ") != strings.Join(want, " ") {
		t.Fatalf("args = %v, want %v", args, want)
	}

	path, err := emptyConfig()
	if err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil || info.Size() != 0 {
		t.Fatalf("emptyConfig %q: %v, size %d", path, err, info.Size())
	}
}
