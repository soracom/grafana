# Using 1.21 because a wire used by Grafana sigsegvs on 1.22 - see (https://github.com/google/wire/issues/400)
FROM mcr.microsoft.com/devcontainers/go:1-1.23-bookworm

ARG BUILDARCH
ARG BUILDOS

# Override Go version (until microsoft Go devcontainer image is updated)
# Remove this when the image is updated
RUN rm -rf /usr/local/go && curl "https://dl.google.com/go/go1.23.6.${BUILDOS}-${BUILDARCH}.tar.gz" | tar -C /usr/local -xz

RUN curl -s https://deb.nodesource.com/setup_22.x | bash
RUN apt update && apt install -y nodejs gcc g++ make
RUN npm install prettier -g -D --save-exact
RUN mkdir /grafana
RUN chown vscode:vscode /grafana

USER vscode

ENV NODE_ENV=dev
ENV NODE_OPTIONS="--max-old-space-size=4096"
