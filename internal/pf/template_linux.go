//go:build linux

package pf

import (
	"bytes"
	"text/template"

	"open-mihomo-gateway/internal/config"
)

// tableName 是 nftables 表名。它不走 cfg.PF.AnchorName —— 那是 macOS 的 pf anchor
// 路径（默认 "com.apple/open_mihomo_gateway"），不是合法的 nft 标识符。
const tableName = "opensurge"

// 开头的 `table` + `delete table` 是 nft 的幂等惯用法：先确保表存在（否则 delete 报错），
// 再删掉重建，这样 `nft -f` 重复执行不会把规则叠加两遍。
//
// 转发本身由 mihomo 的 TUN 栈处理，这里只做出口 NAT，规则语义对齐 macOS 的 pf anchor：
// same-LAN 模式下排除本网段（LAN 内互访不该被 NAT）。
const rulesetTemplate = `table inet {{ .TableName }}
delete table inet {{ .TableName }}

table inet {{ .TableName }} {
	chain postrouting {
		type nat hook postrouting priority srcnat; policy accept;
		oifname "{{ .UpstreamInterface }}" ip saddr {{ .LanCIDR }}{{ if .SameLAN }} ip daddr != {{ .LanCIDR }}{{ end }} masquerade
	}

	chain forward {
		type filter hook forward priority filter; policy accept;
	}
}
`

type templateData struct {
	TableName         string
	UpstreamInterface string
	LanCIDR           string
	SameLAN           bool
}

// RenderAnchor 渲染 nftables 规则集。函数名沿用上游（macOS pf anchor 的叫法），
// 保持调用方不变；见 doc.go。
func RenderAnchor(cfg config.Config) (string, error) {
	lanCIDR, err := cfg.LANPrefix24()
	if err != nil {
		return "", err
	}
	data := templateData{
		TableName:         tableName,
		UpstreamInterface: cfg.Gateway.UpstreamInterface,
		LanCIDR:           lanCIDR,
		SameLAN:           cfg.Gateway.SameLAN(),
	}

	tmpl, err := template.New("nft-ruleset").Parse(rulesetTemplate)
	if err != nil {
		return "", err
	}
	var out bytes.Buffer
	if err := tmpl.Execute(&out, data); err != nil {
		return "", err
	}
	return out.String(), nil
}
