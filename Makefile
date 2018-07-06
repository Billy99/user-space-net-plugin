SUDO?=sudo
OS_ID        = $(shell grep '^ID=' /etc/os-release | cut -f2- -d= | sed -e 's/\"//g')
OS_VERSION_ID= $(shell grep '^VERSION_ID=' /etc/os-release | cut -f2- -d= | sed -e 's/\"//g')

ifeq ($(filter ubuntu debian,$(OS_ID)),$(OS_ID))
	PKG=deb
else ifeq ($(filter rhel centos fedora opensuse opensuse-leap opensuse-tumbleweed,$(OS_ID)),$(OS_ID))
	PKG=rpm
endif


#
# VPP Variables
#

VPPVERSION=18.04

# Building the cnivpp subfolder requires VPP to be installed, or at least a
# handful of files in the proper installed location. VPPINSTALLED indicates
# if required VPP files are installed. 
# For 'make clean', VPPLCLINSTALLED indicates if 'make install' installed
# the minimum set of files or if VPP is actually installed.
ifeq ($(shell test -e /usr/lib64/libvppapiclient.so && echo -n yes),yes)
	VPPINSTALLED=1
ifeq ($(shell test -e /usr/bin/vpp && echo -n yes),yes)
	VPPLCLINSTALLED=0
else
	VPPLCLINSTALLED=1
endif
else
	VPPINSTALLED=0
	VPPLCLINSTALLED=0
endif

# Default to build
default: build
all: build


help:
	@echo "Make Targets:"
	@echo " make                - Build UserSpace CNI."
	@echo " make clean          - Cleanup all build artifacts. Will remove VPP files installed from *make install*."
	@echo " make install        - If VPP is not installed, install the minimum set of files to build."
	@echo "                       CNI-VPP will fail because VPP is still not installed."
	@echo " make install-dep    - Install software dependencies, currently only needed for *make install*."
	@echo " make extras         - Build *vpp-app*, small binary to run in Docker container for testing."
	@echo " make test           - Build test code."
	@echo ""
	@echo "Other:"
	@echo " glide update --strip-vendor - Recalculate dependancies and update *vendor\* with proper packages."
	@echo ""
#	@echo "Makefile variables (debug):"
#	@echo    SUDO=$(SUDO) OS_ID=$(OS_ID) OS_VERSION_ID=$(OS_VERSION_ID) PKG=$(PKG) VPPVERSION=$(VPPVERSION) VPPINSTALLED=$(VPPINSTALLED) VPPLCLINSTALLED=$(VPPLCLINSTALLED)
#	@#echo ""

build:
ifeq ($(VPPINSTALLED),0)
	@echo VPP not installed. Run *make install* to install the minimum set of files to compile, or install VPP.
	@echo
endif
	@cd vendor/git.fd.io/govpp.git/cmd/binapi-generator && go build -v
	@./vendor/git.fd.io/govpp.git/cmd/binapi-generator/binapi-generator \
		--input-dir=/usr/share/vpp/api/ \
		--output-dir=vendor/git.fd.io/govpp.git/core/bin_api/
	@cd userspace && go build -v

test:
	@cd cnivpp/test/memifAddDel && go build -v
	@cd cnivpp/test/vhostUserAddDel && go build -v
	@cd cnivpp/test/ipAddDel && go build -v

install-dep:
ifeq ($(VPPINSTALLED),0)
ifeq ($(PKG),rpm)
	@$(SUDO) -E yum install -y wget cpio rpm
else ifeq ($(PKG),deb)
	@echo Install of VPP files on Debian systems currently not implemented. 
endif
endif

