package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

func main() {
	var cmds map[string]func() = map[string]func(){
		"install": install,
		"clean":   clean,
		"prepare": prepare,
	}
	cmd := os.Args[1]
	cmds[cmd]()
}

func prepare() {
	clean()
	if err := exec.Command("git", "clone", "--depth", "1", "https://github.com/hashicorp/terraform-provider-azurerm.git").Run(); err != nil {
		panic(err.Error())
	}
	os.RemoveAll("./terraform-provider-azurerm/.git")
	injectProviderCode()
	if err := exec.Command("go", "mod", "tidy").Run(); err != nil {
		panic(err.Error())
	}
	if err := exec.Command("go", "mod", "vendor").Run(); err != nil {
		panic(err.Error())
	}
}

func injectProviderCode() {
	exist, err := exists("terraform-provider-azurerm/provider")
	if err != nil {
		panic(err.Error())
	}
	if !exist {
		copyInjectionCode()
	}
}

func copyInjectionCode() {
	_ = os.MkdirAll(filepath.Join("terraform-provider-azurerm", "provider"), os.ModePerm)
	dir, err := os.ReadDir("provider")
	if err != nil {
		panic(err.Error())
	}
	for _, file := range dir {
		copyFile(filepath.Join("provider", file.Name()), filepath.Join("terraform-provider-azurerm", "provider", strings.TrimSuffix(file.Name(), ".tmp")))
	}
}

func copyFile(src, dst string) {
	bytesRead, err := os.ReadFile(src)
	if err != nil {
		panic(err.Error())
	}
	err = os.WriteFile(dst, bytesRead, 0644)
	if err != nil {
		panic(err.Error())
	}
}

func exists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func clean() {
	err := os.RemoveAll("./terraform-provider-azurerm")
	if err != nil {
		panic(err.Error())
	}
	err = os.RemoveAll("./vendor")
	if err != nil {
		panic(err.Error())
	}
}

func install() {
	outputDir := fmt.Sprintf("%s/.tflint.d/plugins", os.Getenv("HOME"))
	if runtime.GOOS == "windows" {
		baseDir := os.Getenv("USERPROFILE")
		outputDir = fmt.Sprintf(`%s\.tflint.d\plugins`, baseDir)
	}
	_ = os.MkdirAll(outputDir, os.ModePerm)
	cmd := exec.Command("go", "build", "-o", outputDir)
	if err := cmd.Run(); err != nil {
		panic(err.Error())
	}
}
