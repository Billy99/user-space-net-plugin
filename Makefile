build:
	@cd vendor/git.fd.io/govpp.git/cmd/binapi-generator && go build -v
	@./vendor/git.fd.io/govpp.git/cmd/binapi-generator/binapi-generator --input-dir=/usr/share/vpp/api/ --output-dir=vendor/git.fd.io/govpp.git/core/bin_api/
	@cd userspace && go build -v

test:
	@cd cnivpp/test/memifAddDel && go build -v
	@cd cnivpp/test/vhostUserAddDel && go build -v
	@cd cnivpp/test/ipAddDel && go build -v

install:
	@cd cmd/binapi-generator && go install -v

extras:
	@cd cnivpp/vpp-app && go build -v

clean:
	@rm -f cnivpp/vpp-app/vpp-app
	@rm -f cnivpp/test/memifAddDel/memifAddDel
	@rm -f cnivpp/test/vhostUserAddDel/vhostUserAddDel
	@rm -f cnivpp/test/ipAddDel/ipAddDel
	@rm -f vendor/git.fd.io/govpp.git/cmd/binapi-generator/binapi-generator 
	@rm -f userspace/userspace

generate:

lint:

.PHONY: build test install extras clean generate

