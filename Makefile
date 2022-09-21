default: build

test:
	go test $$(go list ./... | grep -v integration)

e2e:
	cd integration && go test && cd ../

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
