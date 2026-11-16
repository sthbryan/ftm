package cli

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"
)

// Every command runs the same against a running ftm and against the core it
// starts itself.
func TestTunnelCommandsInBothModes(t *testing.T) {
	for _, m := range modes {
		t.Run(m.name, func(t *testing.T) {
			h := newHarness(t, m)

			if out := h.mustRun("ls"); !strings.Contains(out, "No connections yet") {
				t.Fatalf("ls on an empty config = %q", out)
			}

			out := h.mustRun("add", "Friday game", "--provider", "pinggy", "--port", "30001")
			if !strings.Contains(out, "Saved Friday game") {
				t.Fatalf("add = %q", out)
			}
			h.mustRun("add", "Sunday game")

			var list []Tunnel
			if err := json.Unmarshal([]byte(h.mustRun("ls", "--json")), &list); err != nil {
				t.Fatalf("ls --json is not JSON: %v\n%s", err, h.out)
			}
			if len(list) != 2 || list[0].Name != "Friday game" || list[0].Provider != "pinggy" || list[0].Port != 30001 {
				t.Fatalf("ls --json = %+v", list)
			}
			if list[1].Provider != "cloudflared" || list[1].Port != 30000 {
				t.Fatalf("add defaults = %+v, want cloudflared on 30000", list[1])
			}

			if out := h.mustRun("edit", "friday GAME", "--port", "30002"); !strings.Contains(out, "port 30002") {
				t.Fatalf("edit = %q", out)
			}

			out = h.mustRun("up", "Friday game")
			if !strings.Contains(out, "Friday game is open: https://") {
				t.Fatalf("up = %q (stderr %q)", out, h.err)
			}
			if !m.attached && !strings.Contains(h.err.String(), "Press Ctrl+C") {
				t.Fatalf("a standalone up keeps the tunnel open until Ctrl+C; stderr = %q", h.err)
			}

			if m.attached {
				if out := h.mustRun("logs", "Friday game"); !strings.Contains(out, "tunnel ready") {
					t.Fatalf("logs = %q", out)
				}
				if out := h.mustRun("ls"); !strings.Contains(out, "ONLINE") {
					t.Fatalf("ls after up = %q", out)
				}
				if out := h.mustRun("down"); !strings.Contains(out, "Closed Friday game") {
					t.Fatalf("down = %q", out)
				}
			} else {
				// The tunnel closed with the command that opened it.
				if out := h.mustRun("down"); !strings.Contains(out, "No connection is open") {
					t.Fatalf("down = %q", out)
				}
			}

			if code := h.run("rm", "Sunday game"); code == 0 || !strings.Contains(h.err.String(), "--yes") {
				t.Fatalf("rm without a terminal must ask for --yes; code %d, stderr %q", code, h.err)
			}
			if out := h.mustRun("rm", "Sunday game", "--yes"); !strings.Contains(out, "Removed Sunday game") {
				t.Fatalf("rm = %q", out)
			}
			if code := h.run("logs", "Sunday game"); code == 0 || !strings.Contains(h.err.String(), "No connection called Sunday game") {
				t.Fatalf("logs on a removed tunnel: code %d, stderr %q", code, h.err)
			}
		})
	}
}

func TestShareOpensAndCopiesTheInvite(t *testing.T) {
	for _, m := range modes {
		t.Run(m.name, func(t *testing.T) {
			h := newHarness(t, m)
			out := h.mustRun("share", "--port", "30005")
			if !strings.Contains(out, "Join Foundry on port 30005 on Foundry VTT: https://") {
				t.Fatalf("share = %q", out)
			}
			if !strings.HasPrefix(h.copied, "https://") || !strings.Contains(out, "Link copied") {
				t.Fatalf("copied %q, out %q", h.copied, out)
			}
			// A second share reuses the connection saved on that port.
			h.mustRun("share", "--port", "30005", "--no-copy")
			var list []Tunnel
			_ = json.Unmarshal([]byte(h.mustRun("ls", "--json")), &list)
			if len(list) != 1 {
				t.Fatalf("share saved %d connections, want 1", len(list))
			}
		})
	}
}

