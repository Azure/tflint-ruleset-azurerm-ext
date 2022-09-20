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

// Block is an interface offering general APIs on resource/nested block
type Block interface {
	// CheckBlock checks the resourceBlock/nestedBlock recursively to find the block not in order,
	// and invoke the emit function on that block
	CheckBlock() error

	// ToString prints the sorted block
	ToString() string

	// DefRange gets the definition range of the block
	DefRange() hcl.Range
}

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
	emit                 func(block Block) error
}

// CheckBlock checks the resource block and nested block recursively to find the block not in order,
// and invoke the emit function on that block
func (b *ResourceBlock) CheckBlock() error {
	if !b.CheckOrder() {
		return b.emit(b)
	}
	var err error
	for _, nb := range b.nestedBlocks() {
		if subErr := nb.CheckBlock(); subErr != nil {
			err = multierror.Append(err, subErr)
		}
	}
	return err
}

// DefRange gets the definition range of the resource block
func (b *ResourceBlock) DefRange() hcl.Range {
	return b.Block.DefRange()
}

// BuildResourceBlock Build the root block wrapper using hclsyntax.Block
func BuildResourceBlock(block *hclsyntax.Block, file *hcl.File,
	emitter func(block Block) error) *ResourceBlock {
	b := &ResourceBlock{
		File:             file,
		Block:            block,
		ParentBlockNames: []string{block.Type, block.Labels[0]},
		emit:             emitter,
	}
	b.buildArgs(block.Body.Attributes)
	b.buildNestedBlocks(block.Body.Blocks)
	return b
}

// CheckOrder checks whether the resourceBlock is sorted
func (b *ResourceBlock) CheckOrder() bool {
	return b.sectionsSorted() && b.gaped()
}

// ToString prints the sorted resource block
func (b *ResourceBlock) ToString() string {
	headMetaTxt := toString(b.HeadMetaArgs)
	argTxt := toString(b.RequiredArgs, b.OptionalArgs)
	nbTxt := toString(b.RequiredNestedBlocks, b.OptionalNestedBlocks)
	tailMetaArgTxt := toString(b.TailMetaArgs)
	tailMetaNbTxt := toString(b.TailMetaNestedBlocks)
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
	attrs := attributesByLines(attributes)
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

func attributesByLines(attributes hclsyntax.Attributes) []*hclsyntax.Attribute {
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
	nestedBlockName := nestedBlock.Type
	sortField := nestedBlock.Type
	if nestedBlock.Type == "dynamic" {
		nestedBlockName = nestedBlock.Labels[0]
		sortField = strings.Join(nestedBlock.Labels, "")
	}
	parentBlockNames := append(b.ParentBlockNames, nestedBlockName)
	if b.Block.Type == "dynamic" && nestedBlockName == "content" {
		parentBlockNames = b.ParentBlockNames
	}
	nb := &NestedBlock{
		Name:             nestedBlockName,
		SortField:        sortField,
		Range:            nestedBlock.Range(),
		Block:            nestedBlock,
		ParentBlockNames: parentBlockNames,
		File:             b.File,
		emit:             b.emit,
	}
	nb.buildAttributes(nestedBlock.Body.Attributes)
	nb.buildNestedBlocks(nestedBlock.Body.Blocks)
	return nb
}

func (b *ResourceBlock) buildNestedBlocks(nestedBlocks hclsyntax.Blocks) {
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

func (b *ResourceBlock) sectionsSorted() bool {
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
	return true
}

func (b *ResourceBlock) gaped() bool {
	ranges := []*hcl.Range{
		b.HeadMetaArgs.GetRange(),
		mergeRange(b.RequiredArgs, b.OptionalArgs),
		mergeRange(b.RequiredNestedBlocks, b.OptionalNestedBlocks),
		b.TailMetaArgs.GetRange(),
		b.TailMetaNestedBlocks.GetRange(),
	}
	lastEndLine := -2
	for _, r := range ranges {
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
	b.HeadMetaArgs.add(arg)
}

func (b *ResourceBlock) addTailMetaArg(arg *Arg) {
	if b.TailMetaArgs == nil {
		b.TailMetaArgs = &Args{}
	}
	b.TailMetaArgs.add(arg)
}

func (b *ResourceBlock) addRequiredAttr(arg *Arg) {
	if b.RequiredArgs == nil {
		b.RequiredArgs = &Args{}
	}
	b.RequiredArgs.add(arg)
}

func (b *ResourceBlock) addOptionalAttr(arg *Arg) {
	if b.OptionalArgs == nil {
		b.OptionalArgs = &Args{}
	}
	b.OptionalArgs.add(arg)
}

func (b *ResourceBlock) addTailMetaNestedBlock(nb *NestedBlock) {
	if b.TailMetaNestedBlocks == nil {
		b.TailMetaNestedBlocks = &NestedBlocks{}
	}
	b.TailMetaNestedBlocks.add(nb)
}

func (b *ResourceBlock) addRequiredNestedBlock(nb *NestedBlock) {
	if b.RequiredNestedBlocks == nil {
		b.RequiredNestedBlocks = &NestedBlocks{}
	}
	b.RequiredNestedBlocks.add(nb)
}

func (b *ResourceBlock) addOptionalNestedBlock(nb *NestedBlock) {
	if b.OptionalNestedBlocks == nil {
		b.OptionalNestedBlocks = &NestedBlocks{}
	}
	b.OptionalNestedBlocks.add(nb)
}
