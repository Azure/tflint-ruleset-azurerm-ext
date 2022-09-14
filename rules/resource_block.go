package rules

import (
	"fmt"
	"github.com/hashicorp/go-multierror"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/hashicorp/terraform-provider-azurerm/provider"
	"sort"
	"strings"
)

// ResourceBlock is the wrapper of a resource block
type ResourceBlock struct {
	File                 *hcl.File
	Block                *hclsyntax.Block
	HeadMetaArgs         *HeadMetaArgs
	RequiredArgs         *Args
	OptionalArgs         *Args
	RequiredNestedBlocks *NestedBlocks
	OptionalNestedBlocks *NestedBlocks
	TailMetaArgs         *Args
	TailMetaNestedBlocks *NestedBlocks
	ParentBlockNames     []string
	callBack             func(block Block) error
}

func (b *ResourceBlock) CheckBlock() error {
	if !b.CheckOrder() {
		return b.callBack(b)
	}
	var err error
	for _, nb := range b.nestedBlocks() {
		if subErr := nb.CheckBlock(); subErr != nil {
			err = multierror.Append(err, subErr)
		}
	}
	return err
}

func (b *ResourceBlock) DefRange() hcl.Range {
	return b.Block.DefRange()
}

// BuildResourceBlock Build the root block wrapper using hclsyntax.Block
func BuildResourceBlock(block *hclsyntax.Block, file *hcl.File,
	callBack func(block Block) error) *ResourceBlock {
	b := &ResourceBlock{
		File:             file,
		Block:            block,
		ParentBlockNames: []string{block.Type, block.Labels[0]},
		callBack:         callBack,
	}
	b.buildArgs(block.Body.Attributes)
	b.buildArgGrpsWithNestedBlocks(block.Body.Blocks)
	return b
}

func (b *ResourceBlock) CheckOrder() bool {
	sections := []Section{
		b.HeadMetaArgs,
		b.RequiredArgs,
		b.OptionalArgs,
		b.RequiredNestedBlocks,
		b.OptionalNestedBlocks,
		b.TailMetaArgs,
		b.TailMetaNestedBlocks,
	}
	lastEndLine := -1
	for _, s := range sections {
		if !s.CheckOrder() {
			return false
		}
		r := s.GetRange()
		if r == nil {
			continue
		}
		if r.Start.Line <= lastEndLine {
			return false
		}
		lastEndLine = r.End.Line
	}
	return b.checkGap()
}

func (b *ResourceBlock) ToString() string {
	headMetaTxt := mergePrint(b.HeadMetaArgs)
	argTxt := mergePrint(b.RequiredArgs, b.OptionalArgs)
	nbTxt := mergePrint(b.RequiredNestedBlocks, b.OptionalNestedBlocks)
	tailMetaArgTxt := mergePrint(b.TailMetaArgs)
	tailMetaNbTxt := mergePrint(b.TailMetaNestedBlocks)
	var txts []string
	for _, subTxt := range []string{headMetaTxt, argTxt, nbTxt, tailMetaArgTxt, tailMetaNbTxt} {
		if subTxt != "" {
			txts = append(txts, subTxt)
		}
	}
	txt := strings.Join(txts, "\n\n")
	blockHead := string(b.Block.DefRange().SliceBytes(b.File.Bytes))
	if strings.TrimSpace(txt) == "" {
		txt = fmt.Sprintf("%s {}", blockHead)
	} else {
		txt = fmt.Sprintf("%s {\n%s\n}", blockHead, txt)
	}
	return string(hclwrite.Format([]byte(txt)))
}

func (b *ResourceBlock) nestedBlocks() []*NestedBlock {
	var nbs []*NestedBlock
	for _, subNbs := range []*NestedBlocks{b.RequiredNestedBlocks, b.OptionalNestedBlocks, b.TailMetaNestedBlocks} {
		if subNbs != nil {
			nbs = append(nbs, subNbs.Blocks...)
		}
	}
	return nbs
}

