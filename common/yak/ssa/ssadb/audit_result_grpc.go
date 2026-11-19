//go:build !ssa_only

package ssadb

import "github.com/yaklang/yaklang/common/yakgrpc/ypb"

func (r *AuditResult) ToGRPCModel() *ypb.SyntaxFlowResult {
	res := &ypb.SyntaxFlowResult{
		ResultID:    uint64(r.ID),
		TaskID:      r.TaskID,
		RuleName:    r.RuleName,
		Title:       r.RuleTitle,
		TitleZh:     r.RuleTitleZh,
		Description: r.RuleDesc,
		Severity:    r.RuleSeverity,
		Purpose:     r.RulePurpose,
		ProgramName: r.ProgramName,
		Language:    r.Language,
		RiskCount:   r.RiskCount,
		RuleContent: r.RuleContent,
		Kind:        string(r.Kind),
	}
	return res
}
