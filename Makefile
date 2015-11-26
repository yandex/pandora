.PHONY: all test lint vet fmt travis coverage checkfmt prepare updep

NO_COLOR=\033[0m
OK_COLOR=\033[32;01m
ERROR_COLOR=\033[31;01m
WARN_COLOR=\033[33;01m
PKGSDIRS=$(shell find -L . -type f -name "*.go" -not -path "./Godeps/*")

all: test vet checkfmt

travis: test checkfmt vet coverage

prepare: fmt test vet checkfmt #m updep

test:
	@echo "$(OK_COLOR)Test packages$(NO_COLOR)"
	@go test -v ./...

coverage:
	@echo "$(OK_COLOR)Make coverage report$(NO_COLOR)"
	@./script/coverage.sh
	-goveralls -coverprofile=gover.coverprofile -service=travis-ci

lint:
	@echo "$(OK_COLOR)Run lint$(NO_COLOR)"
	test -z "$$(golint -min_confidence 0.1 ./... | grep -v Godeps/_workspace/src/ | tee /dev/stderr)"

vet:
	@echo "$(OK_COLOR)Run vet$(NO_COLOR)"
	@go vet ./...


checkfmt:
	@echo "$(OK_COLOR)Check formats$(NO_COLOR)"
	@./script/checkfmt.sh .

fmt:
	@echo "$(OK_COLOR)Formatting$(NO_COLOR)"
	@echo $(PKGSDIRS) | xargs -I '{p}' -n1 goimports -w {p}

tools:
	@echo "$(OK_COLOR)Install tools$(NO_COLOR)"
	go get github.com/tools/godep
	go get golang.org/x/tools/cmd/goimports
	go get github.com/golang/lint/golint
	go get github.com/stretchr/testify

updep:
	@echo "$(OK_COLOR)Update dependencies$(NO_COLOR)"
	GOOS=linux godep save ./...
	GOOS=linux godep update github.com/...
	GOOS=linux godep update gopkg.in/...
	GOOS=linux godep update golang.org/...

