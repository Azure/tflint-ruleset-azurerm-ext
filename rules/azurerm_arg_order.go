package rules

import (
	"fmt"
	"github.com/hashicorp/terraform-provider-azurerm/provider"
	"log"
	"regexp"
	"sort"
	"strings"

	"github.com/hashicorp/go-multierror"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/terraform-linters/tflint-plugin-sdk/tflint"
	"github.com/terraform-linters/tflint-ruleset-azurerm-ext/project"
)

type AzurermArgOrderRule struct {
	tflint.DefaultRule
}

// NewAzurermArgOrderRule returns a new rule
func NewAzurermArgOrderRule() *AzurermArgOrderRule {
	return &AzurermArgOrderRule{}
}

// Name returns the rule name
func (r *AzurermArgOrderRule) Name() string {
	return "azurerm_arg_order"
}

// Enabled returns whether the rule is enabled by default
func (r *AzurermArgOrderRule) Enabled() bool {
	return false
}

// Severity returns the rule severity
func (r *AzurermArgOrderRule) Severity() tflint.Severity {
	return tflint.WARNING
}

// Link returns the rule reference link
func (r *AzurermArgOrderRule) Link() string {
	return project.ReferenceLink(r.Name())
}

// Check checks whether the arguments/attributes in a block are sorted in azure doc order
func (r *AzurermArgOrderRule) Check(runner tflint.Runner) error {
	files, err := runner.GetFiles()
	if err != nil {
		return err
	}
	for _, file := range files {
		subErr := r.visitConfig(runner, file)
		if subErr != nil {
			err = multierror.Append(err, subErr)
		}
	}
	return err
}

func (r *AzurermArgOrderRule) visitConfig(runner tflint.Runner, file *hcl.File) error {
	body := file.Body.(*hclsyntax.Body)
	if body == nil {
		return nil
	}
	return r.visitModule(runner, body, file.Bytes)
}

func (r *AzurermArgOrderRule) visitModule(runner tflint.Runner, module *hclsyntax.Body, src []byte) error {
	if module == nil {
		return nil
	}
	var err error
	for _, block := range module.Blocks {
		switch block.Type {
		case "resource", "datasource", "provider":
			if subErr := r.visitAzBlock(runner, block, src); subErr != nil {
				err = multierror.Append(subErr)
			}
		}
	}
	return err
}

func (r *AzurermArgOrderRule) visitAzBlock(runner tflint.Runner, azBlock *hclsyntax.Block, src []byte) error {
	if azBlock == nil {
		return nil
	}
	parentBlockNameSeq := []string{azBlock.Type, azBlock.Labels[0]}
	_, err := r.visitBlock(runner, azBlock, src, parentBlockNameSeq)
	return err
}

func (r *AzurermArgOrderRule) visitBlock(runner tflint.Runner, block *hclsyntax.Block, src []byte, parentBlockNameSeq []string) ([]string, error) {
	if block == nil {
		return []string{}, nil
	}
	log.Printf("start process block %s", block.Type)
	simpleAttrValsMap, complexAttrInnerLinesMap, nestedBlockInnerLinesMap, err := r.getArgVals(runner, block, src, parentBlockNameSeq)
	sortedArgHclLines := r.getInnerSortedLines(simpleAttrValsMap, complexAttrInnerLinesMap, nestedBlockInnerLinesMap, parentBlockNameSeq)
	if !r.checkArgOrder(block, parentBlockNameSeq) {
		r.printBlockWithArgsSorted(runner, block, sortedArgHclLines)
	}
	return sortedArgHclLines, err
}

func (r *AzurermArgOrderRule) checkArgOrder(block *hclsyntax.Block, parentBlockNameSeq []string) bool {
	var argNameSeq []string
	var argStartPosSeq []hcl.Pos
	for attrName, attr := range block.Body.Attributes {
		argNameSeq = append(argNameSeq, attrName)
		argStartPosSeq = append(argStartPosSeq, attr.Range().Start)
	}
	for _, nestedBlock := range block.Body.Blocks {
		argNameSeq = append(argNameSeq, r.getBlockHead(nestedBlock))
		argStartPosSeq = append(argStartPosSeq, nestedBlock.Range().Start)
	}
	sort.Slice(argNameSeq, func(i, j int) bool {
		if argStartPosSeq[i].Line == argStartPosSeq[j].Line {
			return argStartPosSeq[i].Column < argStartPosSeq[j].Column
		}
		return argStartPosSeq[i].Line < argStartPosSeq[j].Line
	})
	_, isArgSorted := r.sortArgNames(argNameSeq, parentBlockNameSeq)
	return isArgSorted
}

// tags = {}
func (r *AzurermArgOrderRule) getComplexAttrInnerLines(runner tflint.Runner, attr *hclsyntax.Attribute, src []byte, parentBlockNameSeq []string) ([]string, error) {
	bytes := []byte(attr.Name)
	bytes = append(bytes, attr.Expr.Range().SliceBytes(src)...)
	tmpFile, _ := hclsyntax.ParseConfig(bytes, attr.SrcRange.Filename, hcl.InitialPos)
	body := tmpFile.Body.(*hclsyntax.Body)
	if body != nil {
		block := body.Blocks[0]
		return r.visitBlock(runner, block, tmpFile.Bytes, parentBlockNameSeq)
	}
	return []string{}, nil
}