func TestUpReportsAFailure(t *testing.T) {
	h := newHarness(t, modes[1])
	h.core.sup.fail["Broken"] = "provider refused"
	h.mustRun("add", "Broken")
	if code := h.run("up", "Broken"); code != 1 || !strings.Contains(h.err.String(), "Broken didn't open: provider refused") {
		t.Fatalf("code %d, stderr %q", code, h.err)
	}
}

func TestUpNeedsANameWithoutATerminal(t *testing.T) {
	h := newHarness(t, modes[1])
	h.mustRun("add", "a")
	h.mustRun("add", "b")
	if code := h.run("up"); code == 0 || !strings.Contains(h.err.String(), "Name the connection") {
		t.Fatalf("code %d, stderr %q", code, h.err)
	}
}

func TestAddRejectsBadInput(t *testing.T) {
	h := newHarness(t, modes[0])
	if code := h.run("add"); code == 0 || !strings.Contains(h.err.String(), "Give the connection a name") {
		t.Fatalf("add without a name: code %d, stderr %q", code, h.err)
	}
	if code := h.run("add", "x", "--provider", "ngrok"); code == 0 || !strings.Contains(h.err.String(), "No provider called ngrok") {
		t.Fatalf("unknown provider: code %d, stderr %q", code, h.err)
	}
	if code := h.run("add", "x", "--port", "70000"); code == 0 || !strings.Contains(h.err.String(), "1 to 65535") {
		t.Fatalf("bad port: code %d, stderr %q", code, h.err)
	}
}

func TestConfigGetAndSet(t *testing.T) {
	for _, m := range modes {
		t.Run(m.name, func(t *testing.T) {
			h := newHarness(t, m)
			if out := h.mustRun("config", "set", "notification_sound", "off"); !strings.Contains(out, "notification_sound = off") {
				t.Fatalf("set = %q", out)
			}
			if out := h.mustRun("config", "get", "notification_sound"); strings.TrimSpace(out) != "off" {
				t.Fatalf("get = %q", out)
			}
			h.mustRun("config", "set", "language", "es")
			if out := h.mustRun("config"); !strings.Contains(out, "language = es") {
				t.Fatalf("config = %q", out)
			}
			if code := h.run("config", "set", "language", "klingon"); code == 0 {
				t.Fatal("an unknown language was accepted")
			}
			if code := h.run("config", "set", "autostart", "on"); code == 0 {
				t.Fatal("autostart turned on in a build without it")
			}
			if out := h.mustRun("config", "path"); strings.TrimSpace(out) != h.deps.Dirs.ConfigFile {
				t.Fatalf("path = %q", out)
			}
		})
	}
}

func TestConfigEditRefusesWhileFtmRuns(t *testing.T) {
	h := newHarness(t, modes[1])
	if code := h.run("config", "edit"); code == 0 || !strings.Contains(h.err.String(), "Close it first") {
		t.Fatalf("code %d, stderr %q", code, h.err)
	}
}

func TestServe(t *testing.T) {
	h := newHarness(t, modes[1])
	if out := h.mustRun("serve", "--open"); !strings.Contains(out, "already open") || h.opened == "" {
		t.Fatalf("serve against a running ftm: out %q, opened %q", out, h.opened)
	}

	h = newHarness(t, modes[0])
	if out := h.mustRun("serve", "--open"); !strings.Contains(out, "Web view: http://localhost:0") || h.opened != "http://localhost:0" {
		t.Fatalf("standalone serve: out %q, opened %q", out, h.opened)
	}
	if h.core.shutdown != 1 {
		t.Fatalf("serve shut the core down %d times, want 1", h.core.shutdown)
	}
}

