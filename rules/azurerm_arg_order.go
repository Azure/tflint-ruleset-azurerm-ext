package rules

import (
	"fmt"
	"github.com/hashicorp/go-multierror"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/terraform-provider-azurerm/provider"
	"github.com/terraform-linters/tflint-plugin-sdk/tflint"
)

// AzurermArgOrderRule checks whether the arguments in a block are sorted in azure doc order
type AzurermArgOrderRule struct {
	DefaultRule
}

// NewAzurermArgOrderRule returns a new rule
func NewAzurermArgOrderRule() *AzurermArgOrderRule {
	return &AzurermArgOrderRule{}
}

// Name returns the rule name
func (r *AzurermArgOrderRule) Name() string {
	return "azurerm_arg_order"
}

// CheckFile checks whether the arguments in a block are sorted in azure doc order
func (r *AzurermArgOrderRule) CheckFile(runner tflint.Runner, file *hcl.File) error {
	blocks := file.Body.(*hclsyntax.Body).Blocks
	var err error
	for _, block := range blocks {
		var subErr error
		rootBlockType := provider.RootBlockType(block.Type)
		if _, isAzBlock := provider.RootBlockTypes[rootBlockType]; isAzBlock {
			subErr = r.visitAzBlock(runner, block)
		}
		if subErr != nil {
			err = multierror.Append(err, subErr)
		}
	}
	return err
}

func (r *AzurermArgOrderRule) visitAzBlock(runner tflint.Runner, azBlock *hclsyntax.Block) error {
	callback := func(block Block) error {
		return runner.EmitIssue(
			r,
			fmt.Sprintf("Arguments are expected to be sorted in following order:\n%s", block.ToString()),
			block.DefRange(),
		)
	}
	file, _ := runner.GetFile(azBlock.Range().Filename)
	b := BuildResourceBlock(azBlock, file, callback)
	return b.CheckBlock()
}
