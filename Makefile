.PHONY: test build doctor status policy-control-test
.PHONY: web-install web-build web-test control-build control-run docker-build docker-push docker-push-multiarch
.PHONY: fpk fpk-x86 fpk-arm fpk-clean
.PHONY: lab-install lab-uninstall-root lab-check lab-up lab-status lab-test
.PHONY: lab-test-tun lab-test-tun-imported-profile lab-test-tun-imported-egress lab-test-tun-local-routing lab-test-tun-device-policy lab-down lab-destroy
.PHONY: real-device-start-off real-device-start-tun real-device-start-tun-proxy
.PHONY: real-device-stop real-device-status real-device-client-check
.PHONY: same-lan-start-tun same-lan-start-tun-proxy same-lan-start-tun-imported-egress
.PHONY: same-lan-stop same-lan-status same-lan-adb-check same-lan-adb-check-imported-egress
.PHONY: same-lan-start-wifi-dhcp-imported-egress same-lan-adb-check-wifi-dhcp-imported-egress
.PHONY: same-lan-stop-wifi-dhcp same-lan-status-wifi-dhcp
.PHONY: same-wifi-dhcp-start-imported-egress same-wifi-dhcp-adb-check-imported-egress
.PHONY: same-wifi-dhcp-start-device-policy same-wifi-dhcp-adb-check-device-policy same-wifi-dhcp-verify-device-policy-recovery
.PHONY: same-wifi-dhcp-stop same-wifi-dhcp-status

test:
	go test ./...

build:
	go build -o bin/omg ./cmd/omg

web-install:
	cd web && pnpm install

web-build:
	cd web && pnpm run build

web-test:
	cd web && pnpm run test

control-build: web-build
	go build -o bin/opensurge-control ./cmd/opensurge-control
	go build -o bin/opensurge-helper ./cmd/opensurge-helper
	go build -o bin/opensurge-install-config ./cmd/opensurge-install-config

control-run: control-build
	./bin/opensurge-control --config examples/config.example.yaml

REPO ?= ghcr.io/funchs/opensurge-fnos
VERSION ?= latest
IMAGE ?= $(REPO):$(VERSION)

docker-build:
	docker build --platform linux/amd64 -t $(IMAGE) .

docker-push: docker-build
	docker push $(IMAGE)

# 多架构：amd64（x86 飞牛机型）+ arm64（ARM 飞牛，以及 Apple Silicon 虚拟机验证）。
#
# 不用 `buildx --platform amd64,arm64` 一步到位，是因为 buildx 的容器化 builder
# 有自己的 buildkitd 配置，继承不到 daemon 的 registry mirror，国内网络下拉
# golang / debian 基础镜像会 EOF。默认 builder 走得到 mirror，所以分开构建再合并。
# 调试轮次专用：反复覆盖 :dev，不占用版本号，也不动 :latest。
# 只推目标机器要的那一个架构（默认 arm64，x86 机型传 ARCH=amd64）。
# 验证跑通了再 make docker-push-multiarch VERSION=vX.Y.Z 打正式版。
ARCH ?= arm64
docker-push-dev:
	docker build --platform linux/$(ARCH) -t $(REPO):dev .
	docker push $(REPO):dev

docker-push-multiarch:
	docker build --platform linux/amd64 -t $(REPO):$(VERSION)-amd64 .
	docker push $(REPO):$(VERSION)-amd64
	docker build --platform linux/arm64 -t $(REPO):$(VERSION)-arm64 .
	docker push $(REPO):$(VERSION)-arm64
	docker buildx imagetools create -t $(REPO):$(VERSION) \
		$(REPO):$(VERSION)-amd64 $(REPO):$(VERSION)-arm64
	@# latest 跟着主线走：发版时顺手把它指到同一组 manifest。
	[ "$(VERSION)" = latest ] || docker buildx imagetools create -t $(REPO):latest \
		$(REPO):$(VERSION)-amd64 $(REPO):$(VERSION)-arm64
	docker buildx imagetools inspect $(REPO):$(VERSION)

# 飞牛应用中心安装包。版本号取 packaging/fnos/fnos/manifest 的 version，
# 不从这里的 VERSION 传——manifest 是单一事实来源，镜像 tag 是 v + 同一数字。
# 详见 packaging/fnos/README.md。
FPK_DIR := packaging/fnos

fpk:
	cd $(FPK_DIR) && ./scripts/build.sh && ./build-fpk.sh all

fpk-x86:
	cd $(FPK_DIR) && ./scripts/build.sh && ./build-fpk.sh x86

fpk-arm:
	cd $(FPK_DIR) && ./scripts/build.sh && ./build-fpk.sh arm

