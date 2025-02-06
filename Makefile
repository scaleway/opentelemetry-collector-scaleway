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
