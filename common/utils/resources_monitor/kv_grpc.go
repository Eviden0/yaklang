//go:build !ssa_only

package resources_monitor

import "github.com/yaklang/yaklang/common/yakgrpc/yakit"

func getResourceHash(resourceID string) string {
	return yakit.Get(resourceID)
}

func setResourceHash(resourceID, hash string) {
	yakit.Set(resourceID, hash)
}
