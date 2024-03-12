default: build

test:
	go test ./..

e2e: install
	cd integration/basic && tflint --chdir=.

prepare:clean
	go run install/main.go prepare
	
build:	prepare
	go build

install: prepare
	go run install/main.go install

lint:
	golint --set_exit_status $$(go list ./...)
	go vet ./...

tools:
	go install golang.org/x/lint/golint@latest

clean:
	go run install/main.go clean

.PHONY: test e2e build install lint tools
