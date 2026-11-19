//go:build !ssa_only

package sfreport

import (
	"github.com/yaklang/gorm"
	"github.com/yaklang/yaklang/common/schema"
	"github.com/yaklang/yaklang/common/yakgrpc/yakit"
	"github.com/yaklang/yaklang/common/yakgrpc/ypb"
)

func filterSSARiskLocal(db *gorm.DB, filter *ypb.SSARisksFilter) *gorm.DB {
	return yakit.FilterSSARisk(db, filter)
}

func getSSARiskDisposalsWithInheritanceLocal(db *gorm.DB, riskID int64) ([]schema.SSARiskDisposals, error) {
	return yakit.GetSSARiskDisposalsWithInheritance(db, riskID)
}