func TestServeRelaunchesAfterAnUpdate(t *testing.T) {
	h := newHarness(t, modes[0])
	h.core.onServe = nil
	close(h.core.restart)
	done := make(chan int, 1)
	go func() { done <- Execute(t.Context(), []string{"serve"}, h.deps) }()
	select {
	case code := <-done:
		if code != 0 {
			t.Fatalf("code %d", code)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("serve did not end when a restart was requested")
	}
}

func TestTUIPointsAtTheRunningFtm(t *testing.T) {
	h := newHarness(t, modes[1])
	if code := h.run(); code != 1 || !strings.Contains(h.out.String(), "Close the other ftm window") {
		t.Fatalf("code %d, out %q", code, h.out)
	}
	h = newHarness(t, modes[0])
	h.mustRun()
}

func TestProvidersWithoutBinaryManager(t *testing.T) {
	for _, m := range modes {
		t.Run(m.name, func(t *testing.T) {
			h := newHarness(t, m)
			if out := h.mustRun("providers"); !strings.Contains(out, "Provider") {
				t.Fatalf("providers = %q", out)
			}
			if code := h.run("providers", "update", "ngrok"); code == 0 || !strings.Contains(h.err.String(), "No provider called ngrok") {
				t.Fatalf("code %d, stderr %q", code, h.err)
			}
		})
	}
}

func TestVersionHelpManAndCompletion(t *testing.T) {
	h := newHarness(t, modes[0])
	if out := h.mustRun("version"); !strings.Contains(out, "Foundry Tunnel Manager v") {
		t.Fatalf("version = %q", out)
	}
	if out := h.mustRun("--help"); !strings.Contains(out, "share") || !strings.Contains(out, "doctor") {
		t.Fatalf("help = %q", out)
	}
	for _, shell := range []string{"bash", "zsh", "fish", "powershell"} {
		if out := h.mustRun("completion", shell); len(out) < 100 {
			t.Fatalf("completion %s = %q", shell, out)
		}
	}
}

func TestManPage(t *testing.T) {
	// fang's man command writes to the process's stdout.
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	stdout := os.Stdout
	os.Stdout = w
	h := newHarness(t, modes[0])
	code := h.run("man")
	os.Stdout = stdout
	_ = w.Close()
	buf := make([]byte, 1<<20)
	n, _ := r.Read(buf)
	if code != 0 || !strings.Contains(string(buf[:n]), ".TH") {
		t.Fatalf("man: code %d, out %q", code, buf[:n])
	}
}

func TestFindTunnel(t *testing.T) {
	tunnels := []Tunnel{{ID: "a1", Name: "Game"}, {ID: "b2", Name: "game"}, {ID: "c3", Name: "Other"}}
	if got, err := findTunnel(tunnels, "b2"); err != nil || got.ID != "b2" {
		t.Fatalf("by id: %+v %v", got, err)
	}
	if got, err := findTunnel(tunnels, "other"); err != nil || got.ID != "c3" {
		t.Fatalf("by name: %+v %v", got, err)
	}
	if _, err := findTunnel(tunnels, "GAME"); err == nil {
		t.Fatal("two tunnels share a name and one was picked")
	}
	if _, err := findTunnel(tunnels, "nope"); err == nil {
		t.Fatal("found a tunnel that isn't there")
	}
}

func TestEditorCommand(t *testing.T) {
	cases := []struct{ visual, editor, goos, want string }{
		{"code -w", "vim", "linux", "code -w"},
		{"", "nano", "linux", "nano"},
		{"", "", "windows", "notepad"},
		{"", "", "linux", "vi"},
	}
	for _, c := range cases {
		if got := editorCommand(c.visual, c.editor, c.goos); got != c.want {
			t.Errorf("editorCommand(%q, %q, %q) = %q, want %q", c.visual, c.editor, c.goos, got, c.want)
		}
	}
}

func TestSettingsPatch(t *testing.T) {
	p, err := settingsPatch("notifications", "on")
	if err != nil || p.NotificationsEnabled == nil || *p.NotificationsEnabled != "granted" {
		t.Fatalf("notifications on = %+v, %v", p, err)
	}
	if _, err := settingsPatch("notification_sound", "maybe"); err == nil {
		t.Fatal("maybe accepted as on/off")
	}
	if _, err := settingsPatch("theme", "dark"); err == nil {
		t.Fatal("unknown key accepted")
	}
}
