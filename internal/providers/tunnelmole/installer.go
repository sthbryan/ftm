package tunnelmole

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"foundry-tunnel/internal/providers"
)

const nodeVersion = "v20.11.1"

type NodeInstaller struct {
	BaseDir string
}

func NewNodeInstaller(baseDir string) *NodeInstaller {
	return &NodeInstaller{BaseDir: baseDir}
}

func (ni *NodeInstaller) NodeDir() string {
	return filepath.Join(ni.BaseDir, "node")
}

func (ni *NodeInstaller) NodeBin() string {
	nodeDir := ni.NodeDir()
	if runtime.GOOS == "windows" {
		return filepath.Join(nodeDir, "node.exe")
	}
	return filepath.Join(nodeDir, "bin", "node")
}

func (ni *NodeInstaller) NpmBin() string {
	nodeDir := ni.NodeDir()
	if runtime.GOOS == "windows" {
		return filepath.Join(nodeDir, "npm.cmd")
	}
	return filepath.Join(nodeDir, "bin", "npm")
}

func (ni *NodeInstaller) TunnelmoleBin() string {
	nodeDir := ni.NodeDir()
	if runtime.GOOS == "windows" {
		return filepath.Join(nodeDir, "tunnelmole.cmd")
	}
	return filepath.Join(nodeDir, "bin", "tunnelmole")
}

func (ni *NodeInstaller) IsInstalled() bool {
	_, err := os.Stat(ni.TunnelmoleBin())
	return err == nil
}

func (ni *NodeInstaller) nodeURL() string {
	var osName, arch, ext string
	
	switch runtime.GOOS {
	case "darwin":
		osName = "darwin"
		arch = "x64"
		ext = "tar.gz"
		if runtime.GOARCH == "arm64" {
			arch = "arm64"
		}
	case "linux":
		osName = "linux"
		arch = "x64"
		ext = "tar.xz"
	case "windows":
		osName = "win"
		arch = "x64"
		ext = "zip"
	default:
		return ""
	}
	
	return fmt.Sprintf("https://nodejs.org/dist/%s/node-%s-%s-%s.%s", 
		nodeVersion, nodeVersion, osName, arch, ext)
}

func (ni *NodeInstaller) Install(progress chan<- providers.DownloadProgress) error {
	if err := os.MkdirAll(ni.BaseDir, 0755); err != nil {
		return fmt.Errorf("failed to create base dir: %w", err)
	}
	
	// Check if already installed
	if ni.IsInstalled() {
		return nil
	}
	
	// Download Node.js
	nodeURL := ni.nodeURL()
	if nodeURL == "" {
		return fmt.Errorf("unsupported platform: %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	
	archivePath := filepath.Join(ni.BaseDir, "nodejs-archive")
	if err := ni.download(nodeURL, archivePath, progress); err != nil {
		return fmt.Errorf("failed to download Node.js: %w", err)
	}
	defer os.Remove(archivePath)
	
	// Extract
	if err := ni.extract(archivePath, ni.NodeDir()); err != nil {
		return fmt.Errorf("failed to extract Node.js: %w", err)
	}
	
	// Install tunnelmole globally in our Node installation
	if err := ni.installTunnelmole(progress); err != nil {
		return fmt.Errorf("failed to install tunnelmole: %w", err)
	}
	
	return nil
}

func (ni *NodeInstaller) download(url, dest string, progress chan<- providers.DownloadProgress) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}
	
	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()
	
	total := resp.ContentLength
	downloaded := int64(0)
	buf := make([]byte, 32*1024)
	
	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			out.Write(buf[:n])
			downloaded += int64(n)
			if total > 0 && progress != nil {
				progress <- providers.DownloadProgress{
					Percent: float64(downloaded) / float64(total) * 50,
					Current: downloaded,
					Total:   total,
				}
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
	}
	
	return nil
}

func (ni *NodeInstaller) extract(archivePath, destDir string) error {
	if strings.HasSuffix(archivePath, ".zip") {
		return ni.extractZip(archivePath, destDir)
	}
	return ni.extractTarGz(archivePath, destDir)
}

func (ni *NodeInstaller) extractTarGz(archivePath, destDir string) error {
	file, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer file.Close()
	
	var tr *tar.Reader
	
	if strings.HasSuffix(archivePath, ".xz") {
		// Would need xz package, fallback to tar command
		cmd := exec.Command("tar", "-xf", archivePath, "-C", ni.BaseDir)
		return cmd.Run()
	}
	
	gz, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer gz.Close()
	
	tr = tar.NewReader(gz)
	
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		
		// Remove first directory component (node-vXX.XX.X-...)
		parts := strings.Split(header.Name, "/")
		if len(parts) > 1 {
			target := filepath.Join(destDir, strings.Join(parts[1:], "/"))
			
			switch header.Typeflag {
			case tar.TypeDir:
				os.MkdirAll(target, os.FileMode(header.Mode))
			case tar.TypeReg:
				os.MkdirAll(filepath.Dir(target), 0755)
				out, err := os.Create(target)
				if err != nil {
					return err
				}
				if _, err := io.Copy(out, tr); err != nil {
					out.Close()
					return err
				}
				out.Close()
				os.Chmod(target, os.FileMode(header.Mode))
			}
		}
	}
	
	return nil
}

func (ni *NodeInstaller) extractZip(archivePath, destDir string) error {
	r, err := zip.OpenReader(archivePath)
	if err != nil {
		return err
	}
	defer r.Close()
	
	for _, f := range r.File {
		// Remove first directory component
		parts := strings.Split(f.Name, "/")
		if len(parts) > 1 {
			target := filepath.Join(destDir, strings.Join(parts[1:], "/"))
			
			if f.FileInfo().IsDir() {
				os.MkdirAll(target, f.Mode())
				continue
			}
			
			os.MkdirAll(filepath.Dir(target), 0755)
			
			rc, err := f.Open()
			if err != nil {
				return err
			}
			
			out, err := os.Create(target)
			if err != nil {
				rc.Close()
				return err
			}
			
			_, err = io.Copy(out, rc)
			out.Close()
			rc.Close()
			
			if err != nil {
				return err
			}
		}
	}
	
	return nil
}

func (ni *NodeInstaller) installTunnelmole(progress chan<- providers.DownloadProgress) error {
	npm := ni.NpmBin()
	
	// Set up npm prefix to install in our node directory
	nodeDir := ni.NodeDir()
	env := os.Environ()
	env = append(env, fmt.Sprintf("NPM_CONFIG_PREFIX=%s", nodeDir))
	env = append(env, fmt.Sprintf("NPM_CONFIG_CACHE=%s", filepath.Join(ni.BaseDir, "npm-cache")))
	
	cmd := exec.Command(npm, "install", "-g", "tunnelmole")
	cmd.Env = env
	cmd.Dir = ni.BaseDir
	
	// Run npm install
	if err := cmd.Run(); err != nil {
		return err
	}
	
	if progress != nil {
		progress <- providers.DownloadProgress{
			Percent: 100,
			Done:    true,
		}
	}
	
	return nil
}
