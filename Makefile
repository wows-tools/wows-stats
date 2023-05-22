SOURCES := $(shell find ./ -type f -not -path '*/.*' -not -name '*_test.go' -name '*.go')

all:
	$(MAKE) wows-stats

wows-stats: ${SOURCES}
	go build

test:
	go test

clean:
	rm -f wows-stats

.PHONY: clean test clean-all all
