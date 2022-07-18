package rules

import (
	"fmt"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/terraform-provider-azurerm/provider"
	"github.com/terraform-linters/tflint-plugin-sdk/tflint"
	"github.com/terraform-linters/tflint-ruleset-azurerm-ext/project"
	"log"
	"sort"
	"strings"
)

type AzurermArgsOrderRule struct {
	tflint.DefaultRule
}

// NewAzurermArgsOrderRule returns a new rule
func NewAzurermArgsOrderRule() *AzurermArgsOrderRule {
	return &AzurermArgsOrderRule{}
}

// Name returns the rule name
func (r *AzurermArgsOrderRule) Name() string {
	return "azurerm_args_order"
}

// Enabled returns whether the rule is enabled by default
func (r *AzurermArgsOrderRule) Enabled() bool {
	return false
}

// Severity returns the rule severity
func (r *AzurermArgsOrderRule) Severity() tflint.Severity {
	return tflint.WARNING
}

// Link returns the rule reference link
func (r *AzurermArgsOrderRule) Link() string {
	return project.ReferenceLink(r.Name())
}

// Check checks whether the arguments/attributes in a block are sorted in azure doc order
func (r *AzurermArgsOrderRule) Check(runner tflint.Runner) error {
	files, _ := runner.GetFiles()
	var arrArr []error
	for name, file := range files {
		if err := r.checkAzurermDocOrder(runner, name, file); err != nil {
			log.Printf("[ERROR] %s: %s", name, err.Error())
			arrArr = append(arrArr, err)
		}
	}
	if arrArr != nil {
		return arrArr[0]
	}
	return nil
}

func (r *AzurermArgsOrderRule) checkAzurermDocOrder(runner tflint.Runner, filename string, file *hcl.File) error {
	if strings.HasSuffix(filename, ".json") {
		return nil
	}
	var lastIdentIndex int
	tokens, diags := hclsyntax.LexConfig(file.Bytes, filename, hcl.InitialPos)
	if diags.HasErrors() {
		return diags
	}

	lines := strings.Split(string(file.Bytes), "\n")
	for i, line := range lines {
		lines[i] = line + "\n"
	}
	for i := 0; i < len(tokens); i++ {
		if tokens[i].Type == hclsyntax.TokenIdent {
			lastIdentIndex = i
		} else if tokens[i].Type == hclsyntax.TokenOBrace {
			if _, ok := provider.RootBlockTypes[provider.RootBlockType(tokens[lastIdentIndex].Bytes)]; ok {
				for j := lastIdentIndex + 1; j < i; j++ {
					if tokens[j].Type == hclsyntax.TokenQuotedLit {
						i, _ = r.checkArgsOrder(runner, []int{j}, tokens, i+1, lines, provider.RootBlockType(tokens[lastIdentIndex].Bytes))
						break
					}
				}
			}
		}
	}
	return nil
}

