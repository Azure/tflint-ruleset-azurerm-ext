package rules

import (
	"fmt"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-provider-azurerm/provider"
	"github.com/terraform-linters/tflint-plugin-sdk/helper"
	"reflect"
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
	return tflint.NOTICE
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
	issue := new(helper.Issue)
	parentBlockNames := []string{azBlock.Type, azBlock.Labels[0]}
	if provider.GetArgSchema(parentBlockNames) == nil {
		return nil
	}
	r.visitBlock(runner, azBlock, parentBlockNames, issue)
	if !IsIssueEmpty(issue) {
		return runner.EmitIssue(issue.Rule, issue.Message, issue.Range)
	}
	return nil
}

func (r *AzurermArgOrderRule) visitBlock(runner tflint.Runner, block *hclsyntax.Block, parentBlockNames []string, issue *helper.Issue) string {
	file, _ := runner.GetFile(block.Range().Filename)
	argSchemas := provider.GetArgSchema(parentBlockNames)
	argGrps := r.getArgGrps(block, argSchemas)
	isGapNeeded := false
	var sortedArgHclTxts []string
	for _, args := range argGrps {
		if len(args) == 0 {
			continue
		}
		if isGapNeeded {
			sortedArgHclTxts = append(sortedArgHclTxts, "")
		}
		for _, arg := range args {
			if arg.Block != nil {
				if arg.Name == "content" && block.Type == "dynamic" {
					sortedArgHclTxts = append(sortedArgHclTxts, r.visitBlock(runner, arg.Block, parentBlockNames, issue))
				} else {
					sortedArgHclTxts = append(sortedArgHclTxts, r.visitBlock(runner, arg.Block, append(parentBlockNames, arg.Name), issue))
				}
			} else {
				sortedArgHclTxts = append(sortedArgHclTxts, string(arg.Range.SliceBytes(file.Bytes)))
			}
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
	if !r.checkArgOrder(argGrps) {
		issue.Rule = r
		issue.Message = fmt.Sprintf("Arguments are expected to be sorted in following order:\n%s", sortedBlockHclTxt)
		issue.Range = block.DefRange()
	}
	return sortedBlockHclTxt
}

func (r *AzurermArgOrderRule) getArgGrps(block *hclsyntax.Block, argSchemas map[string]*schema.Schema) [][]Arg {
	var headMetaArgs, requiredAzAttrs, optionalAzAttrs, nonAzAttrs, requiredAzNestedBlocks, optionalAzNestedBlocks, nonAzNestedBlocks, tailMetaArgs []Arg
	for attrName, attr := range block.Body.Attributes {
		arg := Arg{
			Name:      attrName,
			SortField: attrName,
			Range:     attr.SrcRange,
		}
		if IsHeadMeta(attrName) {
			headMetaArgs = append(headMetaArgs, arg)
		} else if IsTailMeta(attrName) {
			tailMetaArgs = append(tailMetaArgs, arg)
		} else {
			if attrSchema, isAzAttr := argSchemas[attrName]; isAzAttr {
				if attrSchema.Required {
					requiredAzAttrs = append(requiredAzAttrs, arg)
				} else {
					optionalAzAttrs = append(optionalAzAttrs, arg)
				}
			} else {
				nonAzAttrs = append(nonAzAttrs, arg)
			}
		}
	}
	for _, nestedBlock := range block.Body.Blocks {
		var nestedBlockName, sortField string
		if nestedBlock.Type == "dynamic" {
			nestedBlockName = nestedBlock.Labels[0]
			sortField = strings.Join(nestedBlock.Labels, "")
		} else {
			nestedBlockName = nestedBlock.Type
			sortField = nestedBlock.Type
		}
		arg := Arg{
			Name:      nestedBlockName,
			SortField: sortField,
			Range:     hcl.Range{},
			Block:     nestedBlock,
		}
		if IsHeadMeta(nestedBlockName) {
			headMetaArgs = append(headMetaArgs, arg)
		} else if IsTailMeta(nestedBlockName) {
			tailMetaArgs = append(tailMetaArgs, arg)
		} else {
			if blockSchema, isAzNestedBlock := argSchemas[nestedBlockName]; isAzNestedBlock {
				if blockSchema.Required {
					requiredAzNestedBlocks = append(requiredAzNestedBlocks, arg)
				} else {
					optionalAzNestedBlocks = append(optionalAzNestedBlocks, arg)
				}
			} else {
				nonAzNestedBlocks = append(nonAzNestedBlocks, arg)
			}
		}
	}
	sort.SliceStable(headMetaArgs, func(i, j int) bool {
		return GetHeadMetaPriority(headMetaArgs[i].Name) > GetHeadMetaPriority(headMetaArgs[j].Name)
	})
	sort.SliceStable(tailMetaArgs, func(i, j int) bool {
		return GetTailMetaPriority(tailMetaArgs[i].Name) > GetTailMetaPriority(tailMetaArgs[j].Name)
	})
	nonMetaArgGrps := [][]Arg{requiredAzAttrs, optionalAzAttrs, nonAzAttrs, requiredAzNestedBlocks, optionalAzNestedBlocks, nonAzNestedBlocks}
	for _, nonMetaArgs := range nonMetaArgGrps {
		sort.SliceStable(nonMetaArgs, func(i, j int) bool {
			return nonMetaArgs[i].SortField < nonMetaArgs[j].SortField
		})
	}
	argGrps := [][]Arg{headMetaArgs}
	argGrps = append(argGrps, nonMetaArgGrps...)
	argGrps = append(argGrps, tailMetaArgs)
	return argGrps
}

func (r *AzurermArgOrderRule) checkArgOrder(sortedArgGrps [][]Arg) bool {
	var lastArgGrp, sortedArgs []Arg
	isCorrectLayout := true
	for _, argGrp := range sortedArgGrps {
		if isCorrectLayout && len(lastArgGrp) > 0 && len(argGrp) > 0 {
			if argGrp[0].Range.Start.Line-lastArgGrp[len(lastArgGrp)-1].Range.End.Line < 2 {
				isCorrectLayout = false
			}
		}
		if len(argGrp) > 0 {
			lastArgGrp = argGrp
		}
		sortedArgs = append(sortedArgs, argGrp...)
	}
	isCorrectLayout = isCorrectLayout && reflect.DeepEqual(sortedArgs, GetArgsWithOriginalOrder(sortedArgs))
	return isCorrectLayout
}

func (r *AzurermArgOrderRule) getBlockHead(block *hclsyntax.Block) string {
	heads := []string{block.Type}
	for _, label := range block.Labels {
		heads = append(heads, fmt.Sprintf("\"%s\"", label))
	}
	return strings.Join(heads, " ")
}
