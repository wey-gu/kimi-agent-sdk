.DEFAULT_GOAL := check

.PHONY: help
help: ## Show available make targets.
	@echo "Available make targets:"
	@awk 'BEGIN { FS = ":.*## " } /^[A-Za-z0-9_.-]+:.*## / { printf "  %-20s %s\n", $$1, $$2 }' $(MAKEFILE_LIST)

.PHONY: format
format: ## Auto-format sources with ruff.
	@echo "==> Formatting sources"
	@uv run ruff check --fix
	@uv run ruff format

.PHONY: check
check: ## Run linting and type checks.
	@echo "==> Checking (ruff + pyright + ty; ty is non-blocking)"
	@uv run ruff check
	@uv run ruff format --check
	@uv run pyright
	@uv run ty check || true

.PHONY: test
test: ## Run tests.
	@echo "==> Running tests"
	@uv run pytest tests -vv
