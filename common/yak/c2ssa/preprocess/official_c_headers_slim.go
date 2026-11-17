//go:build ssa_only

package preprocess

import (
	"context"

	"github.com/yaklang/yaklang/common/utils"
)

const (
	OfficialCHeadersName    = "c-std-headers"
	OfficialCHeadersZipName = "c-std-headers.zip"
)

func OfficialCHeadersAutoDownloadDisabled() bool {
	return true
}

func OfficialCHeadersPackPath() (string, error) {
	return "", utils.Errorf("official C headers are not bundled in the slim engine")
}

func HasOfficialCHeadersLibrary() bool {
	return false
}

func EnsureOfficialCHeaders(ctx context.Context, force bool) (packPath string, version string, err error) {
	return "", "", utils.Errorf("official C headers are not bundled in the slim engine")
}

func EnsureExternalIncludeDirs() []string {
	return nil
}

func LastOfficialCHeadersAutoDownloadError() error {
	return nil
}

func FetchOfficialCHeadersVersion(ctx context.Context) string {
	return ""
}
