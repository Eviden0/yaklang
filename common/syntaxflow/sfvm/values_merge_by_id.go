package sfvm

import (
	"github.com/yaklang/yaklang/common/utils"
)

// buildValueByKey indexes values by valueDedupKey (program-scoped instruction
// id). Unlike a bare int64 id map this keeps base-layer and diff-layer values
// apart in an incremental overlay query. The boolean reports whether at least
// one value carried a usable id.
func buildValueByKey(values Values) (map[dedupKey]ValueOperator, bool) {
	res := make(map[dedupKey]ValueOperator, len(values))
	hasID := false
	for _, value := range values {
		if utils.IsNil(value) {
			continue
		}
		key, ok := valueDedupKey(value)
		if !ok {
			continue
		}
		if key.kind == 1 {
			hasID = true
		}
		if _, exists := res[key]; !exists {
			res[key] = value
		}
	}
	return res, hasID
}

func intersectValuesByString(left Values, right Values) Values {
	rightByString := make(map[string]struct{}, len(right))
	for _, rv := range right {
		if utils.IsNil(rv) {
			continue
		}
		rightByString[rv.String()] = struct{}{}
	}
	var out []ValueOperator
	for _, lv := range left {
		if utils.IsNil(lv) {
			continue
		}
		if _, ok := rightByString[lv.String()]; ok {
			out = append(out, lv)
		}
	}
	return NewValues(out)
}

// mergeValuesByID implements SyntaxFlow's logical and/or operators. Values are
// matched by (program name, instruction id): instruction ids are per program,
// so an overlay query must not treat a base value and a diff value with the
// same id as one finding.
func mergeValuesByID(left Values, right Values, andMode bool) Values {
	if andMode {
		if left.IsEmpty() && right.IsEmpty() {
			return NewEmptyValues()
		}
		if left.IsEmpty() {
			return right
		}
		if right.IsEmpty() {
			return left
		}
	}

	leftByKey, leftHasID := buildValueByKey(left)
	rightByKey, rightHasID := buildValueByKey(right)

	// Fallback for values without ids: keep existing side in OR mode.
	if !leftHasID || !rightHasID {
		if andMode {
			return intersectValuesByString(left, right)
		}
		return MergeValues(left, right)
	}

	// Preserve the historical output order: left values first, then the right
	// values not already present on the left. In AND mode only values whose
	// (program, id) key exists on both sides survive.
	out := make(Values, 0, len(left)+len(right))
	seen := make(map[dedupKey]struct{}, len(left)+len(right))
	addLeft := func() {
		for _, lv := range left {
			if utils.IsNil(lv) {
				continue
			}
			key, ok := valueDedupKey(lv)
			if !ok {
				continue
			}
			if _, dup := seen[key]; dup {
				continue
			}
			if andMode {
				if _, ok := rightByKey[key]; !ok {
					continue
				}
			}
			seen[key] = struct{}{}
			value := lv
			if leftValue, exists := leftByKey[key]; exists {
				MergeAnchor(leftValue, value)
			}
			if rightValue, exists := rightByKey[key]; exists {
				MergeAnchor(rightValue, value)
			}
			out = append(out, value)
		}
	}
	addLeft()
	if !andMode {
		for _, rv := range right {
			if utils.IsNil(rv) {
				continue
			}
			key, ok := valueDedupKey(rv)
			if !ok {
				continue
			}
			if _, dup := seen[key]; dup {
				continue
			}
			seen[key] = struct{}{}
			value := rv
			if leftValue, exists := leftByKey[key]; exists {
				MergeAnchor(leftValue, value)
			}
			if rightValue, exists := rightByKey[key]; exists {
				MergeAnchor(rightValue, value)
			}
			out = append(out, value)
		}
	}

	if len(out) == 0 && andMode {
		// Program-like compare candidates may not share stable ids/strings.
		// Keep a non-empty side instead of dropping everything.
		if len(right) > 0 {
			return right
		}
		if len(left) > 0 {
			return left
		}
	}
	return out
}
