package rules

import (
	"github.com/terraform-linters/tflint-plugin-sdk/tflint"
)

var headMetaArgPriority = map[string]int{"for_each": 0, "count": 0, "provider": 1}
var tailMetaArgPriority = map[string]int{"lifecycle": 0, "depends_on": 1}

// IsHeadMeta checks whether a name represents a type of head Meta arg
func IsHeadMeta(argName string) bool {
	_, isHeadMeta := headMetaArgPriority[argName]
	return isHeadMeta
}

// IsTailMeta checks whether a name represents a type of tail Meta arg
func IsTailMeta(argName string) bool {
	_, isTailMeta := tailMetaArgPriority[argName]
	return isTailMeta
}

func getExistedRules() map[string]tflint.Rule {
	rules := make(map[string]tflint.Rule)
	for _, rule := range Rules {
		rules[rule.Name()] = rule
	}
	return rules
}
