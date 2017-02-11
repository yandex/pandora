.PHONY: all test lint vet fmt travis coverage checkfmt prepare updep

NO_COLOR=\033[0m
OK_COLOR=\033[32;01m
ERROR_COLOR=\033[31;01m
WARN_COLOR=\033[33;01m
PKGSDIRS=`find -L . -type f -name "*.go" -not -path "./vendor/*"`

all: test vet checkfmt

travis: test checkfmt coverage

prepare: fmt test vet

test:
	@echo "$(OK_COLOR)Test packages$(NO_COLOR)"
	@go test -v `glide novendor`

coverage:
	@echo "$(OK_COLOR)Make coverage report$(NO_COLOR)"
	@./script/coverage.sh
	-goveralls -coverprofile=gover.coverprofile -service=travis-ci

lint:
	@echo "$(OK_COLOR)Run lint$(NO_COLOR)"
	@for dir in `glide novendor` ; do \
	    golint -set_exit_status -min_confidence 0.1 $$dir || FAIL="true" ;\
	done
	@if [[ FAIL ]] ; then  \
        exit 1 ; \
    fi

vet:
	@echo "$(OK_COLOR)Run vet$(NO_COLOR)"
	@go vet `glide novendor`


checkfmt:
	@echo "$(OK_COLOR)Check formats$(NO_COLOR)"
	@./script/checkfmt.sh .

fmt:
	@echo "$(OK_COLOR)Formatting$(NO_COLOR)"
	@echo $(PKGSDIRS) | xargs -I '{p}' -n1 goimports -w {p}

tools:
	@echo "$(OK_COLOR)Install tools$(NO_COLOR)"
	go get golang.org/x/tools/cmd/goimports
	go get github.com/golang/lint/golint
	go get golang.org/x/tools/cmd/cover
	go get github.com/modocache/gover
	go get github.com/mattn/goveralls

updep:
	@echo "$(OK_COLOR)Update dependencies$(NO_COLOR)"
	@glide up

PARENT=$$GOPATH/src/github.com/yandex
DIR=$(PARENT)/pandora

