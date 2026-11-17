//go:build !ssa_only

package ssaapi

import "github.com/yaklang/yaklang/common/yakgrpc/ypb"

func (r *SyntaxFlowResult) GetGRPCModelResult() *ypb.SyntaxFlowResult {
	if r == nil {
		return nil
	}
	var ret *ypb.SyntaxFlowResult
	if r.dbResult != nil {
		ret = r.dbResult.ToGRPCModel()
	}
	if r.memResult != nil {
		ret = r.memResult.ToGRPCModel()
	}
	if ret != nil {
		ret.ResultID = uint64(r.id)
		ret.TaskID = r.TaskID
		ret.SaveKind = string(r.GetResultSaveKind())
	}
	return ret
}
