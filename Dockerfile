# 不写 # syntax= 指令：那会让每次构建都去 Docker Hub 拉前端镜像（国内网络常失败），
# 而这里用到的多阶段、ARG、--platform=$BUILDPLATFORM 都是 BuildKit 内置前端就支持的。

# Web GUI 的 dist 已经在仓库里、由 go:embed 打进二进制，所以构建阶段不需要 node/pnpm。
#
# 构建阶段固定跑在 BUILDPLATFORM（构建机的原生架构）上，靠 Go 交叉编译产出
# TARGETARCH 的二进制——比在模拟器里跑整个构建快一个数量级。
FROM --platform=$BUILDPLATFORM golang:1.25-bookworm AS build

ARG TARGETARCH
ARG MIHOMO_VERSION=1.19.27
# TOFU 固定：校验和是首次下载时算出来钉住的，保证可重复构建，
# 不等于对上游做了供应链溯源。升级版本时两个都要更新。
# amd64 取 compatible 版（GOAMD64=v1），任何 x86_64 飞牛机型都能跑。
ARG MIHOMO_SHA256_AMD64=36850c946615f5c712946b62dbbbd06f6941d6d8a7543b315198bcb24ada3ea9
ARG MIHOMO_SHA256_ARM64=87db0c6660a9557a901b5750f997967e71d8c0af07ea1d1dd4d04c28da7f7e6f

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH="$TARGETARCH" go build -trimpath -ldflags="-s -w" -o /out/opensurge-control ./cmd/opensurge-control \
	&& CGO_ENABLED=0 GOOS=linux GOARCH="$TARGETARCH" go build -trimpath -ldflags="-s -w" -o /out/omg ./cmd/omg

RUN set -eu; \
	case "$TARGETARCH" in \
		amd64) archive="mihomo-linux-amd64-compatible-v${MIHOMO_VERSION}.gz"; sha="$MIHOMO_SHA256_AMD64" ;; \
		arm64) archive="mihomo-linux-arm64-v${MIHOMO_VERSION}.gz";            sha="$MIHOMO_SHA256_ARM64" ;; \
		*) echo "unsupported TARGETARCH: $TARGETARCH" >&2; exit 1 ;; \
	esac; \
	curl -fsSL --retry 3 -o /tmp/mihomo.gz \
		"https://github.com/MetaCubeX/mihomo/releases/download/v${MIHOMO_VERSION}/${archive}"; \
	echo "${sha}  /tmp/mihomo.gz" | sha256sum -c -; \
	gzip -dc /tmp/mihomo.gz > /out/mihomo; \
	chmod 0755 /out/mihomo

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
