package ssacmds

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/yaklang/yaklang/common/utils"
)

// publishReportFile writes a report through a temporary file so a failed
// rename never destroys the previous report. write receives a temporary path
// and must leave the complete document there.
//
// Renaming over an existing file fails on Windows and can fail across devices,
// so the flow is: write temp -> rename temp over target -> keep the old file as
// .bak when the platform refuses the replace -> retry -> copy the finished
// bytes in place as the last resort. At no point is the old report removed
// before the new one exists.
func publishReportFile(path string, write func(string) error) error {
	if strings.TrimSpace(path) == "" {
		return utils.Error("report path is empty")
	}
	if write == nil {
		return utils.Error("report writer is nil")
	}
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, filepath.Base(path)+".tmp-*")
	if err != nil {
		return utils.Wrapf(err, "create report temp file: %v", err)
	}
	tmpPath := tmp.Name()
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return utils.Wrapf(err, "close report temp file: %v", err)
	}
	defer os.Remove(tmpPath)

	if err := write(tmpPath); err != nil {
		return err
	}
	if err := os.Rename(tmpPath, path); err == nil {
		return nil
	}

	// The target may be locked or already exist. Preserve it as .bak when
	// possible, then retry. A failed backup must not abort the publish: the
	// finished document is already on disk.
	if utils.GetFirstExistedFile(path) != "" {
		backup := path + ".bak"
		if utils.GetFirstExistedFile(backup) != "" {
			_ = os.Remove(backup)
		}
		_ = os.Rename(path, backup)
		if err := os.Rename(tmpPath, path); err == nil {
			return nil
		}
	}

	data, readErr := os.ReadFile(tmpPath)
	if readErr != nil {
		return utils.Wrapf(readErr, "read report temp file: %v", readErr)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return utils.Wrapf(err, "publish report %s: %v", path, err)
	}
	return nil
}
