package main

import (
	"github.com/terraform-linters/tflint-plugin-sdk/plugin"
	"github.com/terraform-linters/tflint-plugin-sdk/tflint"
	"github.com/terraform-linters/tflint-ruleset-azurerm-ext/project"
	"github.com/terraform-linters/tflint-ruleset-azurerm-ext/rules"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		RuleSet: &tflint.BuiltinRuleSet{
			Name:    "azurerm-ext",
			Version: project.Version,
			Rules:   rules.Rules,
		},
	})
}
