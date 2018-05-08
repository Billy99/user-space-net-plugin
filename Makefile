build:
	@cd vendor/git.fd.io/govpp.git/cmd/binapi-generator && go build -v
	@./vendor/git.fd.io/govpp.git/cmd/binapi-generator/binapi-generator --input-dir=/usr/share/vpp/api/ --output-dir=vendor/git.fd.io/govpp.git/core/bin_api/
	@cd test/memifAddDel && go build -v
	@cd test/vhostUserAddDel && go build -v

test:
	@cd cmd/binapi-generator && go test -cover .
	@cd api && go test -cover ./...
	@cd core && go test -cover .

install:
	@cd cmd/binapi-generator && go install -v

extras:
	@cd extras/libmemif/examples/raw-data && go build -v
	@cd extras/libmemif/examples/icmp-responder && go build -v

clean:
	@rm -f test/memifAddDel/memifAddDel
	@rm -f test/vhostUserAddDel/vhostUserAddDel
	@rm -f vendor/git.fd.io/govpp.git/cmd/binapi-generator/binapi-generator 

generate:
	@cd core && go generate ./...
	@cd examples && go generate ./...

lint:
	@golint ./... | grep -v vendor | grep -v bin_api || true

.PHONY: build test install extras clean generate

