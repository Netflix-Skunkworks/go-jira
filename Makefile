NAME=jira
GO?=go

OS=$(shell uname -s)
ifeq ($(filter CYGWIN%,$(OS)),$(OS))
	export CWD=$(shell cygpath -wa .)
	export SEP=\\
	export CYGWIN=winsymlinks:native
	BIN ?= $(GOBIN)$(SEP)$(NAME).exe
else
	export CWD=$(shell pwd)
	export SEP=/
	BIN ?= $(GOBIN)$(SEP)$(NAME)
endif

DIST=$(CWD)$(SEP)dist

GOBIN ?= $(CWD)

CURVER ?= $(patsubst v%,%,$(shell [ -d .git ] && git describe --abbrev=0 --tags || grep ^\#\# CHANGELOG.md | awk '{print $$2; exit}'))
LDFLAGS:= -w
VERSION ?= development

build:
	$(GO) build -gcflags="-e" -v -ldflags "$(LDFLAGS) -s" -o '$(BIN)' cmd/jira/main.go

vet:
	@$(GO) vet .
	@$(GO) vet ./jiracli
	@$(GO) vet ./jiracmd
	@$(GO) vet ./jiradata
	@$(GO) vet ./cmd/jira

lint:
	@$(GO) get github.com/golang/lint/golint
	@golint .
	@golint ./jiracli
	@golint ./jiracmd
	@golint ./jiradata
	@golint ./cmd/jira

all:
	GO111MODULE=off $(GO) get -u github.com/mitchellh/gox
	rm -rf dist
	mkdir -p dist
	gox -ldflags="-w -s" -ldflags="-X 'github.com/go-jira/jira.VERSION=$(VERSION)'" -output="dist/github.com/go-jira/jira-{{.OS}}-{{.Arch}}" -osarch="darwin/amd64 linux/386 linux/amd64 windows/386 windows/amd64" ./cmd/jira

install:
	${MAKE} GOBIN=$$HOME/bin build

NEWVER ?= $(shell echo $(CURVER) | awk -F. '{print $$1"."$$2"."$$3+1}')
TODAY  := $(shell date +%Y-%m-%d)

changes:
	@git log --pretty=format:"* %s [%cn] [%h]" --no-merges ^v$(CURVER) HEAD *.go jiracli/*.go jiradata/*.go jiracmd/*.go cmd/*/*.go *.lock | grep -vE 'gofmt|go fmt|version bump'

update-changelog:
	@echo "# Changelog" > CHANGELOG.md.new; \
	echo >> CHANGELOG.md.new; \
	echo "## $(NEWVER) - $(TODAY)" >> CHANGELOG.md.new; \
	echo >> CHANGELOG.md.new; \
	$(MAKE) --no-print-directory --silent changes | \
	perl -pe 's{\[([a-f0-9]+)\]}{[[$$1](https://github.com/Netflix-Skunkworks/go-jira/commit/$$1)]}g' | \
	perl -pe 's{\#(\d+)}{[#$$1](https://github.com/Netflix-Skunkworks/go-jira/issues/$$1)}g' >> CHANGELOG.md.new; \
	tail -n +2 CHANGELOG.md >> CHANGELOG.md.new; \
    perl -pi -e 's{VERSION = "$(CURVER)"}{VERSION = "$(NEWVER)"}' jira.go; \
	mv CHANGELOG.md.new CHANGELOG.md; \
	$(NULL)

update-usage:
	@perl -pi -e 'undef $$/; s|\n```\nusage.*?```|"\n```\n".qx{./jira --help}."```"|esg' README.md

release:
	make update-usage
	git diff --exit-code --quiet README.md || git commit -m "Updated Usage" README.md
	git commit -m "Updated Changelog" CHANGELOG.md
	git commit -m "version bump" jira.go
	git tag v$(NEWVER)
	git push --tags

version:
	@echo $(CURVER)

clean: clean-password-store
	rm -rf ./$(NAME)

clean-password-store:
	rm -f "$(CWD)/_t/.password-store/GoJira/api-token:gojira@corybennett.org.gpg"
	rm -f "$(CWD)/_t/.password-store/GoJira/api-token:mothra@corybennett.org.gpg"

test-password-store:
	ln -s "api-token__gojira@corybennett.org.gpg" "$(CWD)/_t/.password-store/GoJira/api-token:gojira@corybennett.org.gpg"
	ln -s "api-token__mothra@corybennett.org.gpg" "$(CWD)/_t/.password-store/GoJira/api-token:mothra@corybennett.org.gpg"

prove: test-password-store
	chmod -R g-rwx,o-rwx $(CWD)/_t/.gnupg
	OSHT_VERBOSE=1 prove -v _t/*.t

generate:
	cd schemas && go run ./fetch-schemas.go
	grep -h slipscheme jiradata/*.go | grep json | sort | uniq | awk -F\/\/ '{print $$2}' | while read cmd; do $$cmd; done