// checkArgsAlphaOrder checks whether the arguments in a block are sorted in azure doc order recursively
func (r *AzurermArgsOrderRule) checkArgsOrder(runner tflint.Runner,
	blockNameTokenIndSeq []int, tokens hclsyntax.Tokens, startIndex int, lines []string, rootBlockType provider.RootBlockType) (int, string) {
	var blockNameSeq, realArgSeq, remainArgSeq []string
	realArgText := make(map[string][]string)
	var lastIdentTokenIndex, i, lastLine, iLineBegin, iLineEnd int
	var innerArgsText, currArg string
	for i = startIndex; i < len(tokens); i++ {
		if realArgSeq != nil {
			currArg = realArgSeq[len(realArgSeq)-1]
		}
		if tokens[i].Type == hclsyntax.TokenNewline || tokens[i].Type == hclsyntax.TokenComment {
			continue
		} else if r.isReceiver(tokens, i) {
			lastIdentTokenIndex = i
			realArgSeq = append(realArgSeq, string(tokens[i].Bytes))
			currArg = string(tokens[i].Bytes)
			realArgText[currArg] = append(realArgText[currArg], "")
		} else if tokens[i].Type == hclsyntax.TokenOBrace {
			i, innerArgsText = r.checkArgsOrder(runner, append(blockNameTokenIndSeq, lastIdentTokenIndex), tokens, i+1, lines, rootBlockType)
			if realArgSeq != nil {
				realArgText[currArg][len(realArgText[currArg])-1] += innerArgsText
			}
			lastLine = tokens[i-1].Range.End.Line
		} else if tokens[i].Type == hclsyntax.TokenCBrace {
			break
		}
		iLineBegin = tokens[i].Range.Start.Line - 1
		iLineEnd = tokens[i].Range.End.Line - 1
		if realArgSeq != nil && iLineBegin > lastLine {
			realArgText[currArg][len(realArgText[currArg])-1] += strings.Join(lines[iLineBegin:iLineEnd+1], "")
		}
		lastLine = iLineEnd
	}

	// get arg schema
	for _, ind := range blockNameTokenIndSeq {
		blockNameSeq = append(blockNameSeq, string(tokens[ind].Bytes))
	}
	argsMap := provider.GetArgSchema(blockNameSeq, rootBlockType)
	sortedArgsName := provider.GetDocSortedArgNames(argsMap)

	// sort arg text
	sortedArgsText := ""
	isRequiredExist := false
	isOptionalExist := false
	for _, argName := range sortedArgsName {
		if _, ok := realArgText[argName]; ok {
			if !isRequiredExist && argsMap[argName].Required {
				isRequiredExist = true
			}
			if !isOptionalExist && argsMap[argName].Optional {
				if isRequiredExist {
					sortedArgsText += "\n"
				}
				isOptionalExist = true
			}
			sort.Strings(realArgText[argName])
			sortedArgsText += strings.Join(realArgText[argName], "")
		}
	}
	for argName := range realArgText {
		if _, ok := argsMap[argName]; !ok {
			remainArgSeq = append(remainArgSeq, argName)
		}
	}

	isRemainArgAlphaOrder := true
	if remainArgSeq != nil {
		if isRequiredExist || isOptionalExist {
			sortedArgsText += "\n"
		}
		isRemainArgAlphaOrder = sort.StringsAreSorted(remainArgSeq)
		sort.Strings(remainArgSeq)
		for _, argName := range remainArgSeq {
			sortedArgsText += strings.Join(realArgText[argName], "")
		}
	}

	if tokens[i].Range.Start.Line-1 > lastLine {
		sortedArgsText += strings.Join(lines[tokens[i].Range.Start.Line-1:tokens[i].Range.End.Line], "")
	}
	owner := tokens[blockNameTokenIndSeq[len(blockNameTokenIndSeq)-1]]
	sortedArgsTextForPrint := strings.Join(lines[owner.Range.Start.Line-1:tokens[startIndex-1].Range.End.Line], "") + sortedArgsText
	if !r.isArgsInDocOrder(realArgSeq, sortedArgsName) || !isRemainArgAlphaOrder {
		runner.EmitIssue(
			r,
			fmt.Sprintf("Arguments are not sorted in azurerm doc order, correct order is:\n%s", sortedArgsTextForPrint),
			owner.Range,
		)
	}
	return i, sortedArgsText
}

func (r *AzurermArgsOrderRule) isArgsInDocOrder(realArgSeq []string, sortedArgsSeq []string) bool {
	if sortedArgsSeq == nil {
		return true
	}
	var newArgSeq []string
	argMap := make(map[string]bool)
	for _, argName := range sortedArgsSeq {
		argMap[argName] = true
	}
	for _, argName := range realArgSeq {
		if _, ok := argMap[argName]; ok {
			newArgSeq = append(newArgSeq, argName)
		}
	}
	if newArgSeq == nil {
		return true
	}
	nextUnsortedIndex := 0
	for _, argName := range sortedArgsSeq {
		if nextUnsortedIndex == len(newArgSeq) {
			break
		}
		if argName == newArgSeq[nextUnsortedIndex] {
			nextUnsortedIndex += 1
		}
	}
	if nextUnsortedIndex == len(newArgSeq) {
		return true
	}
	return false
}

// isReceiver checks whether an identifier token is not on the right side of an expression
func (r *AzurermArgsOrderRule) isReceiver(tokens hclsyntax.Tokens, index int) bool {
	if tokens[index].Type != hclsyntax.TokenIdent {
		return false
	}
	isNewLine := false
	for i := index - 1; i >= 0; i-- {
		if tokens[i].Type == hclsyntax.TokenNewline {
			isNewLine = true
		} else if tokens[i].Type == hclsyntax.TokenEqual {
			if !isNewLine {
				return false
			}
		}
	}
	return true
}
