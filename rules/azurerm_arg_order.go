package rules

import (
	"fmt"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-provider-azurerm/provider"
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
	return r.visitModule(runner, body)
}

func (r *AzurermArgOrderRule) visitModule(runner tflint.Runner, module *hclsyntax.Body) error {
	if module == nil {
		return nil
	}
	var err error
	for _, block := range module.Blocks {
		rootBlockType := provider.RootBlockType(block.Type)
		if _, isAzBlock := provider.RootBlockTypes[rootBlockType]; isAzBlock {
			if subErr := r.visitAzBlock(runner, block); subErr != nil {
				err = multierror.Append(subErr)
			}
		}
	}
	return err
}

func (r *AzurermArgOrderRule) visitAzBlock(runner tflint.Runner, azBlock *hclsyntax.Block) error {
	if azBlock == nil {
		return nil
	}
	parentBlockNames := []string{azBlock.Type, azBlock.Labels[0]}
	if provider.GetArgSchema(parentBlockNames) == nil {
		return nil
	}
	_, err := r.visitBlock(runner, azBlock, parentBlockNames)
	return err
}

func (r *AzurermArgOrderRule) visitBlock(runner tflint.Runner, block *hclsyntax.Block, parentBlockNames []string) (string, error) {
	if block == nil {
		return "", nil
	}
	argSchemas := provider.GetArgSchema(parentBlockNames)
	argHclTxts, err := r.getArgHclTxts(runner, block, parentBlockNames)
	if err != nil {
		return "", err
	}
	var argNames, sortedArgHclTxts, sortedArgNames []string
	for argName := range argHclTxts {
		argNames = append(argNames, argName)
	}
	localSortedArgNameGrps := r.getSortedArgNames(argNames, argSchemas)
	isGapNeeded := false
	for _, localSortedArgNames := range localSortedArgNameGrps {
		if len(localSortedArgNames) == 0 {
			continue
		}
		sortedArgNames = append(sortedArgNames, localSortedArgNames...)
		if isGapNeeded {
			sortedArgHclTxts = append(sortedArgHclTxts, "")
		}
		for _, argName := range localSortedArgNames {
			sortedArgHclTxts = append(sortedArgHclTxts, argHclTxts[argName])
		}
		isGapNeeded = true
	}
	sortedBlockHclTxt := strings.Join(sortedArgHclTxts, "\n")
	if strings.TrimSpace(sortedBlockHclTxt) == "" {
		sortedBlockHclTxt = fmt.Sprintf("%s {}", r.getBlockHead(block))
	} else {
		sortedBlockHclTxt = fmt.Sprintf("%s {\n%s\n}", r.getBlockHead(block), sortedBlockHclTxt)
	}
	sortedBlockHclTxt = string(hclwrite.Format([]byte(sortedBlockHclTxt)))
	if !r.checkArgOrder(block, sortedArgNames) {
		runner.EmitIssue(
			r,
			fmt.Sprintf("Arguments are not sorted in azurerm doc order, correct order is:\n%s", sortedBlockHclTxt),
			block.DefRange(),
		)
	}
	return sortedBlockHclTxt, err
}

// todo: sort keys for map type attr or the elem of attr
func (r *AzurermArgOrderRule) visitAttr(runner tflint.Runner, attr *hclsyntax.Attribute) (string, error) {
	file, err := runner.GetFile(attr.Range().Filename)
	if err != nil {
		return "", err
	}
	attrHclTxt := string(attr.Range().SliceBytes(file.Bytes))
	return attrHclTxt, nil
}

func (r *AzurermArgOrderRule) getArgHclTxts(runner tflint.Runner, block *hclsyntax.Block,
	parentBlockNames []string) (map[string]string, error) {
	var err error
	argHclTxtsGroups := make(map[string][]string)
	argHclTxts := make(map[string]string)
	for attrName, attr := range block.Body.Attributes {
		hclTxt, subErr := r.visitAttr(runner, attr)
		if subErr != nil {
			err = multierror.Append(err, subErr)
		}
		argHclTxtsGroups[attrName] = append(argHclTxtsGroups[attrName], hclTxt)
	}
	for _, nestedBlock := range block.Body.Blocks {
		nestedBlockNameForSort := r.getBlockHead(nestedBlock)
		var hclTxt string
		var subErr error
		if nestedBlock.Type == "dynamic" {
			hclTxt, subErr = r.visitBlock(runner, nestedBlock, append(parentBlockNames, nestedBlock.Labels[0]))
			nestedBlockNameForSort = nestedBlock.Labels[0]
		} else if block.Type == "dynamic" && nestedBlock.Type == "content" {
			hclTxt, subErr = r.visitBlock(runner, nestedBlock, parentBlockNames)
		} else {
			hclTxt, subErr = r.visitBlock(runner, nestedBlock, append(parentBlockNames, nestedBlock.Type))
		}
		if subErr != nil {
			err = multierror.Append(err, subErr)
		}
		argHclTxtsGroups[nestedBlockNameForSort] = append(argHclTxtsGroups[nestedBlockNameForSort], hclTxt)
	}
	for argName, hclTxtsGroups := range argHclTxtsGroups {
		argHclTxts[argName] = strings.Join(hclTxtsGroups, "\n")
	}
	return argHclTxts, err
}

