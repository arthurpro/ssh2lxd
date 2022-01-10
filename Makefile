version ?= devel
edition ?= ce
githash ?= $(shell git rev-parse --short HEAD)
release ?= 0

arch ?=
archs = amd64 arm64

ifneq ($(githash), "")
	version = $(shell git describe --exact-match --tags $(githash))
endif

sysgroups ?= wheel,lxd

name = ssh2lxd
url = https://github.com/artefactcorp/ssh2lxd
description = "SSH server for LXD containers"
maintainer = ssh2lxd@artefactcorp.com
license = GPL-3.0
iteration = $(release)

binfile ?= /bin/$(name)
envfile ?= /etc/sysconfig/$(name)
servicefile ?= /lib/systemd/system/$(name).service
docfile ?= /usr/share/doc/$(name)/README.md
licensefile ?= /usr/share/licenses/$(name)/LICENSE

#export GOPATH = /tmp/.go
#export GOOS = linux
export GOARCH = $(arch)
export CGO_ENABLED = 0

#export ARCH = $(shell arch)

RELEASE_DIR := ./release

default: build

cross-release:
	@for arch in $(archs); do \
		$(MAKE) release arch=$$arch; \
	done

v:
	@echo $(version) $(githash)
	@env

fmt:
	@go fmt
	@for d in cmd lxd server util; do \
		find ./$$d -type f -name "*.go" -exec go fmt {} \; ;\
	done

build:
	go build \
		-ldflags="-X 'ssh2lxd.version=$(version)' -X 'ssh2lxd.edition=$(edition)' -X 'ssh2lxd.githash=$(githash)' -X 'ssh2lxd.flagGroups=$(sysgroups)'" \
		-o ./$(name) \
		cmd/ssh2lxd/ssh2lxd.go
	@./$(name) -h

ubuntu: sysgroups := adm,lxd
ubuntu: envfile := /etc/default/$(name)

linux: binfile := /$(shell basename $(binfile))
linux: servicefile := /$(shell basename $(servicefile))
linux: docfile := /$(shell basename $(docfile))
linux: licensefile := /$(shell basename $(licensefile))

el ubuntu linux: GOOS=linux GOPATH=/tmp/.go
el ubuntu linux:
	@rm -rf ./build
	@mkdir -p ./build
	@if [[ "$@" != "linux" ]]; then \
		mkdir -p ./build{/bin,/lib/systemd/system,$(shell dirname $(envfile)),$(shell dirname $(docfile)),$(shell dirname $(licensefile))}; \
		cp packaging/$(name).env ./build$(envfile); \
	fi
	@sed -e 's#^EnvironmentFile.*$$#EnvironmentFile=-$(envfile)#' ./packaging/$(name).service >./build$(servicefile)
	@cp README.md ./build$(docfile)
	@cp LICENSE ./build$(licensefile)

	go build \
		-ldflags="-X 'ssh2lxd.version=$(version)' -X 'ssh2lxd.edition=$(edition)' -X 'ssh2lxd.githash=$(githash)' -X 'ssh2lxd.flagGroups=$(sysgroups)'" \
		-o ./build$(binfile) \
		cmd/ssh2lxd/ssh2lxd.go
	@strip ./build$(binfile)
	@if [[ -x /bin/upx ]]; then \
		upx ./build/$(binfile); \
	fi
	#@./build$(binfile) -h

before-release:
	@mkdir -p $(RELEASE_DIR)

release: before-release rpm deb tar
	@echo
	@ls -1 $(RELEASE_DIR)

rpm: el
deb: ubuntu

rpm deb:
	fpm -s dir -t $@ -C ./build \
		--architecture $(arch) \
		--name $(name) \
		--version $(version) \
		--iteration $(iteration) \
		--category Applications/Internet \
		--url $(url) \
		--maintainer $(maintainer) \
		--config-files /etc \
		--license $(license) \
		--description $(description) \
		--after-install=packaging/after-all.sh \
		--after-remove=packaging/after-all.sh \
		.
	@mv *.$@ $(RELEASE_DIR)

tar: linux
	@tar -C ./build -czf $(RELEASE_DIR)/$(name)-$(version)-$(release)-linux-$(GOARCH).tar.gz .

upx:
	upx ./build/$(binfile)
	./build/$(binfile) -v

artefact:
	@echo "rm -rf ./ansible ./cmd ./docker* ./packaging ./release ./vendor ./build.sh ./go.sum ./Makefile"

.PHONY: build
