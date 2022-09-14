package rules

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"math"
	"sort"
	"strings"
)

type Section interface {
	CheckOrder() bool
	ToString() string
	GetRange() *hcl.Range
}

func mergePrint(sections ...Section) string {
	var lines []string
	for _, section := range sections {
		line := section.ToString()
		if line != "" {
			lines = append(lines, line)
		}
	}
	return strings.Join(lines, "\n")
}

func mergeRange(sections ...Section) *hcl.Range {
	start := hcl.Pos{Line: math.MaxInt}
	end := hcl.Pos{Line: -1}
	filename := ""
	isNil := true
	for _, section := range sections {
		r := section.GetRange()
		if r == nil {
			continue
		}
		isNil = false
		if filename == "" {
			filename = r.Filename
		}
		if r.Start.Line < start.Line {
			start = r.Start
		}
		if r.End.Line > end.Line {
			end = r.End
		}
	}
	if isNil {
		return nil
	}
	return &hcl.Range{
		Filename: filename,
		Start:    start,
		End:      end,
	}
}

// Arg includes attr and nested block defined in a block
type Arg struct {
	Name  string
	Range hcl.Range
	File  *hcl.File
}

func (a *Arg) ToString() string {
	return string(hclwrite.Format(a.Range.SliceBytes(a.File.Bytes)))
}

// Args is the collection of args with the same type
type Args struct {
	Args  []*Arg
	Range *hcl.Range
}

func (a *Args) Add(arg *Arg) {
	a.Args = append(a.Args, arg)
	a.updateRange(arg)
}

func (a *Args) CheckOrder() bool {
	if a == nil {
		return true
	}
	var name *string
	for _, arg := range a.Args {
		if name != nil && *name > arg.Name {
			return false
		}
		name = &arg.Name
	}
	return true
}

func (a *Args) ToString() string {
	if a == nil {
		return ""
	}
	sortedArgs := make([]*Arg, len(a.Args))
	copy(sortedArgs, a.Args)
	sort.Slice(sortedArgs, func(i, j int) bool {
		return sortedArgs[i].Name < sortedArgs[j].Name
	})
	var lines []string
	for _, arg := range sortedArgs {
		lines = append(lines, arg.ToString())
	}
	return string(hclwrite.Format([]byte(strings.Join(lines, "\n"))))
}

func (a *Args) GetRange() *hcl.Range {
	if a == nil {
		return nil
	}
	return a.Range
}

// HeadMetaArgs is the collection of args with the same type
type HeadMetaArgs struct {
	Args  []*Arg
	Range *hcl.Range
}

func (a *HeadMetaArgs) Add(arg *Arg) {
	a.Args = append(a.Args, arg)
	a.updateRange(arg)
}

func (a *HeadMetaArgs) CheckOrder() bool {
	if a == nil {
		return true
	}
	score := math.MaxInt
	for _, arg := range a.Args {
		if score < headMetaArgPriority[arg.Name] {
			return false
		}
		score = headMetaArgPriority[arg.Name]
	}
	return true
}

func (a *HeadMetaArgs) ToString() string {
	if a == nil {
		return ""
	}
	sortedArgs := make([]*Arg, len(a.Args))
	copy(sortedArgs, a.Args)
	sort.Slice(sortedArgs, func(i, j int) bool {
		return headMetaArgPriority[sortedArgs[i].Name] > headMetaArgPriority[sortedArgs[j].Name]
	})
	var lines []string
	for _, arg := range sortedArgs {
		lines = append(lines, arg.ToString())
	}
	return string(hclwrite.Format([]byte(strings.Join(lines, "\n"))))
}

func (a *HeadMetaArgs) GetRange() *hcl.Range {
	if a == nil {
		return nil
	}
	return a.Range
}

func (a *Args) updateRange(arg *Arg) {
	if a.Range == nil {
		a.Range = &hcl.Range{
			Filename: arg.Range.Filename,
			Start:    hcl.Pos{Line: math.MaxInt},
			End:      hcl.Pos{Line: -1},
		}
	}
	if a.Range.Start.Line > arg.Range.Start.Line {
		a.Range.Start = arg.Range.Start
	}
	if a.Range.End.Line < arg.Range.End.Line {
		a.Range.End = arg.Range.End
	}
}

func (a *HeadMetaArgs) updateRange(arg *Arg) {
	if a.Range == nil {
		a.Range = &hcl.Range{
			Filename: arg.Range.Filename,
			Start:    hcl.Pos{Line: math.MaxInt},
			End:      hcl.Pos{Line: -1},
		}
	}
	if a.Range.Start.Line > arg.Range.Start.Line {
		a.Range.Start = arg.Range.Start
	}
	if a.Range.End.Line < arg.Range.End.Line {
		a.Range.End = arg.Range.End
	}
}

func buildAttrArg(attr *hclsyntax.Attribute, file *hcl.File) *Arg {
	return &Arg{
		Name:  attr.Name,
		Range: attr.SrcRange,
		File:  file,
	}
}

//func (a *HeadMetaArgs) Check(current hcl.Pos) (hcl.Pos, bool) {
//	if a == nil {
//		return current, true
//	}
//	if a.Range.Start.Line < current.Line {
//		return a.Range.Start, false
//	}
//
//	score := -1
//	for _, arg := range a.Args.Args {
//		if headMetaArgPriority[arg.Name] < score {
//			return arg.Range.Start, false
//		}
//		score = headMetaArgPriority[arg.Name]
//	}
//	return a.Range.End, true
//}

//func (a *Args) Check(current hcl.Pos) (hcl.Pos, bool) {
//	if a == nil {
//		return current, true
//	}
//	if a.Range.Start.Line < current.Line {
//		return a.Range.Start, false
//	}
//
//	var name *string
//	for _, arg := range a.Args {
//		if name == nil {
//			name = &arg.Name
//		}
//		if *name > arg.Name {
//			return arg.Range.Start, false
//		}
//		name = &arg.Name
//	}
//	return a.Range.End, true
//}
