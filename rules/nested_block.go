package rules

import (
	"fmt"
	"github.com/hashicorp/go-multierror"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/hashicorp/terraform-provider-azurerm/provider"
	"math"
	"sort"
	"strings"
)

type Block interface {
	CheckBlock() error
	ToString() string
	DefRange() hcl.Range
}

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
	callBack             func(block Block) error
}

func (b *NestedBlock) CheckBlock() error {
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

func (b *NestedBlock) DefRange() hcl.Range {
	return b.Block.DefRange()
}

func (b *NestedBlock) CheckOrder() bool {
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
	return b.checkGap()
}

func (b *NestedBlock) ToString() string {
	headMetaTxt := mergePrint(b.HeadMetaArgs)
	argsTxt := mergePrint(b.RequiredArgs, b.OptionalArgs)
	nbTxt := mergePrint(b.RequiredNestedBlocks, b.OptionalNestedBlocks)
	var txts []string
	for _, subTxt := range []string{headMetaTxt, argsTxt, nbTxt} {
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

type NestedBlocks struct {
	Blocks []*NestedBlock
	Range  *hcl.Range
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

func (b *NestedBlocks) GetRange() *hcl.Range {
	if b == nil {
		return nil
	}
	return b.Range
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

func (b *NestedBlock) buildArgGrpsWithAttrs(attributes hclsyntax.Attributes) {
	argSchemas := provider.GetArgSchema(b.ParentBlockNames)
	attrs := sortedAttributes(attributes)
	for _, attr := range attrs {
		attrName := attr.Name
		arg := buildAttrArg(attr, b.File)
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
		File:             b.File,
		callBack:         b.callBack,
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
	if b.HeadMetaArgs == nil {
		b.HeadMetaArgs = &HeadMetaArgs{}
	}
	b.HeadMetaArgs.Add(arg)
}

func (b *NestedBlock) addRequiredAttr(arg *Arg) {
	if b.RequiredArgs == nil {
		b.RequiredArgs = &Args{}
	}
	b.RequiredArgs.Add(arg)
}

func (b *NestedBlock) addOptionalAttr(arg *Arg) {
	if b.OptionalArgs == nil {
		b.OptionalArgs = &Args{}
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

//func (b *NestedBlock) Check(current hcl.Pos) (hcl.Pos, bool) {
//	if b.Range.Start.Line < current.Line {
//		return b.Range.Start, false
//	}
//	current = b.Range.Start
//	sections := []Section{
//		b.HeadMetaArgs,
//		b.RequiredArgs,
//		b.OptionalArgs,
//		b.RequiredNestedBlocks,
//		b.OptionalNestedBlocks,
//	}
//	var sorted bool
//	for _, s := range sections {
//		if current, sorted = s.Check(current); !sorted {
//			return current, false
//		}
//	}
//	return b.Range.End, b.checkGap()
//}

//func (b *NestedBlocks) Check(current hcl.Pos) (hcl.Pos, bool) {
//	if b == nil {
//		return current, true
//	}
//	if b.Range.Start.Line < current.Line {
//		return b.Range.Start, false
//	}
//	var sortField *string
//	for _, nb := range b.Blocks {
//		if sortField == nil {
//			sortField = &nb.SortField
//		}
//		if *sortField > nb.SortField {
//			return nb.Range.Start, false
//		}
//		var sorted bool
//		if current, sorted = nb.Check(current); !sorted {
//			return current, false
//		}
//		sortField = &nb.SortField
//	}
//	return b.Range.End, true
//}
