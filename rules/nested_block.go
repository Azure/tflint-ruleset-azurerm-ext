package rules

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/terraform-provider-azurerm/provider"
	"math"
	"strings"
)

type NestedBlock struct {
	File                 *hcl.File
	Block                *hclsyntax.Block
	Name                 string
	SortField            string
	Range                hcl.Range
	HeadMetaArgument     *Args
	RequiredArgs         *Args
	OptionalArgs         *Args
	RequiredNestedBlocks *NestedBlocks
	OptionalNestedBlocks *NestedBlocks
	IsSorted             bool
	ParentBlockNames     []string
}

func (b *NestedBlock) Check(current hcl.Pos) (hcl.Pos, bool) {
	if b.Range.Start.Line < current.Line {
		return b.Range.Start, false
	}
	current = b.Range.Start
	sections := []Section{
		b.HeadMetaArgument,
		b.RequiredArgs,
		b.OptionalArgs,
		b.RequiredNestedBlocks,
		b.OptionalNestedBlocks,
	}
	var sorted bool
	for _, s := range sections {
		if current, sorted = s.Check(current); !sorted {
			return current, false
		}
	}
	return b.Range.End, true
}

type NestedBlocks struct {
	Blocks []*NestedBlock
	Range  *hcl.Range
	Type   ArgGrpType
}

func (b *NestedBlocks) Add(arg *NestedBlock) {
	b.Blocks = append(b.Blocks, arg)
	if b.Range == nil {
		b.Range = &hcl.Range{
			Filename: arg.Range.Filename,
			Start:    hcl.Pos{Line: math.MaxInt},
			End:      hcl.Pos{Line: -1},
		}
	}
	if b.Range.Start.Line > arg.Range.Start.Line {
		b.Range.Start = arg.Range.Start
	}
	if b.Range.End.Line < arg.Range.End.Line {
		b.Range.End = arg.Range.End
	}
}

func (b *NestedBlocks) Check(current hcl.Pos) (hcl.Pos, bool) {
	if b == nil {
		return current, true
	}
	if b.Range.Start.Line < current.Line {
		return b.Range.Start, false
	}
	var name *string
	for _, nb := range b.Blocks {
		if name == nil {
			name = &nb.Name
		}
		if *name > nb.Name {
			return nb.Range.Start, false
		}
		var sorted bool
		if current, sorted = nb.Check(current); !sorted {
			return current, false
		}
		name = &nb.Name
	}
	return b.Range.End, true
}

func (b *NestedBlock) buildArgGrpsWithAttrs(attributes hclsyntax.Attributes) {
	argSchemas := provider.GetArgSchema(b.ParentBlockNames)
	attrs := sortedAttributes(attributes)
	for _, attr := range attrs {
		attrName := attr.Name
		arg := buildAttrArg(attr)
		if IsHeadMeta(attrName) {
			b.addHeadMeta(arg)
			continue
		}
		attrSchema, isAzAttr := argSchemas[attrName]
		if isAzAttr && attrSchema.Required {
			b.addRequiredAttr(arg)
		} else {
			b.addOptionalAttr(arg)
		}
	}
}

func (b *NestedBlock) buildNestedBlocks(nestedBlock hclsyntax.Blocks) {
	for _, nb := range nestedBlock {
		b.buildNestedBlock(nb)
	}
}

func (b *NestedBlock) buildNestedBlock(nestedBlock *hclsyntax.Block) {
	var nestedBlockName, sortField string
	switch nestedBlock.Type {
	case "dynamic":
		nestedBlockName = nestedBlock.Labels[0]
		sortField = strings.Join(nestedBlock.Labels, "")
	default:
		nestedBlockName = nestedBlock.Type
		sortField = nestedBlock.Type
	}
	var parentBlockNames []string
	if nestedBlockName == "content" && b.Block.Type == "dynamic" {
		parentBlockNames = b.ParentBlockNames
	} else {
		parentBlockNames = append(b.ParentBlockNames, nestedBlockName)
	}
	nb := &NestedBlock{
		Name:             nestedBlockName,
		SortField:        sortField,
		Range:            nestedBlock.Range(),
		Block:            nestedBlock,
		ParentBlockNames: parentBlockNames,
	}
	nb.buildArgGrpsWithAttrs(nestedBlock.Body.Attributes)
	nb.buildNestedBlocks(nestedBlock.Body.Blocks)
	argSchemas := provider.GetArgSchema(b.ParentBlockNames)
	if _, required := argSchemas[nb.Name]; required {
		b.addRequiredNestedBlock(nb)
	} else {
		b.addOptionalNestedBlock(nb)
	}
}

func (b *NestedBlock) addHeadMeta(arg *Arg) {
	if b.HeadMetaArgument == nil {
		b.HeadMetaArgument = &Args{Type: HeadMetaArgs}
	}
	b.HeadMetaArgument.Add(arg)
}

func (b *NestedBlock) addRequiredAttr(arg *Arg) {
	if b.RequiredArgs == nil {
		b.RequiredArgs = &Args{Type: RequiredAzAttrs}
	}
	b.RequiredArgs.Add(arg)
}

func (b *NestedBlock) addOptionalAttr(arg *Arg) {
	if b.OptionalArgs == nil {
		b.OptionalArgs = &Args{Type: OptionalAttrs}
	}
	b.OptionalArgs.Add(arg)
}

func (b *NestedBlock) addRequiredNestedBlock(nb *NestedBlock) {
	if b.RequiredNestedBlocks == nil {
		b.RequiredNestedBlocks = &NestedBlocks{}
	}
	b.RequiredNestedBlocks.Add(nb)
}

func (b *NestedBlock) addOptionalNestedBlock(nb *NestedBlock) {
	if b.OptionalNestedBlocks == nil {
		b.OptionalNestedBlocks = &NestedBlocks{}
	}
	b.OptionalNestedBlocks.Add(nb)
}
