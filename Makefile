.DEFAULT_GOAL := help

.PHONY: lint
lint: ## lint codes
	golangci-lint run

.PHONY: fmt
fmt: ## format codes
	goimports -l -w .

.PHONY: test
test: ## run tests
	go test -v -cover ./...

.PHONY: yaegi/test
yaegi/test: ## run yaegi tests
	yaegi test .

.PHONY: vendor
vendor:
	go mod vendor

.PHONY: clean
clean:
	rm -rf ./vendor

.PHONY: __
__:
	@echo "\033[33m"
	@echo "kzmake/traefik-plugin-forward-request"
	@echo "\033[0m"

.PHONY: help
help: __ ## show help
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@cat $(MAKEFILE_LIST) \
	| grep -e "^[a-zA-Z_/\-]*: *.*## *" \
	| awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-24s\033[0m %s\n", $$1, $$2}'
