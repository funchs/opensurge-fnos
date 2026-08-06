// Package macosnetwork 提供控制面所需的主机网络查询与配置能力。
//
// 包名保留了上游的 macOS 出身：本项目要长期 rebase 上游，改名会让每次同步在所有
// import 点冲突。实际实现按平台拆分——darwin 走 networksetup / arp / route，
// linux 走 ip 命令；在 fnOS 上所有「修改宿主机网络」的入口返回 ErrManagedByFnOS。
package macosnetwork
