# syntax=docker/dockerfile:1

# Web GUI 的 dist 已经在仓库里、由 go:embed 打进二进制，所以构建阶段不需要 node/pnpm。
FROM golang:1.25-bookworm AS build

ARG MIHOMO_VERSION=1.19.27
# TOFU 固定：这个校验和是第一次下载时算出来钉住的，保证可重复构建，
# 不等于对上游做了供应链溯源。升级版本时一并更新它。
ARG MIHOMO_SHA256=36850c946615f5c712946b62dbbbd06f6941d6d8a7543b315198bcb24ada3ea9

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o /out/opensurge-control ./cmd/opensurge-control \
	&& CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o /out/omg ./cmd/omg

# compatible 版是 GOAMD64=v1，任何 x86_64 飞牛机型都能跑。
RUN curl -fsSL --retry 3 -o /tmp/mihomo.gz \
		"https://github.com/MetaCubeX/mihomo/releases/download/v${MIHOMO_VERSION}/mihomo-linux-amd64-compatible-v${MIHOMO_VERSION}.gz" \
	&& echo "${MIHOMO_SHA256}  /tmp/mihomo.gz" | sha256sum -c - \
	&& gzip -dc /tmp/mihomo.gz > /out/mihomo \
	&& chmod 0755 /out/mihomo

FROM debian:12-slim

# nftables  → internal/pf 的 nft 命令
# iproute2  → internal/macosnetwork 的 ip neigh / ip route / ip addr
# iputils-ping → PingRouter
# procps    → internal/sysctl 调的 sysctl 命令
# dnsmasq-base → 只要二进制，不要 Debian 那套自启服务（旁路由模式下只做 DNS 转发）
RUN apt-get update \
	&& apt-get install -y --no-install-recommends \
		nftables \
		iproute2 \
		iputils-ping \
		procps \
		dnsmasq-base \
		ca-certificates \
	&& rm -rf /var/lib/apt/lists/*

COPY --from=build /out/opensurge-control /out/omg /out/mihomo /usr/local/bin/
COPY docker/entrypoint.sh /usr/local/bin/entrypoint.sh
COPY examples/config.fnos.example.yaml /usr/share/opensurge/config.fnos.example.yaml

EXPOSE 61767

ENTRYPOINT ["/usr/local/bin/entrypoint.sh"]
