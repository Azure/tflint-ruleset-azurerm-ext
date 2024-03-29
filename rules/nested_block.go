package rules

import (
	"fmt"
	"github.com/hashicorp/go-multierror"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"math"
	"sort"
	"strings"
)

// NestedBlock is a wrapper of the nested block
type NestedBlock struct {
	File                 *hcl.File
	Block                *hclsyntax.Block
	Name                 string
	SortField            string
	Range                hcl.Range
	HeadMetaArgs         *HeadMetaArgs
	RequiredArgs         *Args
	OptionalArgs         *Args
	RequiredNestedBlocks *NestedBlocks
	OptionalNestedBlocks *NestedBlocks
	ParentBlockNames     []string
	emit                 func(block Block) error
}

// CheckBlock checks the nestedBlock recursively to find the block not in order,
// and invoke the emit function on that block
func (b *NestedBlock) CheckBlock() error {
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

// DefRange gets the definition range of the nested block
func (b *NestedBlock) DefRange() hcl.Range {
	return b.Block.DefRange()
}

// CheckOrder checks whether the nestedBlock is sorted
func (b *NestedBlock) CheckOrder() bool {
	return b.checkSubSectionOrder() && b.checkGap()
}

// ToString prints the sorted block
func (b *NestedBlock) ToString() string {
	headMeta := toString(b.HeadMetaArgs)
	args := toString(b.RequiredArgs, b.OptionalArgs)
	nb := toString(b.RequiredNestedBlocks, b.OptionalNestedBlocks)
	var codes []string
	for _, c := range []string{headMeta, args, nb} {
		if c != "" {
			codes = append(codes, c)
		}
	}
	code := strings.Join(codes, "\n\n")
	blockHead := string(b.Block.DefRange().SliceBytes(b.File.Bytes))
	if strings.TrimSpace(code) == "" {
		code = fmt.Sprintf("%s {}", blockHead)
	} else {
		code = fmt.Sprintf("%s {\n%s\n}", blockHead, code)
	}
	return string(hclwrite.Format([]byte(code)))
}

// NestedBlocks is the collection of nestedBlocks with the same type
type NestedBlocks struct {
	Blocks []*NestedBlock
	Range  *hcl.Range
}

// CheckOrder checks whether this type of nestedBlocks are sorted
func (b *NestedBlocks) CheckOrder() bool {
	if b == nil {
		return true
	}
	var sortField *string
	for _, nb := range b.Blocks {
		if sortField != nil && *sortField > nb.SortField {
			return false
		}
		sortField = &nb.SortField
	}
	return true
}

// ToString prints this type of nestedBlocks in order
func (b *NestedBlocks) ToString() string {
	if b == nil {
		return ""
	}
	sortedBlocks := make([]*NestedBlock, len(b.Blocks))
	copy(sortedBlocks, b.Blocks)
	sort.Slice(sortedBlocks, func(i, j int) bool {
		return sortedBlocks[i].SortField < sortedBlocks[j].SortField
	})
	var lines []string
	for _, nb := range sortedBlocks {
		lines = append(lines, nb.ToString())
	}
	return string(hclwrite.Format([]byte(strings.Join(lines, "\n"))))
}

// GetRange returns the entire range of this type of nestedBlocks
func (b *NestedBlocks) GetRange() *hcl.Range {
	if b == nil {
		return nil
	}
	return b.Range
}

func (b *NestedBlocks) add(arg *NestedBlock) {
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

func (b *NestedBlock) nestedBlocks() []*NestedBlock {
	var nbs []*NestedBlock
	for _, subNbs := range []*NestedBlocks{b.RequiredNestedBlocks, b.OptionalNestedBlocks} {
		if subNbs != nil {
			nbs = append(nbs, subNbs.Blocks...)
		}
	}
	return nbs
}

func (b *NestedBlock) buildAttributes(attributes hclsyntax.Attributes) {
	argSchemas := queryBlockSchema(b.ParentBlockNames)
	attrs := attributesByLines(attributes)
	for _, attr := range attrs {
		attrName := attr.Name
		arg := buildAttrArg(attr, b.File)
		if IsHeadMeta(attrName) {
			b.addHeadMeta(arg)
			continue
		}
		if argSchemas == nil {
			b.addOptionalAttr(arg)
			continue
		}
		attrSchema, isAzAttr := argSchemas.Attributes[attrName]
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
		File:             b.File,
		emit:             b.emit,
	}
	nb.buildAttributes(nestedBlock.Body.Attributes)
	nb.buildNestedBlocks(nestedBlock.Body.Blocks)
	blockSchema := queryBlockSchema(b.ParentBlockNames)
	if metaArgOrUnknownBlock(blockSchema) {
		b.addOptionalNestedBlock(nb)
		return
	}

	nbSchema, ok := blockSchema.NestedBlocks[nb.Name]
	if ok && nbSchema.MinItems > 0 {
		b.addRequiredNestedBlock(nb)
	} else {
		b.addOptionalNestedBlock(nb)
	}
}

func (b *NestedBlock) addHeadMeta(arg *Arg) {
	if b.HeadMetaArgs == nil {
		b.HeadMetaArgs = &HeadMetaArgs{}
	}
	b.HeadMetaArgs.add(arg)
}

func (b *NestedBlock) addRequiredAttr(arg *Arg) {
	if b.RequiredArgs == nil {
		b.RequiredArgs = &Args{}
	}
	b.RequiredArgs.add(arg)
}

func (b *NestedBlock) addOptionalAttr(arg *Arg) {
	if b.OptionalArgs == nil {
		b.OptionalArgs = &Args{}
	}
	b.OptionalArgs.add(arg)
}

func (b *NestedBlock) addRequiredNestedBlock(nb *NestedBlock) {
	if b.RequiredNestedBlocks == nil {
		b.RequiredNestedBlocks = &NestedBlocks{}
	}
	b.RequiredNestedBlocks.add(nb)
}

func (b *NestedBlock) addOptionalNestedBlock(nb *NestedBlock) {
	if b.OptionalNestedBlocks == nil {
		b.OptionalNestedBlocks = &NestedBlocks{}
	}
	b.OptionalNestedBlocks.add(nb)
}

func (b *NestedBlock) checkSubSectionOrder() bool {
	sections := []Section{
		b.HeadMetaArgs,
		b.RequiredArgs,
		b.OptionalArgs,
		b.RequiredNestedBlocks,
		b.OptionalNestedBlocks,
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

func (b *NestedBlock) checkGap() bool {
	headMetaRange := mergeRange(b.HeadMetaArgs)
	argRange := mergeRange(b.RequiredArgs, b.OptionalArgs)
	nbRange := mergeRange(b.RequiredNestedBlocks, b.OptionalNestedBlocks)
	lastEndLine := -2
	for _, r := range []*hcl.Range{headMetaRange, argRange, nbRange} {
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
