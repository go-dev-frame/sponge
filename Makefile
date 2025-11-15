SHELL := /bin/bash

PROJECT_NAME := "github.com/go-dev-frame/sponge"
PKG := "$(PROJECT_NAME)"
PKG_LIST := $(shell go list ${PKG}/... | grep -v /vendor/ | grep -v /api/ | grep -v /cmd/)

# delete the templates code start
.PHONY: install
# Installation of dependent plugins
install:
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	go install github.com/envoyproxy/protoc-gen-validate@latest
	go install github.com/srikrsna/protoc-gen-gotag@latest
	go install github.com/go-dev-frame/sponge/cmd/protoc-gen-go-gin@latest
	go install github.com/go-dev-frame/sponge/cmd/protoc-gen-go-rpc-tmpl@latest
	go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2@latest
	go install github.com/pseudomuto/protoc-gen-doc/cmd/protoc-gen-doc@latest
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install github.com/swaggo/swag/cmd/swag@v1.8.12
	go install github.com/ofabry/go-callvis@latest
	go install golang.org/x/pkgsite/cmd/pkgsite@latest
# delete the templates code end


.PHONY: ci-lint
# Check code formatting, naming conventions, security, maintainability, etc. the rules in the .golangci.yml file
ci-lint:
	@gofmt -s -w .
	golangci-lint run ./...


.PHONY: test
# Test *_test.go files, the parameter -count=1 means that caching is disabled
test:
	go test -count=1 -short ${PKG_LIST}


.PHONY: cover
# Generate test coverage
cover:
	go test -short -coverprofile=cover.out -covermode=atomic ${PKG_LIST}
	go tool cover -html=cover.out


.PHONY: graph
# Generate interactive visual function dependency graphs
graph:
	@echo "generating graph ......"
	@cp -f cmd/serverNameExample_mixExample/main.go .
	go-callvis -skipbrowser -format=svg -nostd -file=serverNameExample_mixExample github.com/go-dev-frame/sponge
	@rm -f main.go serverNameExample_mixExample.gv

# delete the templates code start
.PHONY: docs
# Generate swagger docs, only for ⓵ Create web server based on sql
docs:
	@bash scripts/swag-docs.sh
# delete the templates code end

.PHONY: proto
# Generate *.go and template code by proto files, the default is all the proto files in the api directory. you can specify the proto file, multiple files are separated by commas, e.g. make proto FILES=api/user/v1/user.proto
proto:
	@bash scripts/protoc.sh $(FILES)
	go mod tidy
	@gofmt -s -w .


.PHONY: proto-doc
# Generate doc from *.proto files
proto-doc:
	@bash scripts/proto-doc.sh


.PHONY: build
# Build serverNameExample_mixExample for linux amd64 binary
build:
	@echo "building 'serverNameExample_mixExample', linux binary file will output to 'cmd/serverNameExample_mixExample'"
	@cd cmd/serverNameExample_mixExample && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build

# delete the templates code start

.PHONY: build-sponge
# Build sponge for linux amd64 binary
build-sponge:
	@echo "building 'sponge', linux binary file will output to 'cmd/sponge'"
	@cd cmd/sponge && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "all=-s -w"

.PHONY: image-build-sponge
# Build a sponge docker image, e.g. make image-build-sponge TAG=v1.5.8
image-build-sponge:
	@echo "build a sponge docker image'"
	@cd cmd/sponge/scripts && bash build-sponge-image.sh  $(TAG)

# delete the templates code end

.PHONY: run
# Build and run service, you can specify the configuration file, e.g. make run Config=configs/dev.yml
run:
	@bash scripts/run.sh $(Config)


.PHONY: run-nohup
# Run service with nohup in local, e.g. make run-nohup CMD=start, you can specify the configuration file, e.g. make run-nohup CMD=start Config=configs/dev.yml, if you want to stop the service, e.g. make run-nohup CMD=stop
CMD ?= start
run-nohup:
	@bash scripts/run-nohup.sh $(CMD) $(Config)


