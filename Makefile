default: build

test:
	go test $$(go list ./... | grep -v integration)

e2e:
	cd integration && go test && cd ../

build:
	git submodule update --init --recursive
	sh scripts/inject.sh
	go mod tidy 
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

.PHONY: test e2e build install lint tools updateSubmodule
