GIT ?= git
GO ?= go
GOFMT ?= gofmt
DLV ?= dlv
GREP ?= grep
AWK ?= awk
GOLINT ?= golangci-lint

GIT_HASH = $(shell git rev-parse --revs-only --short HEAD 2>/dev/null)
BUILD_ARGS = -ldflags "-X github.com/veraison/corim-store/pkg/build.Build=$(GIT_HASH)"

GOLINT_ARGS ?= run --build-tags test --timeout=3m -E dupl -E gocritic -E staticcheck -E prealloc

ifeq ($(VERBOSE),1)
	TEST_ARGS = -v -args -trace
endif

export TEST_DB_FILE

.PHONY: build
build:
	$(GO) build $(BUILD_ARGS) github.com/veraison/corim-store/cmd/corim-store

.PHONY: test
test:
	$(GO) test -tags=test ./... $(TEST_ARGS)

.PHONY: integ-test
integ-test:
	@scripts/integration-tests.sh setup
	@scripts/integration-tests.sh run
	@scripts/integration-tests.sh teardown

.PHONY: fmt format gofmt
fmt format gofmt: */*/*.go */*/*/*.go
	$(GOFMT) -w */*/*.go */*/*/*.go

.PHONY: lint
lint:
	$(GOLINT) $(GOLINT_ARGS)

.PHONY: cover coverage
cover coverage coverage.out:
	@scripts/coverage.sh

.PHONY: cover-report coverage-report report
cover-report coverage-report report: coverage.out
	$(GO) tool cover -html=coverage.out

.PHONY: presubmit
presubmit:
	# TODO: increase coverage
	@$(MAKE) -s test && $(MAKE) -s lint && COVERAGE_THRESHOLD=30% $(MAKE) -s coverage && $(MAKE) -s format
	@if ! $(GIT) diff-index --quiet HEAD --; then \
		echo -e "\033[1;31mUNCOMMITED CHANGES!\033[0m"; \
		exit 2; \
	 fi

define HELP_TEXT
Usage: make [target]

Targets:
    build:
        Build the project.
    test:
        Run unit tests. Use VERBOSE=1 for SQL traces. Set TEST_DB_FILE to
        specify the path for the sqlite DB file to be used for tests (by default
        in-memory DB is used).
    integ-test:
        Run integration tests. These rely on a Docker container running database
        servers.
    lint:
        Run golangci-lint.
    fmt, format, gofmt:
        Run gofmt -w on Go sourcefiles, fixing any formatting issues.
    cover, coverage:
        Report overall test coverage percentage and generate coverage.out. You
        can specify a space-separated list of packages for which coverage will
        not be checked by setting IGNORE_COVERAGE. You can specify alternative
        threshold by setting COVERAGE_THRESHOLD.
    cover-report, coverage-report, report:
        Open a detailed HTML coverage report in default browser.
    presubmit:
        This target should be run before creating a pull request. It is a
        combination of test, lint, format, and cover targets. Make sure you
        commit everything before running this.
        (Note: if you see UNCOMMITED CHANGES error when you don't expect it,
        that means "make fmt" resulted in formatting fixes, and you ought to
        fixup relevant commits.)
endef
export HELP_TEXT

.PHONY: help
help:
	@echo "$$HELP_TEXT"
