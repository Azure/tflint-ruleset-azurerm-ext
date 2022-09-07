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
	ArgGrps          map[string]*ArgGrp
	IsSorted         bool
	parentBlockNames []string
}

const (
	HEAD_META_ARGS           = "headMetaArgs"
	REQUIRED_AZ_ATTRS        = "requiredAzAttrs"
	OPTIONAL_AZ_ATTRS        = "optionalAzAttrs"
	NONAZ_ATTRS              = "nonAzAttrs"
	REQUIRED_AZ_NESTEDBLOCKS = "requiredAzNestedBlocks"
	OPTIONAL_AZ_NESTEDBLOCKS = "optionalAzNestedBlocks"
	NONAZ_NESTEDBLOCKS       = "nonAzNestedBlocks"
	TAIL_META_ARGS           = "tailMetaArgs"
	ATTRS                    = "attrs"
	NESTEDBLOCKS             = "nestedBlocks"
)

var argGrpTypes = []string{
	HEAD_META_ARGS,
	REQUIRED_AZ_ATTRS,
	OPTIONAL_AZ_ATTRS,
	NONAZ_ATTRS,
	REQUIRED_AZ_NESTEDBLOCKS,
	OPTIONAL_AZ_NESTEDBLOCKS,
	NONAZ_NESTEDBLOCKS,
	TAIL_META_ARGS,
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
	b := &Block{ArgGrps: make(map[string]*ArgGrp)}
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
			b.addArg(HEAD_META_ARGS, arg)
			continue
		}
		if IsTailMeta(attrName) {
			b.addArg(TAIL_META_ARGS, arg)
			continue
		}
		attrSchema, isAzAttr := argSchemas[attrName]
		if isAzAttr && attrSchema.Required {
			b.addArg(REQUIRED_AZ_ATTRS, arg)
			continue
		}
		if isAzAttr {
			b.addArg(OPTIONAL_AZ_ATTRS, arg)
		} else {
			b.addArg(NONAZ_ATTRS, arg)
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
			b.addArg(HEAD_META_ARGS, arg)
			continue
		}
		if IsTailMeta(arg.Name) {
			b.addArg(TAIL_META_ARGS, arg)
			continue
		}
		blockSchema, isAzNestedBlock := argSchemas[arg.Name]
		if isAzNestedBlock && blockSchema.Required {
			b.addArg(REQUIRED_AZ_NESTEDBLOCKS, arg)
			continue
		}
		if isAzNestedBlock {
			b.addArg(OPTIONAL_AZ_NESTEDBLOCKS, arg)
		} else {
			b.addArg(NONAZ_NESTEDBLOCKS, arg)
		}
	}
}

func (b *Block) addArg(argGrpType string, arg *Arg) {
	b.validateArgOrder(argGrpType, arg)
	b.appendArg(argGrpType, arg)
}

func (b *Block) getNestedBlocks() []*Block {
	var args []*Arg
	for _, name := range argGrpTypes {
		args = append(args, b.ArgGrps[name].Args...)
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
	sortedArgGrpNames := []string{HEAD_META_ARGS, ATTRS, NESTEDBLOCKS, TAIL_META_ARGS}
	for _, name := range sortedArgGrpNames {
		args := b.ArgGrps[name].Args
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
	for _, name := range argGrpTypes {
		b.sortArgs(name)
	}
}

func (b *Block) sortArgs(argGrpName string) {
	args := b.ArgGrps[argGrpName].Args
	switch argGrpName {
	case HEAD_META_ARGS:
		sort.Slice(args, func(i, j int) bool {
			return GetHeadMetaPriority(args[i].Name) > GetHeadMetaPriority(args[j].Name)
		})
	case TAIL_META_ARGS:
		sort.Slice(args, func(i, j int) bool {
			return GetTailMetaPriority(args[i].Name) > GetTailMetaPriority(args[j].Name)
		})
	default:
		sort.Slice(args, func(i, j int) bool {
			return args[i].SortField < args[j].SortField
		})
	}
}

func (b *Block) validateArgOrder(argGrpType string, arg *Arg) {
	argGrp := b.ArgGrps[argGrpType]
	validateFunc := func(existedArg *Arg) bool {
		switch argGrpType {
		case HEAD_META_ARGS:
			return (GetHeadMetaPriority(arg.Name) > GetHeadMetaPriority(existedArg.Name) && ComparePos(arg.Range.Start, existedArg.Range.Start) > 0) ||
				(GetHeadMetaPriority(arg.Name) < GetHeadMetaPriority(existedArg.Name) && ComparePos(arg.Range.Start, existedArg.Range.Start) < 0)
		case TAIL_META_ARGS:
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

func (b *Block) appendArg(argGrpType string, arg *Arg) {
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
	for _, name := range argGrpTypes {
		if len(b.ArgGrps[name].Args) == 0 {
			continue
		}
		if !b.ArgGrps[name].IsSorted {
			return false
		}
		if lastGrp != nil && ComparePos(b.ArgGrps[name].Start, lastGrp.End) <= 0 {
			return false
		}
		lastGrp = b.ArgGrps[name]
	}
	return true
}

func (b *Block) isCorrectlySplit() bool {
	names := []string{HEAD_META_ARGS, ATTRS, NESTEDBLOCKS, TAIL_META_ARGS}
	var lastGrp *ArgGrp
	for _, name := range names {
		if len(b.ArgGrps[name].Args) == 0 {
			continue
		}
		if lastGrp != nil && b.ArgGrps[name].Start.Line-lastGrp.End.Line < 2 {
			return false
		}
		lastGrp = b.ArgGrps[name]
	}
	return true
}

func (b *Block) mergeGeneralArgs() {
	attrGrpNames := []string{REQUIRED_AZ_ATTRS, OPTIONAL_AZ_ATTRS, NONAZ_ATTRS}
	b.mergeArgGrps(ATTRS, attrGrpNames)
	blockGrpNames := []string{REQUIRED_AZ_NESTEDBLOCKS, OPTIONAL_AZ_NESTEDBLOCKS, NONAZ_NESTEDBLOCKS}
	b.mergeArgGrps(NESTEDBLOCKS, blockGrpNames)
}

func (b *Block) mergeArgGrps(targetGrpName string, srcGrpNames []string) {
	b.ArgGrps[targetGrpName] = new(ArgGrp)
	for _, name := range srcGrpNames {
		b.ArgGrps[targetGrpName].Args = append(b.ArgGrps[targetGrpName].Args, b.ArgGrps[name].Args...)
		if ComparePos(b.ArgGrps[targetGrpName].Start, b.ArgGrps[name].Start) > 0 {
			b.ArgGrps[targetGrpName].Start = b.ArgGrps[name].Start
		}
		if ComparePos(b.ArgGrps[targetGrpName].End, b.ArgGrps[name].End) < 0 {
			b.ArgGrps[targetGrpName].End = b.ArgGrps[name].End
		}
		b.ArgGrps[targetGrpName].IsSorted = b.ArgGrps[targetGrpName].IsSorted && b.ArgGrps[name].IsSorted
	}
}
