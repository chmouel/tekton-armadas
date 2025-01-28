GOLANGCI_LINT=golangci-lint
GOFUMPT=gofumpt
LDFLAGS=
OUTPUT_DIR=bin
GO           = go
TIMEOUT_UNIT = 20m
TIMEOUT_E2E  = 45m
DEFAULT_GO_TEST_FLAGS := -v -race -failfast
GO_TEST_FLAGS :=

SHELL := bash
TOPDIR := $(shell git rev-parse --show-toplevel)
TMPDIR := $(TOPDIR)/tmp
SH_FILES := $(shell find hack/ -type f -name "*.sh" -print)
YAML_FILES := $(shell find . -not -regex '^./vendor/.*' -type f -regex ".*y[a]ml" -print)
MD_FILES := $(shell find . -type f -regex ".*md"  -not -regex '^./vendor/.*'  -not -regex '^./.vale/.*'  -not -regex "^./docs/themes/.*" -not -regex "^./.git/.*" -print)

##@ General
all: allbinaries test lint ## compile all binaries, test and lint
check: lint test ## run lint and test

.PHONY: help
help: ## print this help
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

FORCE:
.PHONY: vendor
vendor: ## generate vendor directory
	@echo Generating vendor directory
	@go mod tidy && go mod vendor

##@ Build
allbinaries: $(OUTPUT_DIR)/orchestrator-reconciler $(OUTPUT_DIR)/minion-controller

$(OUTPUT_DIR)/%: cmd/% FORCE ## compile binaries
	go build -mod=vendor $(FLAGS)  -v -o $@ ./$<

##@ Testing
test: test-unit ## Run test-unit
test-clean:  ## Clean testcache
	@echo "Cleaning test cache"
	@go clean -testcache
.PHONY: test test-unit
test-no-cache: test-clean test-unit ## Run test-unit without caching
test-unit: ## Run unit tests
	@echo "Running unit tests..."
	$(GO) test $(DEFAULT_GO_TEST_FLAGS) $(GO_TEST_FLAGS) -timeout $(TIMEOUT_UNIT) ./pkg/...

.PHONY: html-coverage
html-coverage: ## generate html coverage
	@mkdir -p tmp
	@go test -coverprofile=tmp/c.out ./.../ && go tool cover -html=tmp/c.out

##@ Linting
.PHONY: lint
lint: lint-go lint-yaml lint-md lint-shell ## run all linters

.PHONY: lint-go
lint-go: ## runs go linter on all go files
	@echo "Linting go files..."
	@$(GOLANGCI_LINT) run ./... --modules-download-mode=vendor \
							--max-issues-per-linter=0 \
							--max-same-issues=0 \
							--timeout $(TIMEOUT_UNIT)

.PHONY: lint-yaml
lint-yaml: ${YAML_FILES} ## runs yamllint on all yaml files
	@echo "Linting yaml files..."
	@yamllint -c .yamllint $(YAML_FILES)


.PHONY: lint-md
lint-md: ## runs markdownlint and vale on all markdown files
	@echo "Linting markdown files..."
	@markdownlint $(MD_FILES)
	@echo "Grammar check with vale of documentation..."
	@vale docs/content *.md --minAlertLevel=error --output=line
	@echo "CodeSpell on docs content"
	@codespell docs/content

.PHONY: lint-shell
lint-shell: ${SH_FILES} ## runs shellcheck on all shell files
	@echo "Linting shell script files..."
	@shellcheck $(SH_FILES)

.PHONY: gitlint
gitlint: ## Run gitlint
	@gitlint --commit "`git log --format=format:%H --no-merges -1`" --ignore "Merge branch"

.PHONY: pre-commit
pre-commit: ## Run pre-commit hooks script manually
	@pre-commit run --all-files

##@ Linters Fixing
.PHONY: fix-linters
fix-linters: fix-golangci-lint fix-markdownlint fix-trailing-spaces fumpt ## run all linters fixes

.PHONY: fix-markdownlint
fix-markdownlint: ## run markdownlint and fix on all markdown file
	@echo "Fixing markdown files..."
	@markdownlint --fix $(MD_FILES)

.PHONY: fix-trailing-spaces
fix-trailing-spaces: ## remove trailing spaces on all markdown and yaml file
	@sed --in-place 's/[[:space:]]\+$$//' $(MD_FILES) $(YAML_FILES)
	@[[ -n `git status --porcelain $(MD_FILES) $(YAML_FILES)` ]] && { echo "Markdowns and Yaml files has been cleaned 🧹. Cleaned Files: ";git status --porcelain $(MD_FILES) $(YAML_FILES) ;} || echo "Markdown and YAML are clean ✨"

.PHONY: fix-golangci-lint
fix-golangci-lint: ## run golangci-lint and fix on all go files
	@echo "Fixing some golangi-lint files..."
	@$(GOLANGCI_LINT) run ./... --modules-download-mode=vendor \
							--max-issues-per-linter=0 \
							--max-same-issues=0 \
							--timeout $(TIMEOUT_UNIT) \
							--fix
	@[[ -n `git status --porcelain` ]] && { echo "Go files has been cleaned 🧹. Cleaned Files: ";git status --porcelain ;} || echo "Go files are clean ✨"

.PHONY: fmt 
fmt: ## formats the GO code(excludes vendors dir)
	@go fmt `go list ./... | grep -v /vendor/`

.PHONY: fumpt 
fumpt: ## formats the GO code with gofumpt(excludes vendors dir)
	@find pkg -name '*.go'|xargs -P4 $(GOFUMPT) -w -extra

##@ Generated files
check-generated: # check if all files that needs to be generated are generated
	@git status -uno |grep -E "modified:[ ]*(vendor/|.*.golden$)" && \
		{ echo "Vendor directory or Golden files has not been generated properly, commit the change first" ; \
		  git status -uno ;	exit 1 ;} || true

.PHONY: update-golden
update-golden: ## run unit tests (updating golden files)
	@echo "Running unit tests to update golden files..."
	@./hack/update-golden.sh

.PHONY: generated
generated: update-golden fumpt ## generate all files that needs to be generated

##@ Misc

.PHONY: clean
clean: ## clean build artifacts
	rm -fR bin