func (r *AzurermArgOrderRule) getArgVals(runner tflint.Runner, block *hclsyntax.Block, src []byte, parentBlockNameSeq []string) (map[string][]string, map[string][][]string, map[string][][]string, error) {
	complexAttrReg := "\\{(?s).*\\}"
	simpleAttrValsMap := make(map[string][]string)
	complexAttrInnerLinesMap := make(map[string][][]string)
	nestedBlockInnerLinesMap := make(map[string][][]string)
	var err error
	for attrName, attr := range block.Body.Attributes {
		attrVal := string(attr.Expr.Range().SliceBytes(src))
		if isComplexAttr, _ := regexp.MatchString(complexAttrReg, attrVal); isComplexAttr {
			sortedInnerLines, subErr := r.getComplexAttrInnerLines(runner, attr, src, append(parentBlockNameSeq, attrName))
			if subErr != nil {
				err = multierror.Append(subErr)
			}
			complexAttrInnerLinesMap[attrName] = append(complexAttrInnerLinesMap[attrName], sortedInnerLines)
		} else {
			simpleAttrValsMap[attrName] = append(simpleAttrValsMap[attrName], attrVal)
		}
	}
	for _, nestedBlock := range block.Body.Blocks {
		nestedBlockName := r.getBlockHead(nestedBlock)
		sortedInnerLines, subErr := r.visitBlock(runner, nestedBlock, src, append(parentBlockNameSeq, nestedBlock.Type))
		if subErr != nil {
			err = multierror.Append(subErr)
		}
		nestedBlockInnerLinesMap[nestedBlockName] = append(nestedBlockInnerLinesMap[nestedBlockName], sortedInnerLines)
	}
	return simpleAttrValsMap, complexAttrInnerLinesMap, nestedBlockInnerLinesMap, err
}

func (r *AzurermArgOrderRule) getInnerSortedLines(simpleAttrValsMap map[string][]string, complexAttrInnerLinesMap map[string][][]string,
	nestedBlockInnerLinesMap map[string][][]string, parentBlockNameSeq []string) []string {
	var argNameSeq, attrNameSeq []string
	for simpleAttrName := range simpleAttrValsMap {
		argNameSeq = append(argNameSeq, simpleAttrName)
		attrNameSeq = append(attrNameSeq, simpleAttrName)
	}
	for complexAttrName := range complexAttrInnerLinesMap {
		argNameSeq = append(argNameSeq, complexAttrName)
		attrNameSeq = append(attrNameSeq, complexAttrName)
	}
	for nestedBlockName, _ := range nestedBlockInnerLinesMap {
		argNameSeq = append(argNameSeq, nestedBlockName)
	}

	sortedArgNameSeq, _ := r.sortArgNames(argNameSeq, parentBlockNameSeq)
	maxArgNameLen := r.getMaxStrLen(attrNameSeq)
	schemaMap := provider.GetArgSchema(parentBlockNameSeq)
	lastArgName := ""
	var argHclLines []string
	for _, argName := range sortedArgNameSeq {
		if argName == lastArgName {
			continue
		}
		if lastArgName != "" {
			lastArgSchema, islastArgInSchema := schemaMap[lastArgName]
			argSchema, isArgInSchema := schemaMap[argName]
			if islastArgInSchema && (!isArgInSchema || (lastArgSchema.Required && argSchema.Optional)) {
				argHclLines = append(argHclLines, "")
			}
		}
		lastArgName = argName
		if _, argIsSimpleAttr := simpleAttrValsMap[argName]; argIsSimpleAttr {
			argHclLines = append(argHclLines, r.buildSimpleAttrHclLines(argName, simpleAttrValsMap[argName], maxArgNameLen)...)
		}
		if _, argIsComplexAttr := complexAttrInnerLinesMap[argName]; argIsComplexAttr {
			argHclLines = append(argHclLines, r.buildComplexAttrHclLines(argName, complexAttrInnerLinesMap[argName], maxArgNameLen)...)
		}
		if _, argIsNestedBlock := nestedBlockInnerLinesMap[argName]; argIsNestedBlock {
			argHclLines = append(argHclLines, r.buildNestedBlockHclLines(argName, nestedBlockInnerLinesMap[argName])...)
		}
	}
	return argHclLines
}

