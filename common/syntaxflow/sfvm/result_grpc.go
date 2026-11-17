//go:build !ssa_only

package sfvm

import "github.com/yaklang/yaklang/common/yakgrpc/ypb"

func (s *SFFrameResult) ToGRPCModel() *ypb.SyntaxFlowResult {
	if s == nil {
		return nil
	}
	return &ypb.SyntaxFlowResult{
		ResultID:    0,
		TaskID:      "", // TODO: TaskID
		Kind:        "", // TODO: Kind
		ProgramName: "", // TODO: ProgramName

		RuleName:    s.rule.RuleName,
		Title:       s.rule.Title,
		TitleZh:     s.rule.TitleZh,
		Description: s.GetDescription(),
		Severity:    string(s.rule.Severity),
		Purpose:     string(s.rule.Purpose),
		Language:    string(s.rule.Language),
		RuleContent: s.rule.Content,
	}
}
