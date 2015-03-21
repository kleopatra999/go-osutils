.PHONY: \
	all \
	precommit \
	check_for_codeship \
	deps \
	updatedeps \
	testdeps \
	updatetestdeps \
	build \
	install \
	cov \
	test \
	codeshipsteps \
	doc \
	clean

all: test install

precommit: doc

check_for_codeship:
	@ if ! which codeship > /dev/null; then \
		echo "error: codeship not installed" >&2; \
	  fi

deps:
	go get -d -v ./...

updatedeps:
	go get -d -v -u -f ./...

testdeps: deps
	go get -d -v -t ./...

updatetestdeps: updatedeps
	go get -d -v -t -u -f ./...

build: deps
	go build ./...

install: deps
	go install ./...

cov: testdeps
	go get -v github.com/axw/gocov/gocov
	gocov test | gocov report

test: testdeps
	go test -test.v ./...

codeshipsteps: check_for_codeship 
	codeship steps

doc:
	go get -v github.com/robertkrimen/godocdown/godocdown
	cp .readme.header README.md
	godocdown | tail -n +7 >> README.md

clean:
	go clean -i ./...
