#
# Makefile for building all things related to this repo
#
NAME := protoc-gen-es6rpc
ORG := pinpt
PKG := $(ORG)/$(NAME)

SHELL := /bin/bash
BASEDIR := $(shell echo $${PWD})
BUILDDIR := $(BASEDIR)/build
UNAME_S := $(shell uname -s)
LDFLAGS := -ldflags="-s -w"

.PHONY: default clean setup build install

default: help

clean:
	@rm -rf $(BUILDDIR)

build:
	@mkdir -p $(BUILDDIR)
	@go build -o $(BUILDDIR)/$(NAME) $(LDFLAGS) $(BASEDIR)/$(NAME).go

setup:
	@go get github.com/golang/protobuf/proto

install: build
	@cp $(BUILDDIR)/$(NAME) $(GOPATH)/bin

.PHONY: help

help:
	@echo -e '\033[0;35mUsage: make <TARGETS>\033[0m'
	@echo ''
	@echo -e '\033[0;32mMain Targets:\033[0m'
	@echo -e '    \033[0;33msetup\033[0m              \033[0;34mSetup and check the local dev environment.\033[0m'
	@echo -e '    \033[0;33mbuild\033[0m              \033[0;34mGenerate source code for plugin.\033[0m'
	@echo -e '    \033[0;33minstall\033[0m            \033[0;34mInstall plugin to $(GOPATH)/bin\033[0m'
	@echo -e '    \033[0;33mclean\033[0m              \033[0;34mRemove temporary build directory.\033[0m'
	@echo ''
