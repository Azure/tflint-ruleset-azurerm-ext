package rules

import (
	"fmt"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/terraform-provider-azurerm/provider"
	"sort"
	"strings"
)

// ResourceBlock is the wrapper of a resource block
type ResourceBlock struct {
	File                 *hcl.File
	Block                *hclsyntax.Block
	HeadMetaArgs         *Args
	RequiredArgs         *Args
	OptionalArgs         *Args
	RequiredNestedBlocks *NestedBlocks
	OptionalNestedBlocks *NestedBlocks
	TailMetaArgs         *Args
	TailMetaNestedBlocks *NestedBlocks
	parentBlockNames     []string
}

// BuildResourceBlock Build the root block wrapper using hclsyntax.Block
func BuildResourceBlock(block *hclsyntax.Block, file *hcl.File) *ResourceBlock {
	b := &ResourceBlock{
		File:             file,
		Block:            block,
		parentBlockNames: []string{block.Type, block.Labels[0]},
	}
	b.buildArgs(block.Body.Attributes)
	b.buildArgGrpsWithNestedBlocks(block.Body.Blocks)
	return b
}

// CheckArgOrder recursively Checks whether the args in a block and its nested block are correctly sorted
func (b *ResourceBlock) CheckArgOrder() (hcl.Pos, bool) {
	sections := []Section{
		b.HeadMetaArgs,
		b.RequiredArgs,
		b.OptionalArgs,
		b.RequiredNestedBlocks,
		b.OptionalNestedBlocks,
		b.TailMetaArgs,
		b.TailMetaNestedBlocks,
	}
	var current hcl.Pos
	var sorted bool
	for _, s := range sections {
		if current, sorted = s.Check(current); !sorted {
			return current, false
		}
	}
	return *new(hcl.Pos), true
}

