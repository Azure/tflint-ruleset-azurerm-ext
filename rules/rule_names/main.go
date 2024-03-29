package main

import (
	"fmt"
	"github.com/Azure/tflint-ruleset-azurerm-ext/rules"
	"sort"
)

func main() {
	sort.Slice(rules.Rules, func(i, j int) bool {
		return rules.Rules[i].Name() < rules.Rules[j].Name()
	})
	for _, rule := range rules.Rules {
		fmt.Printf("%s.md\n", rule.Name())
	}
}
