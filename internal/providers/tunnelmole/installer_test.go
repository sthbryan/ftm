package tunnelmole

import (
	"archive/tar"
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"
)

func writeGzip(t *testing.T, path string, write func(gz *gzip.Writer)) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	gz := gzip.NewWriter(f)
	write(gz)
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestUnpackBareGzip(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "tmole-mac.gz")
	writeGzip(t, src, func(gz *gzip.Writer) {
		_, _ = gz.Write([]byte("\xcf\xfa\xed\xfe binary"))
	})

	dest := filepath.Join(dir, "tmole")
	if err := unpack(src, dest); err != nil {
		t.Fatal(err)
	}
	if data, _ := os.ReadFile(dest); string(data) != "\xcf\xfa\xed\xfe binary" {
		t.Errorf("unpacked %q", data)
	}
}

func TestUnpackTarball(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "tmole-linux.gz")
	writeGzip(t, src, func(gz *gzip.Writer) {
		tw := tar.NewWriter(gz)
		for name, body := range map[string]string{"README": "docs", "dist/tmole": "ELF binary"} {
			_ = tw.WriteHeader(&tar.Header{Name: name, Mode: 0755, Size: int64(len(body)), Typeflag: tar.TypeReg})
			_, _ = tw.Write([]byte(body))
		}
		_ = tw.Close()
	})

	dest := filepath.Join(dir, "tmole")
	if err := unpack(src, dest); err != nil {
		t.Fatal(err)
	}
	if data, _ := os.ReadFile(dest); string(data) != "ELF binary" {
		t.Errorf("unpacked %q", data)
	}
}
