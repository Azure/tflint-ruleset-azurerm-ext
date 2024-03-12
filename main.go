package main

import (
	"github.com/Azure/tflint-ruleset-azurerm-ext/project"
	"github.com/Azure/tflint-ruleset-azurerm-ext/rules"
	"github.com/terraform-linters/tflint-plugin-sdk/plugin"
	"github.com/terraform-linters/tflint-plugin-sdk/tflint"
)

var version = "0.6.0"

func main() {
	project.Version = version
	plugin.Serve(&plugin.ServeOpts{
		RuleSet: &tflint.BuiltinRuleSet{
			Name:    "azurerm-ext",
			Version: project.Version,
			Rules:   rules.Rules,
		},
	})
}