func (r *AzurermArgOrderRule) sortArgNames(argNameSeq []string, parentBlockNameSeq []string) ([]string, bool) {
	argSchemaMap := provider.GetArgSchema(parentBlockNameSeq)
	orderRuleFunc := func(i, j int) bool {
		switch argNameSeq[i] {
		case "for_each":
			return true
		case "depends_on":
			return false
		}
		switch argNameSeq[j] {
		case "for_each":
			return false
		case "depends_on":
			return true
		}
		iSchema, iInSchema := argSchemaMap[argNameSeq[i]]
		jSchema, jInSchema := argSchemaMap[argNameSeq[j]]
		if (iInSchema && !jInSchema) || (!iInSchema && jInSchema) {
			return iInSchema
		}
		if (iSchema.Required && jSchema.Optional) || (iSchema.Optional && jSchema.Required) {
			return iSchema.Required
		}
		return argNameSeq[i] < argNameSeq[j]
	}
	isSorted := sort.SliceIsSorted(argNameSeq, orderRuleFunc)
	sortedArgNameSeq := argNameSeq[:]
	sort.Slice(sortedArgNameSeq, orderRuleFunc)
	return sortedArgNameSeq, isSorted
}

func (r *AzurermArgOrderRule) getBlockHead(block *hclsyntax.Block) string {
	heads := []string{block.Type}
	for _, label := range block.Labels {
		heads = append(heads, fmt.Sprintf("\"%s\"", label))
	}
	return strings.Join(heads, " ")
}

func (r *AzurermArgOrderRule) printBlockWithArgsSorted(runner tflint.Runner, block *hclsyntax.Block, sortedInnerLines []string) {
	var blockLines []string
	indent := "  "
	labelLine := fmt.Sprintf("%s {", r.getBlockHead(block))
	tailLine := "}"
	blockLines = append(blockLines, labelLine)
	for _, innerLine := range sortedInnerLines {
		if innerLine != "" {
			innerLine = indent + innerLine
		}
		blockLines = append(blockLines, innerLine)
	}
	blockLines = append(blockLines, tailLine)
	runner.EmitIssue(
		r,
		fmt.Sprintf("Arguments are not sorted in azurerm doc order, correct order is:\n%s", strings.Join(blockLines, "\n")),
		block.DefRange(),
	)
	log.Printf("\n%s", strings.Join(blockLines, "\n"))
}

func (r *AzurermArgOrderRule) buildSimpleAttrHclLines(attrName string, attrValGroup []string, maxArgNameLen int) []string {
	var simpleAttrHclLines []string
	indent := "  "
	sort.Strings(attrValGroup)
	template := fmt.Sprintf("%%-%ds = %%s", maxArgNameLen)
	for _, attrVal := range attrValGroup {
		attrValLines := strings.Split(attrVal, "\n")
		for i, line := range attrValLines {
			line = strings.TrimSpace(line)
			if i == 0 {
				attrValLines[0] = fmt.Sprintf(template, attrName, line)
			} else if i == len(attrValLines)-1 && line == "]" {
				attrValLines[i] = line
			} else {
				if line != "" {
					line = indent + line
				}
				attrValLines[i] = indent + line
			}
		}
		simpleAttrHclLines = append(simpleAttrHclLines, attrValLines...)
	}
	return simpleAttrHclLines
}

func (r *AzurermArgOrderRule) buildComplexAttrHclLines(attrName string, innerLinesGroup [][]string, maxArgNameLen int) []string {
	var complexAttrHclLines []string
	indent := "  "
	labelLine := fmt.Sprintf(fmt.Sprintf("%%-%ds = {", maxArgNameLen), attrName)
	tailLine := "}"
	sort.Slice(innerLinesGroup, func(i, j int) bool {
		return strings.Join(innerLinesGroup[i], "") < strings.Join(innerLinesGroup[j], "")
	})
	for _, innerLines := range innerLinesGroup {
		complexAttrHclLines = append(complexAttrHclLines, labelLine)
		for _, innerLine := range innerLines {
			if innerLine != "" {
				innerLine = indent + innerLine
			}
			complexAttrHclLines = append(complexAttrHclLines, innerLine)
		}
		complexAttrHclLines = append(complexAttrHclLines, tailLine)
	}
	return complexAttrHclLines
}

func (r *AzurermArgOrderRule) buildNestedBlockHclLines(nestedBlockName string, innerLinesGroup [][]string) []string {
	var nestedBlockHclLines []string
	indent := "  "
	labelLine := fmt.Sprintf("%s {", nestedBlockName)
	tailLine := "}"
	sort.Slice(innerLinesGroup, func(i, j int) bool {
		return strings.Join(innerLinesGroup[i], "") < strings.Join(innerLinesGroup[j], "")
	})
	for _, innerLines := range innerLinesGroup {
		nestedBlockHclLines = append(nestedBlockHclLines, labelLine)
		for _, innerLine := range innerLines {
			if innerLine != "" {
				innerLine = indent + innerLine
			}
			nestedBlockHclLines = append(nestedBlockHclLines, innerLine)
		}
		nestedBlockHclLines = append(nestedBlockHclLines, tailLine)
	}
	return nestedBlockHclLines
}

func (r *AzurermArgOrderRule) getMaxStrLen(strSlice []string) int {
	maxStrLen := 0
	if len(strSlice) > 0 {
		for _, str := range strSlice {
			if len(str) > maxStrLen {
				maxStrLen = len(str)
			}
		}
	}
	return maxStrLen
}
