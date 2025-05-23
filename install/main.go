package main

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
)

func main() {
	var cmds = map[string]func(){
		"install": install,
		"clean":   clean,
		"prepare": prepare,
	}
	cmd := os.Args[1]
	cmds[cmd]()
}

func prepare() {
	clean()
	goModEnsure()
}

func goModEnsure() {
	if err := exec.Command("go", "mod", "tidy").Run(); err != nil {
		panic(err.Error())
	}
	if err := exec.Command("go", "mod", "vendor").Run(); err != nil {
		panic(err.Error())
	}
}

func clean() {}

func install() {
	outputDir := fmt.Sprintf("%s/.tflint.d/plugins", os.Getenv("HOME"))
	if runtime.GOOS == "windows" {
		baseDir := os.Getenv("USERPROFILE")
		outputDir = fmt.Sprintf(`%s\.tflint.d\plugins`, baseDir)
	}
	if dir := os.Getenv("TFLINT_PLUGIN_DIR"); dir != "" {
		outputDir = dir
	} else {
		_ = os.MkdirAll(outputDir, os.ModePerm)
	}
	//#nosec G204
	cmd := exec.Command("go", "build", "-o", outputDir)
	if err := cmd.Run(); err != nil {
		panic(err.Error())
	}
}
