package rules

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
)

// Arg includes attr and nested block defined in a block
type Arg struct {
	Name      string
	SortField string
	Range     hcl.Range
	Block     *Block
}

// ArgGrp is the collection of args with the same type
type ArgGrp struct {
	Args     []*Arg
	Start    hcl.Pos
	End      hcl.Pos
	IsSorted bool
}

// ArgGrpType is an enumeration used for differentiating arguments
type ArgGrpType string

// the enumeration for argument group types
const (
	HeadMetaArgs           ArgGrpType = "headMetaArgs"
	RequiredAzAttrs                   = "requiredAzAttrs"
	OptionalAzAttrs                   = "optionalAzAttrs"
	NonAzAttrs                        = "nonAzAttrs"
	RequiredAzNestedBlocks            = "requiredAzNestedBlocks"
	OptionalAzNestedBlocks            = "optionalAzNestedBlocks"
	NonAzNestedBlocks                 = "nonAzNestedBlocks"
	TailMetaArgs                      = "tailMetaArgs"
	Attrs                             = "attrs"
	NestedBlocks                      = "nestedBlocks"
)

var argGrpTypes = []ArgGrpType{
	HeadMetaArgs,
	RequiredAzAttrs,
	OptionalAzAttrs,
	NonAzAttrs,
	RequiredAzNestedBlocks,
	OptionalAzNestedBlocks,
	NonAzNestedBlocks,
	TailMetaArgs,
}

func buildAttrArg(attr *hclsyntax.Attribute) *Arg {
	return &Arg{
		Name:      attr.Name,
		SortField: attr.Name,
		Range:     attr.SrcRange,
	}
}
