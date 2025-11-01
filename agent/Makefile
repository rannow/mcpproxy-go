# MCP Agent Testing Makefile
# Convenient shortcuts for running tests

.PHONY: help test test-unit test-integration test-e2e test-cov test-cov-html \
        test-fast test-slow test-all lint format clean install-dev

help:  ## Show this help message
	@echo "MCP Agent Testing Commands:"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'

install-dev:  ## Install development dependencies
	pip install -r requirements.txt
	pip install -e .

test:  ## Run all tests
	pytest

test-unit:  ## Run unit tests only
	pytest -m unit -v

test-integration:  ## Run integration tests only
	pytest -m integration -v

test-e2e:  ## Run end-to-end tests only
	pytest -m e2e -v

test-diagnostic:  ## Run diagnostic tool tests
	pytest -m diagnostic -v

test-config:  ## Run configuration tests
	pytest -m config -v

test-graph:  ## Run LangGraph workflow tests
	pytest -m graph -v

test-fast:  ## Run fast tests only (exclude slow tests)
	pytest -m "not slow" -v

test-slow:  ## Run slow tests only
	pytest -m slow -v

test-no-api:  ## Run tests that don't require API
	pytest -m "not requires_api" -v

test-cov:  ## Run tests with coverage report
	pytest --cov=mcp_agent --cov-report=term-missing

test-cov-html:  ## Run tests with HTML coverage report
	pytest --cov=mcp_agent --cov-report=html
	@echo "Coverage report generated at htmlcov/index.html"

test-cov-xml:  ## Run tests with XML coverage report (for CI)
	pytest --cov=mcp_agent --cov-report=xml

test-parallel:  ## Run tests in parallel (requires pytest-xdist)
	pytest -n auto

test-verbose:  ## Run tests with verbose output
	pytest -vv

test-debug:  ## Run tests with maximum verbosity and debug info
	pytest -vv --log-cli-level=DEBUG

test-failed:  ## Re-run only failed tests from last run
	pytest --lf -v

test-watch:  ## Watch for changes and re-run tests (requires pytest-watch)
	ptw -- -v

test-specific:  ## Run specific test file (usage: make test-specific FILE=tests/unit/test_diagnostic_tools.py)
	pytest $(FILE) -v

lint:  ## Run code linters
	ruff check mcp_agent tests
	mypy mcp_agent

format:  ## Format code with black and ruff
	black mcp_agent tests
	ruff check --fix mcp_agent tests

clean:  ## Clean up generated files
	rm -rf .pytest_cache
	rm -rf htmlcov
	rm -f coverage.xml
	rm -f .coverage
	rm -rf .mypy_cache
	rm -rf .ruff_cache
	find . -type d -name __pycache__ -exec rm -rf {} +
	find . -type f -name "*.pyc" -delete

test-all: clean test-cov lint  ## Run all tests with coverage and linting

# CI/CD targets
ci-test:  ## Run tests for CI (with coverage and XML report)
	pytest --cov=mcp_agent --cov-report=xml --cov-report=term -v

ci-test-fast:  ## Run fast CI tests (exclude slow and API tests)
	pytest -m "not slow and not requires_api" --cov=mcp_agent --cov-report=xml -v

# Development helpers
dev-setup:  ## Setup development environment
	pip install -r requirements.txt
	pip install -e .
	pip install pytest-xdist pytest-watch

test-one:  ## Run one test (usage: make test-one TEST=test_name)
	pytest -k $(TEST) -v

# Coverage thresholds
test-cov-check:  ## Run tests and check coverage threshold (80%)
	pytest --cov=mcp_agent --cov-report=term-missing --cov-fail-under=80

# Test data generation (for future use)
generate-fixtures:  ## Generate test fixtures (placeholder)
	@echo "Fixture generation not yet implemented"

# Documentation
test-docs:  ## Generate test documentation
	@echo "Generating test documentation..."
	@find tests -name "*.py" -type f | xargs grep -h "^class Test" | sort | uniq
	@echo ""
	@echo "See tests/README.md for detailed documentation"
