//go:build !ssa_only

package syntaxflow_scan

import (
	"github.com/yaklang/gorm"
	"github.com/yaklang/yaklang/common/schema"
	"github.com/yaklang/yaklang/common/yakgrpc/yakit"
	"github.com/yaklang/yaklang/common/yakgrpc/ypb"
)

type ssaRiskLevelCount struct {
	Count    int64  `json:"count"`
	Severity string `json:"severity"`
}

func getMaxScanBatch(db *gorm.DB, programs []string) (uint64, error) {
	return yakit.GetMaxScanBatch(db, programs)
}

func getSSARiskLevelCount(db *gorm.DB, filter *ypb.SSARisksFilter) ([]*ssaRiskLevelCount, error) {
	counts, err := yakit.GetSSARiskLevelCount(db, filter)
	if err != nil {
		return nil, err
	}
	out := make([]*ssaRiskLevelCount, 0, len(counts))
	for _, count := range counts {
		out = append(out, &ssaRiskLevelCount{Count: count.Count, Severity: count.Severity})
	}
	return out, nil
}

func queryLatestSSAProgramNameByProjectId(db *gorm.DB, projectID uint64) (string, error) {
	return yakit.QueryLatestSSAProgramNameByProjectId(db, projectID)
}

func filterSyntaxFlowRule(db *gorm.DB, filter *ypb.SyntaxFlowRuleFilter) *gorm.DB {
	return yakit.FilterSyntaxFlowRule(db, filter)
}

func applySyntaxFlowRuleModeFilter(db *gorm.DB, modes []string) *gorm.DB {
	return yakit.ApplySyntaxFlowRuleModeFilter(db, modes)
}

func applySyntaxFlowRuleInput(input *ypb.SyntaxFlowRuleInput, rule *schema.SyntaxFlowRule) (*schema.SyntaxFlowRule, error) {
	return yakit.ApplySyntaxFlowRuleInput(input, rule)
}
