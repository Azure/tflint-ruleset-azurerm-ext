package rules

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/terraform-linters/tflint-plugin-sdk/tflint"
	"sort"
)

var headMetaArgPriority, tailMetaArgPriority = map[string]int{"for_each": 1, "count": 1, "provider": 0}, map[string]int{"lifecycle": 1, "depends_on": 0}

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

// GetHeadMetaPriority gets the priority of a head Meta arg
func GetHeadMetaPriority(argName string) int {
	return headMetaArgPriority[argName]
}

// GetTailMetaPriority gets the priority of a tail Meta arg
func GetTailMetaPriority(argName string) int {
	return tailMetaArgPriority[argName]
}

// GetArgsWithOriginalOrder returns the args with original order
func GetArgsWithOriginalOrder(args []Arg) []Arg {
	argsWithOriginalOrder := make([]Arg, len(args), len(args))
	copy(argsWithOriginalOrder, args)
	sort.Slice(argsWithOriginalOrder, func(i, j int) bool {
		if argsWithOriginalOrder[i].Range.Start.Line == argsWithOriginalOrder[j].Range.Start.Line {
			return argsWithOriginalOrder[i].Range.Start.Column < argsWithOriginalOrder[j].Range.Start.Column
		}
		return argsWithOriginalOrder[i].Range.Start.Line < argsWithOriginalOrder[j].Range.Start.Line
	})
	return argsWithOriginalOrder
}

// ComparePos compares the value of hcl.Pos pos1 and pos2,
//negative result means pos1 is prior to pos2,
//zero result means the 2 positions are identical,
//positive result means pos2 is prior to pos1
func ComparePos(pos1 hcl.Pos, pos2 hcl.Pos) int {
	if pos1.Line == pos2.Line {
		return pos1.Column - pos2.Column
	}
	return pos1.Line - pos2.Line
}

func getExistedRules() map[string]tflint.Rule {
	rules := make(map[string]tflint.Rule)
	for _, rule := range Rules {
		if rule.Name() == "basic_ext_ignore_config" {
			continue
		}
		rules[rule.Name()] = rule
	}
	return rules
}
