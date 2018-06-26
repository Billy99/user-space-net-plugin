build:
	@cd vendor/git.fd.io/govpp.git/cmd/binapi-generator && go build -v
	@./vendor/git.fd.io/govpp.git/cmd/binapi-generator/binapi-generator --input-dir=/usr/share/vpp/api/ --output-dir=vendor/git.fd.io/govpp.git/core/bin_api/
	@cd vpp-app && go build -v
	@cd test/memifAddDel && go build -v
	@cd test/vhostUserAddDel && go build -v
	@cd test/ipAddDel && go build -v

test:

install:
	@cd cmd/binapi-generator && go install -v

extras:

clean:
	@rm -f vpp-app/vpp-app
	@rm -f test/memifAddDel/memifAddDel
	@rm -f test/vhostUserAddDel/vhostUserAddDel
	@rm -f test/ipAddDel/ipAddDel
	@rm -f vendor/git.fd.io/govpp.git/cmd/binapi-generator/binapi-generator 

generate:

lint:

.PHONY: build test install extras clean generate

