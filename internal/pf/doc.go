// Package pf 管理网关的防火墙/NAT 规则。
//
// 包名和 "anchor" 措辞保留了上游的 macOS pf 出身：本项目要长期 rebase 上游，
// 改名会让每次同步在调用点冲突。实际实现按平台拆分——darwin 走 pfctl anchor，
// linux 走 nftables（table inet opensurge）。
package pf