.PHONY: run-docker
# Run service in local docker, if you want to update the service, run the make run-docker command again
run-docker: image-build-local
	@bash scripts/deploy-docker.sh


.PHONY: binary-package
# Packaged binary files
binary-package: build
	@bash scripts/binary-package.sh


.PHONY: deploy-binary
# Deploy binary to remote linux server, e.g. make deploy-binary USER=root PWD=123456 IP=192.168.1.10
deploy-binary: binary-package
	@expect scripts/deploy-binary.sh $(USER) $(PWD) $(IP)


.PHONY: image-build-local
# Build image for local docker, tag=latest, use binary files to build
image-build-local: build
	@bash scripts/image-build-local.sh


.PHONY: image-build
# Build image for remote repositories, use binary files to build, e.g. make image-build REPO_HOST=addr TAG=latest
image-build:
	@bash scripts/image-build.sh $(REPO_HOST) $(TAG)


.PHONY: image-build2
# Build image for remote repositories, phase II build, e.g. make image-build2 REPO_HOST=addr TAG=latest
image-build2:
	@bash scripts/image-build2.sh $(REPO_HOST) $(TAG)


.PHONY: image-push
# Push docker image to remote repositories, e.g. make image-push REPO_HOST=addr TAG=latest
image-push:
	@bash scripts/image-push.sh $(REPO_HOST) $(TAG)


.PHONY: deploy-k8s
# Deploy service to k8s
deploy-k8s:
	@bash scripts/deploy-k8s.sh


.PHONY: image-build-rpc-test
# Build grpc test image for remote repositories, e.g. make image-build-rpc-test REPO_HOST=addr TAG=latest
image-build-rpc-test:
	@bash scripts/image-rpc-test.sh $(REPO_HOST) $(TAG)


.PHONY: patch
# Patch some dependent code. 1. Add database initialization code, e.g. make patch TYPE=mysql, also supports mongodb, postgresql, sqlite. 2. Add dependent types.proto file, e.g. make patch TYPE=types-pb
patch:
	@bash scripts/patch.sh $(TYPE)


.PHONY: copy-proto
# Copy proto file from the grpc server directory, multiple directories or proto files separated by commas. default is to copy all proto files, e.g. make copy-proto SERVER=yourServerDir, copy specified proto files, e.g. make copy-proto SERVER=yourServerDir PROTO_FILE=yourProtoFile1,yourProtoFile2
copy-proto:
	@sponge patch copy-proto --server-dir=$(SERVER) --proto-file=$(PROTO_FILE)


.PHONY: modify-proto-pkg-name
# Modify the 'package' and 'go_package' names of all proto files in the 'api' directory
modify-proto-pkg-name:
	@sponge patch modify-proto-package --dir=api --server-dir=.


.PHONY: update-config
# Update internal/config code base on yaml file
update-config:
	@sponge config --server-dir=.


.PHONY: clean
# Clean binary file, cover.out, template file
clean:
	@rm -vrf cmd/serverNameExample_mixExample/serverNameExample_mixExample*
	@rm -vrf cover.out
	@rm -vrf main.go serverNameExample_mixExample.gv
	@rm -vrf internal/ecode/*.go.gen*
	@rm -vrf internal/routers/*.go.gen*
	@rm -vrf internal/handler/*.go.gen*
	@rm -vrf internal/service/*.go.gen*
	@rm -rf serverNameExample-binary.tar.gz
	@echo "clean finished"


# Show help
help:
	@echo ''
	@echo 'Usage:'
	@echo '  make <target>'
	@echo ''
	@echo 'Targets:'
	@awk '/^[a-zA-Z\-_0-9]+:/ { \
	helpMessage = match(lastLine, /^# (.*)/); \
		if (helpMessage) { \
			helpCommand = substr($$1, 0, index($$1, ":")-1); \
			helpMessage = substr(lastLine, RSTART + 2, RLENGTH); \
			printf "\033[1;36m  %-22s\033[0m %s\n", helpCommand,helpMessage; \
		} \
	} \
	{ lastLine = $$0 }' $(MAKEFILE_LIST)

.DEFAULT_GOAL := all
