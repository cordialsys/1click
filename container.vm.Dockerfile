FROM base

RUN dnf -y update
RUN dnf install -y jq curl git wget

# enable cloud-init
# While not functionally needed, it's required to be able to provision SSH keys with aws/gcp, which the marketplace agent (on aws) requires.
RUN dnf install -y cloud-init cloud-utils-growpart gdisk parted
RUN systemctl enable cloud-init-local.service cloud-init.service cloud-config.service cloud-final.service

# netbird
RUN dnf install -y dnf-plugins-core
# RUN dnf install -y 'dnf5-command(config-manager)'
RUN tee /etc/yum.repos.d/netbird.repo <<EOF
[netbird]
name=netbird
baseurl=https://pkgs.netbird.io/yum/
enabled=1
gpgcheck=0
gpgkey=https://pkgs.netbird.io/yum/repodata/repomd.xml.key
repo_gpgcheck=1
EOF
RUN dnf config-manager --add-repo /etc/yum.repos.d/netbird.repo
RUN dnf install -y netbird

# docker 
RUN dnf config-manager --add-repo https://download.docker.com/linux/centos/docker-ce.repo
RUN dnf -y install docker-ce docker-ce-cli containerd.io
RUN systemctl enable docker

# otel collector
RUN wget https://github.com/open-telemetry/opentelemetry-collector-releases/releases/download/v0.118.0/otelcol-contrib_0.118.0_linux_$(uname -m | sed 's/x86_64/amd64/' | sed 's/aarch64/arm64/').rpm
RUN rpm -ivh otelcol-contrib_0.118.0_linux_*.rpm && rm *.rpm
RUN usermod -a -G systemd-journal otelcol-contrib
# remove the otel systemd unit as we want to replace it
RUN systemctl disable otelcol-contrib
RUN rm /usr/lib/systemd/system/otelcol-contrib.service
# Permit otelcol-contrib to modify it's own env file to inject secret
RUN chown otelcol-contrib /etc/otelcol-contrib/otelcol-contrib.conf
RUN mkdir -p /var/home/otelcol-contrib
RUN chown -R otelcol-contrib /var/home/otelcol-contrib

# packages we like (note: no neovim as it pulls in like 500MB)
run dnf -y install epel-release
RUN dnf -y install zsh git htop tmux fd-find bat ripgrep wget jq util-linux-user
RUN dnf clean all

# use /var/bin for dynamically downloaded prod binaries (e.g. /usr/bin is immutable)
ENV PATH=${PATH}:/var/bin
RUN mkdir -p /var/bin && \
    ln -s /var/bin/cord /usr/bin/cord && \
    ln -s /var/bin/treasury /usr/bin/treasury && \
    ln -s /var/bin/signer /usr/bin/signer

COPY infra/scripts /scripts
COPY infra/usr /usr
COPY infra/etc /etc

# add panel release
COPY --from=build-panel /go/bin/panel /usr/bin/panel
COPY --from=build-panel /www /www

ENV PATH=${PATH}:/scripts

# Change /root home to be /var/home/root
RUN sed -i /etc/passwd -e "s|:/root:|:/var/home/root:|"
RUN su root /scripts/add_user.sh root
RUN /scripts/add_user.sh cordial
# Permit 'cordial' to restart systemd services, as this is how to pull new images
RUN echo "cordial ALL=(ALL) NOPASSWD: /bin/systemctl restart *" > /etc/sudoers.d/cordial
# Permit 'cordial' to run bootc commands
RUN echo "cordial ALL=(ALL) NOPASSWD: /usr/bin/bootc *" >> /etc/sudoers.d/cordial

# Start systemd services
RUN systemctl enable /usr/units/panel.service
RUN systemctl enable /usr/units/treasury-firewall.service
# We want some services to be started manually by panel, so we just link them.
RUN ln -s /usr/units/treasury.service /etc/systemd/system/treasury.service
RUN ln -s /usr/units/start-treasury.service /etc/systemd/system/start-treasury.service
RUN ln -s /usr/units/blueprint.service /etc/systemd/system/blueprint.service

# # disable SELinux (for Tailscale SSH + other systemd services)
# RUN sed -i 's/SELINUX=enforcing/SELINUX=disabled/' /etc/selinux/config
# bootc-image-builder will assume selinux is enabled if this file simply exists
RUN rm /etc/selinux/config
RUN rm -r /ostree/repo
