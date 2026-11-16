package supervisor

import (
	"strings"
	"testing"

	"github.com/sthbryan/ftm/internal/core"
)

func TestCrashReportsWhatTheProviderPrinted(t *testing.T) {
	script := `echo "Error: could not connect to bore.pub:7835"
echo "Caused by:"
echo "    timed out"
exit 1`

	m, updates := newScriptManager(t, script)
	t.Cleanup(m.StopAll)

	if err := m.Start(tunnelFixture()); err != nil {
		t.Fatalf("Start: %v", err)
	}

	status := waitForState(t, updates, core.TunnelStateError)

	if !strings.Contains(status.ErrorMessage, "could not connect to bore.pub:7835") {
		t.Errorf("ErrorMessage = %q, want the reason the provider printed", status.ErrorMessage)
	}
	if !strings.Contains(status.ErrorMessage, "timed out") {
		t.Errorf("ErrorMessage = %q, want the cause that followed it", status.ErrorMessage)
	}
}

func TestFailureReason(t *testing.T) {
	tests := []struct {
		name  string
		lines []string
		want  string
	}{
		{
			name: "bore reports the cause on the lines after the error",
			lines: []string{
				"2026-08-03T20:11:02Z  INFO bore_cli::client: connecting to server",
				"Error: could not connect to bore.pub:7835",
				"",
				"Caused by:",
				"    timed out",
			},
			want: "Error: could not connect to bore.pub:7835 Caused by: timed out",
		},
		{
			name: "the last error line wins over earlier noise",
			lines: []string{
				"Error: retrying",
				"INFO reconnected",
				"INFO listening",
				"Error: connection refused",
			},
			want: "Error: connection refused",
		},
		{
			name: "without a marker the last line is used",
			lines: []string{
				"INFO starting",
				"INFO goodbye",
			},
			want: "INFO goodbye",
		},
		{
			name:  "empty logs fall back",
			lines: []string{"", "   "},
			want:  "exit status 1",
		},
		{
			name:  "ansi colours are stripped",
			lines: []string{"\x1b[31mError: connection refused\x1b[0m"},
			want:  "Error: connection refused",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := failureReason(tt.lines, "exit status 1"); got != tt.want {
				t.Errorf("failureReason() = %q, want %q", got, tt.want)
			}
		})
	}
}
