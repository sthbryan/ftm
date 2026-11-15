package bore

import "testing"

func TestAssetName(t *testing.T) {
	tests := []struct {
		goos, goarch, want string
	}{
		{"darwin", "arm64", "bore-v0.6.0-aarch64-apple-darwin.tar.gz"},
		{"darwin", "amd64", "bore-v0.6.0-x86_64-apple-darwin.tar.gz"},
		{"linux", "amd64", "bore-v0.6.0-x86_64-unknown-linux-musl.tar.gz"},
		{"linux", "arm64", "bore-v0.6.0-aarch64-unknown-linux-musl.tar.gz"},
		{"windows", "amd64", "bore-v0.6.0-x86_64-pc-windows-msvc.zip"},
	}

	for _, tt := range tests {
		got, err := assetName("v0.6.0", tt.goos, tt.goarch)
		if err != nil || got != tt.want {
			t.Errorf("assetName(%s, %s) = %q, %v; want %q", tt.goos, tt.goarch, got, err, tt.want)
		}
	}

	if _, err := assetName("v0.6.0", "windows", "arm64"); err == nil {
		t.Error("assetName(windows/arm64) returned no error")
	}
}
