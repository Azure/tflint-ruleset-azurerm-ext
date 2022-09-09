package rules

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"math"
)

type Section interface {
	Check(current hcl.Pos) (hcl.Pos, bool)
}

// Arg includes attr and nested block defined in a block
type Arg struct {
	Name      string
	SortField string
	Range     hcl.Range
}

// Args is the collection of args with the same type
type Args struct {
	Args  []*Arg
	Range *hcl.Range
	Type  ArgGrpType
}

func (a *Args) Add(arg *Arg) {
	a.Args = append(a.Args, arg)
	if a.Range == nil {
		a.Range = &hcl.Range{
			Filename: arg.Range.Filename,
			Start:    hcl.Pos{Line: math.MaxInt},
			End:      hcl.Pos{Line: -1},
		}
	}
	if a.Range.Start.Line > arg.Range.Start.Line {
		a.Range.Start = arg.Range.Start
	}
	if a.Range.End.Line < arg.Range.End.Line {
		a.Range.End = arg.Range.End
	}
}

func (a *Args) Check(current hcl.Pos) (hcl.Pos, bool) {
	if a == nil {
		return current, true
	}
	if a.Range.Start.Line < current.Line {
		return a.Range.Start, false
	}

	if a.Type == HeadMetaArgs {
		score := -1
		for _, arg := range a.Args {
			if headMetaArgPriority[arg.Name] < score {
				return arg.Range.Start, false
			}
			score = headMetaArgPriority[arg.Name]
		}
		return a.Range.End, true
	}

	var name *string
	for _, arg := range a.Args {
		if name == nil {
			name = &arg.Name
		}
		if *name > arg.Name {
			return arg.Range.Start, false
		}
		name = &arg.Name
	}
	return a.Range.End, true
}

// ArgGrpType is an enumeration used for differentiating arguments
type ArgGrpType string

// the enumeration for argument group types
const (
	HeadMetaArgs         ArgGrpType = "headMetaArgs"
	RequiredAzAttrs                 = "requiredAzAttrs"
	OptionalAttrs                   = "optionalAttrs"
	RequiredNestedBlocks            = "requiredNestedBlocks"
	OptionalNestedBlocks            = "optionalNestedBlocks"
	TailMetaArgs                    = "tailMetaArgs"
	TailMetaNestedBlocks            = "tailNestedArgs"
)

func buildAttrArg(attr *hclsyntax.Attribute) *Arg {
	return &Arg{
		Name:      attr.Name,
		SortField: attr.Name,
		Range:     attr.SrcRange,
	}
}