func (b *ResourceBlock) buildArgs(attributes hclsyntax.Attributes) {
	argSchemas := provider.GetArgSchema(b.parentBlockNames)
	attrs := sortedAttributes(attributes)
	for _, attr := range attrs {
		attrName := attr.Name
		arg := buildAttrArg(attr)
		if IsHeadMeta(attrName) {
			b.addHeadMetaArg(arg)
			continue
		}
		if IsTailMeta(attrName) {
			b.addTailMetaArg(arg)
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

func sortedAttributes(attributes hclsyntax.Attributes) []*hclsyntax.Attribute {
	var attrs []*hclsyntax.Attribute
	for _, attr := range attributes {
		attrs = append(attrs, attr)
	}
	sort.Slice(attrs, func(i, j int) bool {
		return attrs[i].Range().Start.Line < attrs[j].Range().Start.Line
	})
	return attrs
}

func (b *ResourceBlock) buildNestedBlock(nestedBlock *hclsyntax.Block) *NestedBlock {
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
	nb := &NestedBlock{
		Name:             nestedBlockName,
		SortField:        sortField,
		Range:            nestedBlock.Range(),
		Block:            nestedBlock,
		ParentBlockNames: parentBlockNames,
	}
	nb.buildArgGrpsWithAttrs(nestedBlock.Body.Attributes)
	nb.buildNestedBlocks(nestedBlock.Body.Blocks)
	return nb
}

func (b *ResourceBlock) buildArgGrpsWithNestedBlocks(nestedBlocks hclsyntax.Blocks) {
	argSchemas := provider.GetArgSchema(b.parentBlockNames)
	for _, nestedBlock := range nestedBlocks {
		nb := b.buildNestedBlock(nestedBlock)
		if IsTailMeta(nb.Name) {
			b.addTailMetaNestedBlock(nb)
			continue
		}
		blockSchema, isAzNestedBlock := argSchemas[nb.Name]
		if isAzNestedBlock && blockSchema.Required {
			b.addRequiredNestedBlock(nb)
		} else {
			b.addOptionalNestedBlock(nb)
		}
	}
}

//func (b *ResourceBlock) printSorted() string {
//	b.sortArgGrps()
//	//b.mergeGeneralArgs()
//	return b.print()
//}

//func (b *ResourceBlock) print() string {
//	isGapNeeded := false
//	var sortedArgTxts []string
//
//	for _, group := range b.getSections() {
//		if isGapNeeded {
//			sortedArgTxts = append(sortedArgTxts, "")
//		}
//		for _, arg := range group.Args {
//			sortedArgTxts = append(sortedArgTxts, b.printArg(arg))
//		}
//		isGapNeeded = true
//	}
//	sortedBlockHclTxt := strings.Join(sortedArgTxts, "\n")
//	blockHead := string(b.Block.DefRange().SliceBytes(b.File.Bytes))
//	if strings.TrimSpace(sortedBlockHclTxt) == "" {
//		sortedBlockHclTxt = fmt.Sprintf("%s {}", blockHead)
//	} else {
//		sortedBlockHclTxt = fmt.Sprintf("%s {\n%s\n}", blockHead, sortedBlockHclTxt)
//	}
//	return string(hclwrite.Format([]byte(sortedBlockHclTxt)))
//}

//func (b *ResourceBlock) printArg(arg *Arg) string {
//	if arg.Block != nil {
//		return arg.Block.printSorted()
//	}
//	return string(arg.Range.SliceBytes(b.File.Bytes))
//}

//func (b *ResourceBlock) sortArgGrps() {
//	for _, argGrpType := range argGrpTypes {
//		b.sortArgs(argGrpType)
//	}
//}

//func (b *ResourceBlock) sortArgs(argGrpType ArgGrpType) {
//	section := b.getSection(argGrpType)
//	args := section.Args
//	switch argGrpType {
//	case HeadMetaArgs:
//		sort.Slice(args, func(i, j int) bool {
//			return GetHeadMetaPriority(args[i].Name) > GetHeadMetaPriority(args[j].Name)
//		})
//	case TailMetaArgs:
//		sort.Slice(args, func(i, j int) bool {
//			return GetTailMetaPriority(args[i].Name) > GetTailMetaPriority(args[j].Name)
//		})
//	default:
//		sort.Slice(args, func(i, j int) bool {
//			return args[i].SortField < args[j].SortField
//		})
//	}
//}
//
//func (b *ResourceBlock) validateArgOrder(argGrpType ArgGrpType, arg *Arg) {
//	argGrp := b.getSection(argGrpType)
//	validateFunc := func(existedArg *Arg) bool {
//		switch argGrpType {
//		case HeadMetaArgs:
//			return (GetHeadMetaPriority(arg.Name) > GetHeadMetaPriority(existedArg.Name) && ComparePos(arg.Range.Start, existedArg.Range.Start) > 0) ||
//				(GetHeadMetaPriority(arg.Name) < GetHeadMetaPriority(existedArg.Name) && ComparePos(arg.Range.Start, existedArg.Range.Start) < 0)
//		case TailMetaArgs:
//			return (GetTailMetaPriority(arg.Name) > GetTailMetaPriority(existedArg.Name) && ComparePos(arg.Range.Start, existedArg.Range.Start) > 0) ||
//				(GetTailMetaPriority(arg.Name) < GetTailMetaPriority(existedArg.Name) && ComparePos(arg.Range.Start, existedArg.Range.Start) < 0)
//		default:
//			return (arg.SortField < existedArg.SortField && ComparePos(arg.Range.Start, existedArg.Range.Start) > 0) ||
//				(arg.SortField > existedArg.SortField && ComparePos(arg.Range.Start, existedArg.Range.Start) < 0)
//		}
//	}
//	for _, existedArg := range argGrp.Args {
//		if !argGrp.IsSorted {
//			break
//		}
//		if validateFunc(existedArg) {
//			argGrp.IsSorted = false
//		}
//	}
//}
//
//func (b *ResourceBlock) isArgGrpsSorted() bool {
//	var lastGrp *Args
//	for _, section := range b.getSections() {
//		if len(section.Args) == 0 {
//			continue
//		}
//		if !section.IsSorted {
//			return false
//		}
//		if lastGrp != nil && ComparePos(section.Start, lastGrp.End) <= 0 {
//			return false
//		}
//		lastGrp = section
//	}
//	return true
//}
//
//func (b *ResourceBlock) isCorrectlySplit() bool {
//	var lastGrp *Args
//	for _, section := range b.getSections() {
//		if len(section.Args) == 0 {
//			continue
//		}
//		if lastGrp != nil && section.Start.Line-lastGrp.End.Line < 2 {
//			return false
//		}
//		lastGrp = section
//	}
//	return true
//}

func (b *ResourceBlock) addHeadMetaArg(arg *Arg) {
	if b.HeadMetaArgs == nil {
		b.HeadMetaArgs = &Args{Type: HeadMetaArgs}
	}
	b.HeadMetaArgs.Add(arg)
}

func (b *ResourceBlock) addTailMetaArg(arg *Arg) {
	if b.TailMetaArgs == nil {
		b.TailMetaArgs = &Args{Type: TailMetaArgs}
	}
	b.TailMetaArgs.Add(arg)
}

func (b *ResourceBlock) addRequiredAttr(arg *Arg) {
	if b.RequiredArgs == nil {
		b.RequiredArgs = &Args{Type: RequiredAzAttrs}
	}
	b.RequiredArgs.Add(arg)
}

func (b *ResourceBlock) addOptionalAttr(arg *Arg) {
	if b.OptionalArgs == nil {
		b.OptionalArgs = &Args{Type: OptionalAttrs}
	}
	b.OptionalArgs.Add(arg)
}

func (b *ResourceBlock) addTailMetaNestedBlock(nb *NestedBlock) {
	if b.TailMetaNestedBlocks == nil {
		b.TailMetaNestedBlocks = &NestedBlocks{Type: TailMetaNestedBlocks}
	}
	b.TailMetaNestedBlocks.Add(nb)
}

func (b *ResourceBlock) addRequiredNestedBlock(nb *NestedBlock) {
	if b.RequiredNestedBlocks == nil {
		b.RequiredNestedBlocks = &NestedBlocks{Type: RequiredNestedBlocks}
	}
	b.RequiredNestedBlocks.Add(nb)
}

func (b *ResourceBlock) addOptionalNestedBlock(nb *NestedBlock) {
	if b.OptionalNestedBlocks == nil {
		b.OptionalNestedBlocks = &NestedBlocks{Type: OptionalNestedBlocks}
	}
	b.OptionalNestedBlocks.Add(nb)
}

func (b *ResourceBlock) ExpectedLayoutError() error {
	return fmt.Errorf("new error")
}
