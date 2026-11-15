package providers

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// InstallRecord is what ftm remembers about a binary it downloaded. The
// version is a fallback for binaries whose --version cannot run (tmole needs
// Rosetta on Apple Silicon); the ETag tracks tunnelmole's unversioned file.
type InstallRecord struct {
	Version     string    `json:"version,omitempty"`
	ETag        string    `json:"etag,omitempty"`
	SHA256      string    `json:"sha256,omitempty"`
	URL         string    `json:"url,omitempty"`
	InstalledAt time.Time `json:"installedAt"`
	// Size and ModTime identify the file ftm wrote, so the record is not
	// trusted for a binary that was replaced since.
	Size    int64     `json:"size,omitempty"`
	ModTime time.Time `json:"modTime,omitempty"`
}

// Matches reports whether the record still describes the file at path.
func (r InstallRecord) Matches(path string) bool {
	info, err := os.Stat(path)
	if err != nil || r.Size == 0 {
		return false
	}
	return info.Size() == r.Size && info.ModTime().Equal(r.ModTime)
}

const manifestName = "installed.json"

var manifestMu sync.Mutex

func manifestPath() string {
	return filepath.Join(BinDir(), manifestName)
}

func readManifest() map[string]InstallRecord {
	records := map[string]InstallRecord{}
	data, err := os.ReadFile(manifestPath())
	if err != nil {
		return records
	}
	_ = json.Unmarshal(data, &records)
	return records
}

// LookupInstall returns what was recorded when name was last installed.
func LookupInstall(name string) (InstallRecord, bool) {
	manifestMu.Lock()
	defer manifestMu.Unlock()
	rec, ok := readManifest()[name]
	return rec, ok
}

// RecordInstall stores rec for name, replacing the previous record.
func RecordInstall(name string, rec InstallRecord) error {
	manifestMu.Lock()
	defer manifestMu.Unlock()

	records := readManifest()
	if rec.InstalledAt.IsZero() {
		rec.InstalledAt = time.Now().UTC()
	}
	records[name] = rec

	data, err := json.MarshalIndent(records, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(BinDir(), 0755); err != nil {
		return err
	}

	tmp := manifestPath() + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return err
	}
	return os.Rename(tmp, manifestPath())
}

// RecordRelease stores the release that was just installed for name at path.
func RecordRelease(name, path string, rel *Release, etag string) error {
	if etag == "" {
		etag = rel.ETag
	}
	rec := InstallRecord{
		Version: rel.Version,
		ETag:    etag,
		SHA256:  rel.Asset.SHA256,
		URL:     rel.Asset.URL,
	}
	if info, err := os.Stat(path); err == nil {
		rec.Size = info.Size()
		rec.ModTime = info.ModTime()
	}
	return RecordInstall(name, rec)
}
