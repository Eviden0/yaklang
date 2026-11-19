//go:build !ssa_only

package ssaapi

import (
	"github.com/samber/lo"
	"github.com/yaklang/yaklang/common/schema"
	"github.com/yaklang/yaklang/common/yakgrpc/ypb"
)

func (r *SyntaxFlowResult) GetGRPCModelRisk() []*ypb.SSARisk {
	if r == nil {
		return nil
	}

	// load risk from database
	if r.dbResult != nil && len(r.riskMap) != len(r.dbResult.RiskHashs) {
		for name := range r.dbResult.RiskHashs {
			r.getRisk(name)
		}
	}
	// Binding an audit row changes metadata but not the number of risks.
	// Do not cache by map length: it would return stale ids and timestamps.
	return lo.MapToSlice(r.riskMap, func(name string, risk *schema.SSARisk) *ypb.SSARisk {
		return risk.ToGRPCModel()
	})
}
