
FROM vm 

# For building a dev image to run locally for testing/development.

#################################
#################################
RUN dnf -y install curl
# prepackaged Go
ENV GO_VER=1.24.2
ENV PATH="/var/home/root/go/bin:/usr/local/go/bin:$PATH"
RUN echo curl -fsSLO https://go.dev/dl/go$GO_VER.linux-$(uname -m | sed 's/x86_64/amd64/' | sed 's/aarch64/arm64/').tar.gz
RUN curl -fsSLO https://go.dev/dl/go$GO_VER.linux-$(uname -m | sed 's/x86_64/amd64/' | sed 's/aarch64/arm64/').tar.gz
RUN cat go$GO_VER.linux-*.tar.gz  | tar -C /usr/local -xz
RUN rm go$GO_VER.linux-*.tar.gz
RUN go version
RUN go install github.com/mitranim/gow@latest
RUN go install github.com/faultymajority/git-semver/v7@main

# prepackaged Rust
ENV RUST_VER=1.85
ENV PATH="/var/home/root/.cargo/bin:$PATH"
RUN curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh -s -- -y --default-toolchain ${RUST_VER}
RUN curl -L --proto '=https' --tlsv1.2 -sSf https://raw.githubusercontent.com/cargo-bins/cargo-binstall/main/install-from-binstall-release.sh | bash
RUN cargo version
# various deps needs to build our rust binaries
RUN dnf config-manager --set-enabled crb
RUN dnf install glibc-devel gcc cmake openssl-devel glibc-static -y
# xh is a better curl
RUN cargo install xh
# just
RUN  curl --proto '=https' --tlsv1.2 -sSf https://just.systems/install.sh | bash -s -- --to /usr/bin
#################################
#################################

# Fix docker-in-docker on macos
RUN systemctl enable /usr/units/configure-docker.service
