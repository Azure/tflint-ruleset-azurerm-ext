package rules

import (
	"fmt"
	"github.com/hashicorp/go-multierror"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/hashicorp/terraform-provider-azurerm/provider"
	"github.com/terraform-linters/tflint-plugin-sdk/tflint"
	"sort"
	"strings"
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

// Block is the wrapper of hclsyntax.Block, it contains more info
type Block struct {
	File             *hcl.File
	Block            *hclsyntax.Block
	ArgGrps          map[ArgGrpType]*ArgGrp
	IsSorted         bool
	parentBlockNames []string
}

// ArgGrpType is an enumeration used for differentiating arguments
type ArgGrpType string

const (
	// the enumeration for argument group types

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

// BuildRootBlock Build the root block wrapper using hclsyntax.Block
func BuildRootBlock(block *hclsyntax.Block, file *hcl.File) *Block {
	return buildBlock(block, file, []string{block.Type, block.Labels[0]})
}

// CheckArgOrder recursively Checks whether the args in a block and its nested block are correctly sorted
func (b *Block) CheckArgOrder(runner tflint.Runner, r tflint.Rule) error {
	if !b.isSorted() {
		return runner.EmitIssue(r, fmt.Sprintf("Arguments are expected to be sorted in following order:\n%s", b.printSorted()), b.Block.DefRange())
	}
	var err error
	for _, block := range b.getNestedBlocks() {
		subErr := block.CheckArgOrder(runner, r)
		if subErr != nil {
			err = multierror.Append(err, subErr)
		}
	}
	return err
}

func buildBlock(block *hclsyntax.Block, file *hcl.File, parentBlockNames []string) *Block {
	b := initBlock(block, file, parentBlockNames)
	b.buildArgGrpsWithAttrs(block.Body.Attributes)
	b.buildArgGrpsWithNestedBlocks(block.Body.Blocks)
	return b
}

func initBlock(block *hclsyntax.Block, file *hcl.File, parentBlockNames []string) *Block {
	b := &Block{ArgGrps: make(map[ArgGrpType]*ArgGrp)}
	b.File = file
	b.Block = block
	for _, argGrpType := range argGrpTypes {
		b.ArgGrps[argGrpType] = &ArgGrp{IsSorted: true}
	}
	b.parentBlockNames = append(b.parentBlockNames, parentBlockNames...)
	return b
}

func buildAttrArg(attr *hclsyntax.Attribute) *Arg {
	return &Arg{
		Name:      attr.Name,
		SortField: attr.Name,
		Range:     attr.SrcRange,
	}
}

func (b *Block) buildArgGrpsWithAttrs(attributes hclsyntax.Attributes) {
	argSchemas := provider.GetArgSchema(b.parentBlockNames)
	for attrName, attr := range attributes {
		arg := buildAttrArg(attr)
		if IsHeadMeta(attrName) {
			b.addArg(HeadMetaArgs, arg)
			continue
		}
		if IsTailMeta(attrName) {
			b.addArg(TailMetaArgs, arg)
			continue
		}
		attrSchema, isAzAttr := argSchemas[attrName]
		if isAzAttr && attrSchema.Required {
			b.addArg(RequiredAzAttrs, arg)
			continue
		}
		if isAzAttr {
			b.addArg(OptionalAzAttrs, arg)
		} else {
			b.addArg(NonAzAttrs, arg)
		}
	}
}

func (b *Block) buildNestedBlockArg(nestedBlock *hclsyntax.Block) *Arg {
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
		parentBlockNames = b.parentBlockNames
	} else {
		parentBlockNames = append(b.parentBlockNames, nestedBlockName)
	}
	return &Arg{
		Name:      nestedBlockName,
		SortField: sortField,
		Range:     nestedBlock.Range(),
		Block:     buildBlock(nestedBlock, b.File, parentBlockNames),
	}
}

func (b *Block) buildArgGrpsWithNestedBlocks(nestedBlocks hclsyntax.Blocks) {
	argSchemas := provider.GetArgSchema(b.parentBlockNames)
	for _, nestedBlock := range nestedBlocks {
		arg := b.buildNestedBlockArg(nestedBlock)
		if IsHeadMeta(arg.Name) {
			b.addArg(HeadMetaArgs, arg)
			continue
		}
		if IsTailMeta(arg.Name) {
			b.addArg(TailMetaArgs, arg)
			continue
		}
		blockSchema, isAzNestedBlock := argSchemas[arg.Name]
		if isAzNestedBlock && blockSchema.Required {
			b.addArg(RequiredAzNestedBlocks, arg)
			continue
		}
		if isAzNestedBlock {
			b.addArg(OptionalAzNestedBlocks, arg)
		} else {
			b.addArg(NonAzNestedBlocks, arg)
		}
	}
}

func (b *Block) addArg(argGrpType ArgGrpType, arg *Arg) {
	b.validateArgOrder(argGrpType, arg)
	b.appendArg(argGrpType, arg)
}

func (b *Block) getNestedBlocks() []*Block {
	var args []*Arg
	for _, argGrpType := range argGrpTypes {
		args = append(args, b.ArgGrps[argGrpType].Args...)
	}
	var nestedBlocks []*Block
	for _, arg := range args {
		if arg.Block != nil {
			nestedBlocks = append(nestedBlocks, arg.Block)
		}
	}
	return nestedBlocks
}

func (b *Block) isSorted() bool {
	if !b.isArgGrpsSorted() {
		return false
	}
	b.mergeGeneralArgs()
	return b.isCorrectlySplit()
}

func (b *Block) printSorted() string {
	b.sortArgGrps()
	b.mergeGeneralArgs()
	return b.print()
}

func (b *Block) print() string {
	isGapNeeded := false
	var sortedArgTxts []string
	sortedArgGrpTypes := []ArgGrpType{HeadMetaArgs, Attrs, NestedBlocks, TailMetaArgs}
	for _, argGrpType := range sortedArgGrpTypes {
		args := b.ArgGrps[argGrpType].Args
		if len(args) == 0 {
			continue
		}
		if isGapNeeded {
			sortedArgTxts = append(sortedArgTxts, "")
		}
		for _, arg := range args {
			sortedArgTxts = append(sortedArgTxts, b.printArg(arg))
		}
		isGapNeeded = true
	}
	sortedBlockHclTxt := strings.Join(sortedArgTxts, "\n")
	blockHead := string(b.Block.DefRange().SliceBytes(b.File.Bytes))
	if strings.TrimSpace(sortedBlockHclTxt) == "" {
		sortedBlockHclTxt = fmt.Sprintf("%s {}", blockHead)
	} else {
		sortedBlockHclTxt = fmt.Sprintf("%s {\n%s\n}", blockHead, sortedBlockHclTxt)
	}
	return string(hclwrite.Format([]byte(sortedBlockHclTxt)))
}

func (b *Block) printArg(arg *Arg) string {
	if arg.Block != nil {
		return arg.Block.printSorted()
	}
	return string(arg.Range.SliceBytes(b.File.Bytes))
}

func (b *Block) sortArgGrps() {
	for _, argGrpType := range argGrpTypes {
		b.sortArgs(argGrpType)
	}
}

func (b *Block) sortArgs(argGrpType ArgGrpType) {
	args := b.ArgGrps[argGrpType].Args
	switch argGrpType {
	case HeadMetaArgs:
		sort.Slice(args, func(i, j int) bool {
			return GetHeadMetaPriority(args[i].Name) > GetHeadMetaPriority(args[j].Name)
		})
	case TailMetaArgs:
		sort.Slice(args, func(i, j int) bool {
			return GetTailMetaPriority(args[i].Name) > GetTailMetaPriority(args[j].Name)
		})
	default:
		sort.Slice(args, func(i, j int) bool {
			return args[i].SortField < args[j].SortField
		})
	}
}

func (b *Block) validateArgOrder(argGrpType ArgGrpType, arg *Arg) {
	argGrp := b.ArgGrps[argGrpType]
	validateFunc := func(existedArg *Arg) bool {
		switch argGrpType {
		case HeadMetaArgs:
			return (GetHeadMetaPriority(arg.Name) > GetHeadMetaPriority(existedArg.Name) && ComparePos(arg.Range.Start, existedArg.Range.Start) > 0) ||
				(GetHeadMetaPriority(arg.Name) < GetHeadMetaPriority(existedArg.Name) && ComparePos(arg.Range.Start, existedArg.Range.Start) < 0)
		case TailMetaArgs:
			return (GetTailMetaPriority(arg.Name) > GetTailMetaPriority(existedArg.Name) && ComparePos(arg.Range.Start, existedArg.Range.Start) > 0) ||
				(GetTailMetaPriority(arg.Name) < GetTailMetaPriority(existedArg.Name) && ComparePos(arg.Range.Start, existedArg.Range.Start) < 0)
		default:
			return (arg.SortField < existedArg.SortField && ComparePos(arg.Range.Start, existedArg.Range.Start) > 0) ||
				(arg.SortField > existedArg.SortField && ComparePos(arg.Range.Start, existedArg.Range.Start) < 0)
		}
	}
	for _, existedArg := range argGrp.Args {
		if !argGrp.IsSorted {
			break
		}
		if validateFunc(existedArg) {
			argGrp.IsSorted = false
		}
	}
}

func (b *Block) appendArg(argGrpType ArgGrpType, arg *Arg) {
	argGrp := b.ArgGrps[argGrpType]
	argGrp.Args = append(argGrp.Args, arg)
	if ComparePos(argGrp.Start, arg.Range.Start) > 0 {
		argGrp.Start = arg.Range.Start
	}
	if ComparePos(argGrp.End, arg.Range.End) < 0 {
		argGrp.End = arg.Range.End
	}
}

func (b *Block) isArgGrpsSorted() bool {
	var lastGrp *ArgGrp
	for _, argGrpType := range argGrpTypes {
		if len(b.ArgGrps[argGrpType].Args) == 0 {
			continue
		}
		if !b.ArgGrps[argGrpType].IsSorted {
			return false
		}
		if lastGrp != nil && ComparePos(b.ArgGrps[argGrpType].Start, lastGrp.End) <= 0 {
			return false
		}
		lastGrp = b.ArgGrps[argGrpType]
	}
	return true
}

func (b *Block) isCorrectlySplit() bool {
	argGrpTypes := []ArgGrpType{HeadMetaArgs, Attrs, NestedBlocks, TailMetaArgs}
	var lastGrp *ArgGrp
	for _, argGrpType := range argGrpTypes {
		if len(b.ArgGrps[argGrpType].Args) == 0 {
			continue
		}
		if lastGrp != nil && b.ArgGrps[argGrpType].Start.Line-lastGrp.End.Line < 2 {
			return false
		}
		lastGrp = b.ArgGrps[argGrpType]
	}
	return true
}

func (b *Block) mergeGeneralArgs() {
	attrGrpTypes := []ArgGrpType{RequiredAzAttrs, OptionalAzAttrs, NonAzAttrs}
	b.mergeArgGrps(Attrs, attrGrpTypes)
	blockGrpTypes := []ArgGrpType{RequiredAzNestedBlocks, OptionalAzNestedBlocks, NonAzNestedBlocks}
	b.mergeArgGrps(NestedBlocks, blockGrpTypes)
}

func (b *Block) mergeArgGrps(targetGrpType ArgGrpType, srcGrpTypes []ArgGrpType) {
	b.ArgGrps[targetGrpType] = new(ArgGrp)
	for _, srcGrpType := range srcGrpTypes {
		b.ArgGrps[targetGrpType].Args = append(b.ArgGrps[targetGrpType].Args, b.ArgGrps[srcGrpType].Args...)
		if ComparePos(b.ArgGrps[targetGrpType].Start, b.ArgGrps[srcGrpType].Start) > 0 {
			b.ArgGrps[targetGrpType].Start = b.ArgGrps[srcGrpType].Start
		}
		if ComparePos(b.ArgGrps[targetGrpType].End, b.ArgGrps[srcGrpType].End) < 0 {
			b.ArgGrps[targetGrpType].End = b.ArgGrps[srcGrpType].End
		}
		b.ArgGrps[targetGrpType].IsSorted = b.ArgGrps[targetGrpType].IsSorted && b.ArgGrps[srcGrpType].IsSorted
	}
}
