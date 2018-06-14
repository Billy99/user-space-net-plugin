FROM centos
MAINTAINER Billy McFall <bmcfall@redhat.com>

# Add any additional repos to image
COPY *.repo /etc/yum.repos.d/

# Install VPP
RUN yum install -y vpp-plugins vpp-devel vpp-api-python vpp-api-lua vpp-api-java; yum clean all

# Overwrite VPP systemfiles
COPY startup.conf /etc/vpp/startup.conf
COPY 80-vpp.conf /etc/sysctl.d/80-vpp.conf

# Install UserSpace CNI
COPY vpp-app /usr/sbin/vpp-app

# Install script to start both VPP and vpp-agent
COPY vppcni.sh vppcni.sh

# Setup VPP UserGroup and User
#RUN useradd --no-log-init -r -g vpp vpp
#USER vpp

CMD bash -C './vppcni.sh';'bash'
#CMD [ "./vppcni.sh" ]
