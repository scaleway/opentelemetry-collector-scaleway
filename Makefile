include ./Makefile.Common

ALL_MODULES := $(shell find . -type f -name "go.mod" -exec dirname {} \; | sort | grep -E '^./' )

all-modules:
	@echo $(ALL_MODULES) | tr ' ' '\n' | sort

# Append root module to all modules
GOMODULES = $(ALL_MODULES)

# Define a delegation target for each module
.PHONY: $(GOMODULES)
$(GOMODULES):
	@echo "Running target '$(TARGET)' in module '$@'"
	$(MAKE) -C $@ $(TARGET)

.PHONY: for-all-target
for-all-target: $(GOMODULES)

.PHONY: gotest
gotest:
	@$(MAKE) for-all-target TARGET="test"

.PHONY: gotest-with-cover
gotest-with-cover:
	@$(MAKE) for-all-target TARGET="test-with-cover"
	$(GOCMD) tool covdata textfmt -i=./coverage/unit -o ./coverage.txt

.PHONY: golint
golint:
	@$(MAKE) for-all-target TARGET="lint"

.PHONY: gotidy
gotidy:
	@$(MAKE) for-all-target TARGET="tidy"

.PHONY: multimod-verify
multimod-verify:
	$(MULTIMOD) verify

.PHONY: multimod-prerelease
multimod-prerelease:
	$(MULTIMOD) prerelease --module-set-names stable
	$(MAKE) gotidy
