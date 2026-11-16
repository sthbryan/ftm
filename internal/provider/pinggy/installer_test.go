package pinggy

import "testing"

func TestAssetName(t *testing.T) {
	tests := []struct {
		goos, goarch, want string
	}{
		{"darwin", "arm64", "pinggy-macos-arm64"},
		{"darwin", "amd64", "pinggy-macos-x64"},
		{"linux", "amd64", "pinggy-linux-x64"},
		{"linux", "arm64", "pinggy-linux-arm64"},
		{"windows", "amd64", "pinggy-win-x64.exe"},
		{"windows", "arm64", "pinggy-win-arm64.exe"},
	}

	for _, tt := range tests {
		got, err := assetName(tt.goos, tt.goarch)
		if err != nil || got != tt.want {
			t.Errorf("assetName(%s, %s) = %q, %v; want %q", tt.goos, tt.goarch, got, err, tt.want)
		}
	}
}
