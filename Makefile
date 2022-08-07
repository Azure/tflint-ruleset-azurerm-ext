default: build

test:
	go test $$(go list ./... | grep -v integration)

e2e:
	cd integration && go test && cd ../

prepare:
	git submodule update --init --recursive
	sh scripts/inject.sh
	
build:	prepare
	go mod tidy
	go mod vendor
	go build

install:build
	mkdir -p ~/.tflint.d/plugins
	mv ./tflint-ruleset-azurerm-ext ~/.tflint.d/plugins

lint:
	golint --set_exit_status $$(go list ./...)
	go vet ./...

tools:
	go install golang.org/x/lint/golint@latest

updateSubmodule:
	git submodule update --init --recursive
	sh scripts/updateSubmodule.sh

.PHONY: test e2e build install lint tools updateSubmodule
