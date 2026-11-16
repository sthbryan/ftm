package cli

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/sthbryan/ftm/internal/i18n"
)

// Uninstall removes this ftm binary.
func Uninstall(out io.Writer) error {
	path, err := os.Executable()
	if err != nil {
		if path, err = exec.LookPath("ftm"); err != nil {
			return fmt.Errorf("%s", i18n.T("uninstall_not_found"))
		}
	}
	if abs, err := filepath.EvalSymlinks(path); err == nil {
		path = abs
	}

	fmt.Fprintln(out, i18n.TF("uninstall_removing", path))
	if err := removeSelf(path); err != nil {
		return fmt.Errorf("%s", i18n.TF("uninstall_error", err.Error()))
	}

	fmt.Fprintln(out, i18n.T("uninstall_success"))
	return nil
}
