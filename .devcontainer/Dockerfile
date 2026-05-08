# Using 1.21 because a wire used by Grafana sigsegvs on 1.22 - see (https://github.com/google/wire/issues/400)
FROM mcr.microsoft.com/devcontainers/go:1-1.21-bookworm

# The base image ships a stale Yarn apt source whose signing key has rotated,
# which breaks `apt update`. We don't install yarn via apt (corepack/base image
# provides it), so drop the list before refreshing apt.
RUN rm -f /etc/apt/sources.list.d/yarn.list /etc/apt/trusted.gpg.d/yarn*.gpg

RUN curl -s https://deb.nodesource.com/setup_16.x | bash
RUN apt update && apt install -y nodejs gcc g++ make
RUN npm install prettier -g -D --save-exact
RUN mkdir /grafana
RUN chown vscode:vscode /grafana

USER vscode

ENV NODE_ENV=dev
ENV NODE_OPTIONS="--max-old-space-size=4096"