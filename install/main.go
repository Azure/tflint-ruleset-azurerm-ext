package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/google/go-github/v47/github"
	"github.com/hashicorp/go-getter/v2"
	"golang.org/x/oauth2"
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
	latest := latestTag()
	prepareTerraformAzurermProviderCode(latest)
	injectProviderCode()
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

func prepareTerraformAzurermProviderCode(latest string) {
	repoUrl := fmt.Sprintf("github.com/hashicorp/terraform-provider-azurerm?ref=%s&&depth=1", latest)
	_, err := getter.Get(context.Background(), "terraform-provider-azurerm", repoUrl)
	if err != nil {
		panic(fmt.Sprintf("cannot clone repo:%s", err.Error()))
	}
}

func latestTag() string {
	c := gitClient()
	tags, _, err := c.Repositories.ListTags(context.TODO(), "hashicorp", "terraform-provider-azurerm", &github.ListOptions{
		Page:    0,
		PerPage: 10,
	})
	if err != nil {
		panic(err.Error())
	}
	if len(tags) == 0 {
		panic("no terraform-azurerm-provider tags found")
	}
	latest := tags[0].GetName()
	return latest
}

func gitClient() *github.Client {
	var client *github.Client
	token := os.Getenv("TOKEN")
	if token != "" {
		ts := oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: token},
		)
		tc := oauth2.NewClient(context.TODO(), ts)
		client = github.NewClient(tc)
	} else {
		client = github.NewClient(nil)
	}
	return client
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
		bytesRead, err := os.ReadFile(filepath.Join("provider", file.Name()))
		if err != nil {
			panic(err.Error())
		}
		err = os.WriteFile(filepath.Join("terraform-provider-azurerm", "provider", strings.TrimSuffix(file.Name(), ".tmp")), bytesRead, 0600)
		if err != nil {
			panic(err.Error())
		}
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
