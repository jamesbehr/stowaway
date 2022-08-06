.PHONY: test doctest clean

stowaway: $(shell find -name '*.go')
	env CGO_ENABLED=0 go build

test:
	go test ./...

doctest: stowaway
	docker run --rm -it \
		-v $(shell pwd)/stowaway:/usr/bin/stowaway \
		-v $(shell pwd):/home/me/stowaway \
		jamesbehr/docshtest:latest \
		--run-highlighted-code-fences console \
		stowaway/README.md

clean:
	rm stowaway