func (r *AzurermArgOrderRule) getSortedArgNames(argNames []string, argSchemas map[string]*schema.Schema) [][]string {
	getSortedAzArgNamesByOptionality := func(isRequired bool) []string {
		var sortedArgNames []string
		for _, argName := range argNames {
			argSchema, isAzArg := argSchemas[argName]
			if !isAzArg || argSchema.Required != isRequired {
				continue
			}
			sortedArgNames = append(sortedArgNames, argName)
		}
		sort.Strings(sortedArgNames)
		return sortedArgNames
	}
	sortedRequiredAzArgNames := getSortedAzArgNamesByOptionality(true)
	sortedOptionalAzArgNames := getSortedAzArgNamesByOptionality(false)
	var nonAzArgNames []string
	for _, argName := range argNames {
		if _, isAzArg := argSchemas[argName]; !isAzArg {
			nonAzArgNames = append(nonAzArgNames, argName)
		}
	}
	sortedHeadMetaArgNames, sortedNonAzOrMetaArgNames, sortedTailMetaArgNames := r.getSortedNonAzArgNames(nonAzArgNames)
	return [][]string{sortedHeadMetaArgNames, sortedRequiredAzArgNames, sortedOptionalAzArgNames, sortedNonAzOrMetaArgNames, sortedTailMetaArgNames}
}

func (r *AzurermArgOrderRule) getSortedNonAzArgNames(nonAzArgNames []string) ([]string, []string, []string) {
	headMetaArgPriority := map[string]int{"for_each": 1, "count": 1, "provider": 0}
	tailMetaArgPriority := map[string]int{"lifecycle": 1, "depends_on": 0}
	var headMetaArgNames, nonAzOrMetaArgNames, tailMetaArgNames, dynamicBlockNames []string
	for _, argName := range nonAzArgNames {
		if _, isHeadMeta := headMetaArgPriority[argName]; isHeadMeta {
			headMetaArgNames = append(headMetaArgNames, argName)
		} else if _, isTailMeta := tailMetaArgPriority[argName]; isTailMeta {
			tailMetaArgNames = append(tailMetaArgNames, argName)
		} else {
			if strings.Split(argName, " ")[0] == "dynamic" {
				dynamicBlockNames = append(dynamicBlockNames, argName)
			} else {
				nonAzOrMetaArgNames = append(nonAzOrMetaArgNames, argName)
			}
		}
	}
	sort.Slice(headMetaArgNames, func(i, j int) bool {
		return headMetaArgPriority[headMetaArgNames[i]] < headMetaArgPriority[headMetaArgNames[j]]
	})
	sort.Slice(tailMetaArgNames, func(i, j int) bool {
		return tailMetaArgPriority[tailMetaArgNames[i]] < tailMetaArgPriority[tailMetaArgNames[j]]
	})
	sort.Strings(dynamicBlockNames)
	sort.Strings(nonAzOrMetaArgNames)
	tailMetaArgNames = append(dynamicBlockNames, tailMetaArgNames...)
	return headMetaArgNames, nonAzOrMetaArgNames, tailMetaArgNames
}

func (r *AzurermArgOrderRule) checkArgOrder(block *hclsyntax.Block, sortedArgNames []string) bool {
	var argNames []string
	var argStartPos []hcl.Pos
	for attrName, attr := range block.Body.Attributes {
		argNames = append(argNames, attrName)
		argStartPos = append(argStartPos, attr.Range().Start)
	}
	for _, nestedBlock := range block.Body.Blocks {
		argNames = append(argNames, r.getBlockHead(nestedBlock))
		argStartPos = append(argStartPos, nestedBlock.Range().Start)
	}
	sort.Slice(argNames, func(i, j int) bool {
		if argStartPos[i].Line == argStartPos[j].Line {
			return argStartPos[i].Column < argStartPos[j].Column
		}
		return argStartPos[i].Line < argStartPos[j].Line
	})
	return CompareSliceOrder(argNames, sortedArgNames)
}

func (r *AzurermArgOrderRule) getBlockHead(block *hclsyntax.Block) string {
	heads := []string{block.Type}
	for _, label := range block.Labels {
		heads = append(heads, fmt.Sprintf("\"%s\"", label))
	}
	return strings.Join(heads, " ")
}

func CompareSliceOrder(real []string, expect []string) bool {
	if len(real) < len(expect) {
		return false
	}
	i, j := 0, 0
	for i < len(real) && j < len(expect) {
		if real[i] == expect[j] {
			j++
		}
		i++
	}
	if j == len(expect) {
		return true
	}
	return false
}
