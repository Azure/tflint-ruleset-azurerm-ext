package rules

import (
	"fmt"
	"github.com/hashicorp/go-multierror"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/terraform-provider-azurerm/provider"
	"github.com/terraform-linters/tflint-plugin-sdk/tflint"
	"github.com/terraform-linters/tflint-ruleset-azurerm-ext/project"
	"strings"
)

// AzurermResourceTagRule checks whether the tags arg is specified if supported
type AzurermResourceTagRule struct {
	tflint.DefaultRule
}

// NewAzurermResourceTagRule returns a new rule
func NewAzurermResourceTagRule() *AzurermResourceTagRule {
	return &AzurermResourceTagRule{}
}

// Name returns the rule name
func (r *AzurermResourceTagRule) Name() string {
	return "azurerm_resource_tag"
}

// Enabled returns whether the rule is enabled by default
func (r *AzurermResourceTagRule) Enabled() bool {
	return false
}

// Severity returns the rule severity
func (r *AzurermResourceTagRule) Severity() tflint.Severity {
	return tflint.NOTICE
}

// Link returns the rule reference link
func (r *AzurermResourceTagRule) Link() string {
	return project.ReferenceLink(r.Name())
}

// Check checks whether the tags arg is specified if supported
func (r *AzurermResourceTagRule) Check(runner tflint.Runner) error {
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

func (r *AzurermResourceTagRule) visitConfig(runner tflint.Runner, file *hcl.File) error {
	body := file.Body.(*hclsyntax.Body)
	return r.visitModule(runner, body)
}

func (r *AzurermResourceTagRule) visitModule(runner tflint.Runner, module *hclsyntax.Body) error {
	var err error
	for _, block := range module.Blocks {
		switch provider.RootBlockType(block.Type) {
		case provider.Resource:
			if subErr := r.visitAzResource(runner, block); subErr != nil {
				err = multierror.Append(err, subErr)
			}
		}
	}
	return err
}

func (r *AzurermResourceTagRule) visitAzResource(runner tflint.Runner, azBlock *hclsyntax.Block) error {
	parentBlockNames := []string{azBlock.Type, azBlock.Labels[0]}
	argSchemas := provider.GetArgSchema(parentBlockNames)
	if argSchemas == nil {
		return nil
	}
	r.visitBlock(runner, azBlock, parentBlockNames)
	return nil
}

func (r *AzurermResourceTagRule) visitBlock(runner tflint.Runner, block *hclsyntax.Block, parentBlockNames []string) error {
	var err error
	if block.Type != "dynamic" {
		argSchemas := provider.GetArgSchema(parentBlockNames)
		if _, isTagSupported := argSchemas["tags"]; isTagSupported {
			if _, isTagSet := block.Body.Attributes["tags"]; !isTagSet {
				nestedBlockSeq := ""
				if len(parentBlockNames) > 2 {
					nestedBlockSeq = fmt.Sprintf("nested block `%s` of ", strings.Join(parentBlockNames[2:], " "))
				}
				err = runner.EmitIssue(
					r,
					fmt.Sprintf("`tags` argument is not set but supported in %s%s `%s`", nestedBlockSeq, parentBlockNames[0], parentBlockNames[1]),
					block.DefRange(),
				)
			}
		}
	}
	for _, nestedBlock := range block.Body.Blocks {
		var subErr error
		if nestedBlock.Type == "dynamic" {
			subErr = r.visitBlock(runner, nestedBlock, append(parentBlockNames, nestedBlock.Labels[0]))
		} else if block.Type == "dynamic" {
			if nestedBlock.Type == "content" {
				subErr = r.visitBlock(runner, nestedBlock, parentBlockNames)
			}
		} else {
			subErr = r.visitBlock(runner, nestedBlock, append(parentBlockNames, nestedBlock.Type))
		}
		if subErr != nil {
			err = multierror.Append(err, subErr)
		}
	}
	return err
}