install:
ifeq ($(VPPINSTALLED),0)
ifeq ($(PKG),rpm)
	@echo VPP not installed, installing required files. Run *sudo make clean* to remove installed files.
	@mkdir -p tmpvpp/
	@cd tmpvpp && wget http://cbs.centos.org/kojifiles/packages/vpp/$(VPPVERSION)/1/x86_64/vpp-lib-$(VPPVERSION)-1.x86_64.rpm
	@cd tmpvpp && wget http://cbs.centos.org/kojifiles/packages/vpp/$(VPPVERSION)/1/x86_64/vpp-devel-$(VPPVERSION)-1.x86_64.rpm
	@cd tmpvpp && rpm2cpio ./vpp-devel-$(VPPVERSION)-1.x86_64.rpm | cpio -ivd \
		./usr/include/vpp-api/client/vppapiclient.h
	@cd tmpvpp && rpm2cpio ./vpp-lib-$(VPPVERSION)-1.x86_64.rpm | cpio -ivd \
		./usr/lib64/libvppapiclient.so.0.0.0
	@cd tmpvpp && rpm2cpio ./vpp-lib-$(VPPVERSION)-1.x86_64.rpm | cpio -ivd \
		./usr/share/vpp/api/interface.api.json \
		./usr/share/vpp/api/l2.api.json \
		./usr/share/vpp/api/memif.api.json \
		./usr/share/vpp/api/vhost_user.api.json \
		./usr/share/vpp/api/vpe.api.json
	@$(SUDO) -E mkdir -p /usr/include/vpp-api/client/
	@$(SUDO) -E cp tmpvpp/usr/include/vpp-api/client/vppapiclient.h /usr/include/vpp-api/client/.
	@$(SUDO) -E chown -R bin:bin /usr/include/vpp-api/
	@echo   Installed /usr/include/vpp-api/client/vppapiclient.h
	@$(SUDO) -E cp tmpvpp/usr/lib64/libvppapiclient.so.0.0.0 /usr/lib64/.
	@$(SUDO) -E ln -s /usr/lib64/libvppapiclient.so.0.0.0 /usr/lib64/libvppapiclient.so
	@$(SUDO) -E ln -s /usr/lib64/libvppapiclient.so.0.0.0 /usr/lib64/libvppapiclient.so.0
	@$(SUDO) -E chown -R bin:bin /usr/lib64/libvppapiclient.so*
	@echo   Installed /usr/lib64/libvppapiclient.so
	@$(SUDO) -E mkdir -p /usr/share/vpp/api/
	@$(SUDO) -E cp tmpvpp/usr/share/vpp/api/*.json /usr/share/vpp/api/.
	@$(SUDO) -E chown -R bin:bin /usr/share/vpp/
	@echo   Installed /usr/share/vpp/api/*.json
	@rm -rf tmpvpp
else ifeq ($(PKG),deb)
	@echo Install of VPP files on Debian systems currently not implemented. 
endif
endif


extras:
	@cd vendor/git.fd.io/govpp.git/cmd/binapi-generator && go build -v
	@./vendor/git.fd.io/govpp.git/cmd/binapi-generator/binapi-generator \
		--input-dir=/usr/share/vpp/api/ \
		--output-dir=vendor/git.fd.io/govpp.git/core/bin_api/
	@cd cnivpp/vpp-app && go build -v

clean:
	@rm -f cnivpp/vpp-app/vpp-app
	@rm -f cnivpp/test/memifAddDel/memifAddDel
	@rm -f cnivpp/test/vhostUserAddDel/vhostUserAddDel
	@rm -f cnivpp/test/ipAddDel/ipAddDel
	@rm -f vendor/git.fd.io/govpp.git/cmd/binapi-generator/binapi-generator 
	@rm -f userspace/userspace
ifeq ($(VPPLCLINSTALLED),1)
	@echo VPP was installed by *make install*, so cleaning up files.
	@$(SUDO) -E rm -rf /usr/include/vpp-api/
	@$(SUDO) -E rm /usr/lib64/libvppapiclient.so*
	@$(SUDO) -E rm -rf /usr/share/vpp/
endif

generate:

lint:

.PHONY: build test install extras clean generate

