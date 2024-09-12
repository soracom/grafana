# Using 1.21 because a wire used by Grafana sigsegvs on 1.22 - see (https://github.com/google/wire/issues/400)
FROM mcr.microsoft.com/devcontainers/go:1-1.21-bookworm

RUN curl -s https://deb.nodesource.com/setup_16.x | bash
RUN apt update && apt install -y nodejs gcc g++ make
RUN npm install prettier -g -D --save-exact
RUN mkdir /grafana
RUN chown vscode:vscode /grafana

USER vscode

ENV NODE_ENV=dev