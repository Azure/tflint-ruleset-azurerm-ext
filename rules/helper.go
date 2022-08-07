package rules

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/terraform-linters/tflint-plugin-sdk/helper"
	"sort"
)

type Arg struct {
	Name      string
	SortField string
	Range     hcl.Range
	Block     *hclsyntax.Block
}

var headMetaArgPriority, tailMetaArgPriority = map[string]int{"for_each": 1, "count": 1, "provider": 0}, map[string]int{"lifecycle": 1, "depends_on": 0}

func IsHeadMeta(argName string) bool {
	_, isHeadMeta := headMetaArgPriority[argName]
	return isHeadMeta
}

func IsTailMeta(argName string) bool {
	_, isTailMeta := tailMetaArgPriority[argName]
	return isTailMeta
}

func GetHeadMetaPriority(argName string) int {
	return headMetaArgPriority[argName]
}

func GetTailMetaPriority(argName string) int {
	return tailMetaArgPriority[argName]
}

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

func IsIssueEmpty(issue *helper.Issue) bool {
	return *issue == helper.Issue{}
}