fpk-clean:
	rm -f $(FPK_DIR)/app.tgz $(FPK_DIR)/*.fpk

doctor:
	go run ./cmd/omg doctor --config examples/config.example.yaml

status:
	go run ./cmd/omg status --config examples/config.example.yaml

policy-control-test:
	./tests/integration/policy-control.sh

lab-install:
	./tests/lab/install-host-deps.sh

lab-uninstall-root:
	./tests/lab/install-host-deps.sh --uninstall-root

lab-check:
	./tests/lab/lab.sh check

lab-up:
	./tests/lab/lab.sh up

lab-status:
	./tests/lab/lab.sh status

lab-test:
	./tests/lab/lab.sh test

lab-test-tun:
	./tests/lab/lab.sh test-tun

lab-test-tun-imported-profile:
	OMG_LAB_MIHOMO_PROFILE=tests/lab/mihomo-profile.imported-tun.yaml ./tests/lab/lab.sh test-tun

lab-test-tun-imported-egress:
	OMG_LAB_MIHOMO_PROFILE=tests/lab/mihomo-profile.imported-tun-egress.yaml ./tests/lab/lab.sh test-tun

lab-test-tun-local-routing:
	OMG_LAB_MIHOMO_PROFILE=tests/lab/mihomo-profile.imported-tun-egress.yaml OMG_LAB_LOCAL_ROUTING_TEST=true ./tests/lab/lab.sh test-tun

lab-test-tun-device-policy:
	./tests/lab/lab.sh test-tun-device-policy

lab-down:
	./tests/lab/lab.sh down

lab-destroy:
	./tests/lab/lab.sh destroy

real-device-start-off:
	./tests/real-device/smoke.sh start-off

real-device-start-tun:
	./tests/real-device/smoke.sh start-tun

real-device-start-tun-proxy:
	OMG_REAL_DEVICE_UPSTREAM_PROXY_ENABLED=true ./tests/real-device/smoke.sh start-tun

real-device-stop:
	./tests/real-device/smoke.sh stop

real-device-status:
	./tests/real-device/smoke.sh status

real-device-client-check:
	./tests/real-device/smoke.sh client-check

same-lan-start-tun:
	./tests/same-lan/smoke.sh start-tun

same-lan-start-tun-proxy:
	OMG_SAME_LAN_UPSTREAM_PROXY_ENABLED=true ./tests/same-lan/smoke.sh start-tun

same-lan-start-tun-imported-egress:
	OMG_SAME_LAN_IMPORTED_EGRESS=true ./tests/same-lan/smoke.sh start-tun-imported-egress

same-lan-stop:
	./tests/same-lan/smoke.sh stop

same-lan-status:
	./tests/same-lan/smoke.sh status

same-lan-adb-check:
	./tests/same-lan/smoke.sh adb-check

same-lan-adb-check-imported-egress:
	OMG_SAME_LAN_IMPORTED_EGRESS=true ./tests/same-lan/smoke.sh adb-check-imported-egress

same-lan-start-wifi-dhcp-imported-egress:
	OMG_SAME_WIFI_DHCP_ENABLED=true OMG_SAME_LAN_IMPORTED_EGRESS=true ./tests/same-lan/smoke.sh start-wifi-dhcp-imported-egress

same-lan-adb-check-wifi-dhcp-imported-egress:
	OMG_SAME_WIFI_DHCP_ENABLED=true OMG_SAME_LAN_IMPORTED_EGRESS=true ./tests/same-lan/smoke.sh adb-check-wifi-dhcp-imported-egress

same-lan-stop-wifi-dhcp:
	OMG_SAME_WIFI_DHCP_ENABLED=true OMG_SAME_LAN_IMPORTED_EGRESS=true ./tests/same-lan/smoke.sh stop

same-lan-status-wifi-dhcp:
	OMG_SAME_WIFI_DHCP_ENABLED=true ./tests/same-lan/smoke.sh status

same-wifi-dhcp-start-imported-egress: same-lan-start-wifi-dhcp-imported-egress

same-wifi-dhcp-adb-check-imported-egress: same-lan-adb-check-wifi-dhcp-imported-egress

same-wifi-dhcp-start-device-policy:
	OMG_SAME_WIFI_DHCP_ENABLED=true OMG_SAME_LAN_IMPORTED_EGRESS=true OMG_SAME_WIFI_DEVICE_POLICY_ENABLED=true ./tests/same-lan/smoke.sh start-wifi-dhcp-device-policy

same-wifi-dhcp-adb-check-device-policy:
	OMG_SAME_WIFI_DHCP_ENABLED=true OMG_SAME_LAN_IMPORTED_EGRESS=true OMG_SAME_WIFI_DEVICE_POLICY_ENABLED=true ./tests/same-lan/smoke.sh adb-check-wifi-dhcp-device-policy

same-wifi-dhcp-verify-device-policy-recovery:
	OMG_SAME_WIFI_DHCP_ENABLED=true OMG_SAME_WIFI_DEVICE_POLICY_ENABLED=true ./tests/same-lan/smoke.sh verify-wifi-dhcp-device-policy-recovery

same-wifi-dhcp-stop: same-lan-stop-wifi-dhcp

same-wifi-dhcp-status: same-lan-status-wifi-dhcp
