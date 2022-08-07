default: build

test:
	go test $$(go list ./... | grep -v integration)

e2e:
	cd integration && go test && cd ../

prepare:clean
	git clone https://github.com/hashicorp/terraform-provider-azurerm.git
	rm -rf terraform-provider-azurerm/.git
	sh scripts/inject.sh
	go mod tidy
	go mod vendor
	
build:	prepare
	go build

install:build
	mkdir -p ~/.tflint.d/plugins
	mv ./tflint-ruleset-azurerm-ext ~/.tflint.d/plugins

lint:
	golint --set_exit_status $$(go list ./...)
	go vet ./...

tools:
	go install golang.org/x/lint/golint@latest

clean:
	rm -rf ./terraform-provider-azurerm ./vendor

.PHONY: test e2e build install lint tools updateSubmodule