func (b *ResourceBlock) buildArgs(attributes hclsyntax.Attributes) {
	argSchemas := provider.GetArgSchema(b.ParentBlockNames)
	attrs := sortedAttributes(attributes)
	for _, attr := range attrs {
		attrName := attr.Name
		arg := buildAttrArg(attr, b.File)
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
		File:             b.File,
		callBack:         b.callBack,
	}
	nb.buildArgGrpsWithAttrs(nestedBlock.Body.Attributes)
	nb.buildNestedBlocks(nestedBlock.Body.Blocks)
	return nb
}

func (b *ResourceBlock) buildArgGrpsWithNestedBlocks(nestedBlocks hclsyntax.Blocks) {
	argSchemas := provider.GetArgSchema(b.ParentBlockNames)
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

func (b *ResourceBlock) checkGap() bool {
	headMetaRange := mergeRange(b.HeadMetaArgs)
	argRange := mergeRange(b.RequiredArgs, b.OptionalArgs)
	nbRange := mergeRange(b.RequiredNestedBlocks, b.OptionalNestedBlocks)
	tailMetaArgRange := mergeRange(b.TailMetaArgs)
	tailMetaNbRange := mergeRange(b.TailMetaNestedBlocks)
	lastEndLine := -2
	for _, r := range []*hcl.Range{headMetaRange, argRange, nbRange, tailMetaArgRange, tailMetaNbRange} {
		if r == nil {
			continue
		}
		if r.Start.Line-lastEndLine < 2 {
			return false
		}
		lastEndLine = r.End.Line
	}
	return true
}

func (b *ResourceBlock) addHeadMetaArg(arg *Arg) {
	if b.HeadMetaArgs == nil {
		b.HeadMetaArgs = &HeadMetaArgs{}
	}
	b.HeadMetaArgs.Add(arg)
}

func (b *ResourceBlock) addTailMetaArg(arg *Arg) {
	if b.TailMetaArgs == nil {
		b.TailMetaArgs = &Args{}
	}
	b.TailMetaArgs.Add(arg)
}

func (b *ResourceBlock) addRequiredAttr(arg *Arg) {
	if b.RequiredArgs == nil {
		b.RequiredArgs = &Args{}
	}
	b.RequiredArgs.Add(arg)
}

func (b *ResourceBlock) addOptionalAttr(arg *Arg) {
	if b.OptionalArgs == nil {
		b.OptionalArgs = &Args{}
	}
	b.OptionalArgs.Add(arg)
}

func (b *ResourceBlock) addTailMetaNestedBlock(nb *NestedBlock) {
	if b.TailMetaNestedBlocks == nil {
		b.TailMetaNestedBlocks = &NestedBlocks{}
	}
	b.TailMetaNestedBlocks.Add(nb)
}

func (b *ResourceBlock) addRequiredNestedBlock(nb *NestedBlock) {
	if b.RequiredNestedBlocks == nil {
		b.RequiredNestedBlocks = &NestedBlocks{}
	}
	b.RequiredNestedBlocks.Add(nb)
}

func (b *ResourceBlock) addOptionalNestedBlock(nb *NestedBlock) {
	if b.OptionalNestedBlocks == nil {
		b.OptionalNestedBlocks = &NestedBlocks{}
	}
	b.OptionalNestedBlocks.Add(nb)
}

func (b *ResourceBlock) ExpectedLayoutError() error {
	return fmt.Errorf("new error")
}

//// Check recursively Checks whether the args in a block and its nested block are correctly sorted
//func (b *ResourceBlock) Check(current hcl.Pos) (hcl.Pos, bool) {
//	sections := []Section{
//		b.HeadMetaArgs,
//		b.RequiredArgs,
//		b.OptionalArgs,
//		b.RequiredNestedBlocks,
//		b.OptionalNestedBlocks,
//		b.TailMetaArgs,
//		b.TailMetaNestedBlocks,
//	}
//	var sorted bool
//	for _, s := range sections {
//		if current, sorted = s.Check(current); !sorted {
//			return current, false
//		}
//	}
//	return *new(hcl.Pos), b.checkGap()
//}
