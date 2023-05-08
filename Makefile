SOURCES := $(shell find ./ -type f -not -path '*/.*' -not -name '*_test.go' -name '*.go')

all:
	$(MAKE) wows-recruiting-bot

wows-recruiting-bot: ${SOURCES}
	go build

test:
	go test

clean:
	rm -f wows-recruiting-bot

.PHONY: clean test clean-all all
