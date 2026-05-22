package format

// EvaluateRules checks info against each configured rule and returns per-rule results and overall compliance.
// When rules is nil, no rules are configured and the submission is always compliant.
func EvaluateRules(info FileInfo, rules *FormatRulesConfig) ([]RuleResult, bool) {
	if rules == nil {
		return []RuleResult{}, true
	}

	results := make([]RuleResult, 0, 4)
	compliant := true

	if rules.MinPages != nil {
		pass := info.PageCount >= *rules.MinPages
		if !pass {
			compliant = false
		}
		results = append(results, RuleResult{
			Rule:     "min_pages",
			Status:   statusStr(pass),
			Actual:   info.PageCount,
			Required: *rules.MinPages,
		})
	}

	if rules.MaxPages != nil {
		pass := info.PageCount <= *rules.MaxPages
		if !pass {
			compliant = false
		}
		results = append(results, RuleResult{
			Rule:     "max_pages",
			Status:   statusStr(pass),
			Actual:   info.PageCount,
			Required: *rules.MaxPages,
		})
	}

	if rules.MinWordCount != nil {
		pass := info.WordCount >= *rules.MinWordCount
		if !pass {
			compliant = false
		}
		results = append(results, RuleResult{
			Rule:     "min_word_count",
			Status:   statusStr(pass),
			Actual:   info.WordCount,
			Required: *rules.MinWordCount,
		})
	}

	if len(rules.RequiredTypes) > 0 {
		pass := contains(rules.RequiredTypes, info.FileType)
		if !pass {
			compliant = false
		}
		results = append(results, RuleResult{
			Rule:     "file_type",
			Status:   statusStr(pass),
			Actual:   info.FileType,
			Required: rules.RequiredTypes,
		})
	}

	return results, compliant
}

func statusStr(pass bool) string {
	if pass {
		return "pass"
	}
	return "fail"
}

func contains(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}
