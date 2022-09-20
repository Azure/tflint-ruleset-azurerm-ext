package rules

import "github.com/terraform-linters/tflint-plugin-sdk/tflint"

// Rules is a list of all rules
var Rules = []tflint.Rule{
	NewAzurermArgOrderRule(),
	NewAzurermResourceTagRule(),
}

func init() {
	loadIgnoreConfig()
}
