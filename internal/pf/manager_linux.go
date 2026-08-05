//go:build linux

package pf

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"open-mihomo-gateway/internal/config"
	"open-mihomo-gateway/internal/process"
	"open-mihomo-gateway/internal/runtime"
)

type Manager struct {
	cfg   config.Config
	paths runtime.Paths
}

func New(cfg config.Config, paths runtime.Paths) Manager {
	return Manager{cfg: cfg, paths: paths}
}

func (m Manager) Check() error {
	if _, err := exec.LookPath("nft"); err != nil {
		return fmt.Errorf("nft not found in PATH")
	}
	return nil
}

// WriteAnchor 复用 paths.PFAnchor 作为规则文件路径。名字是 macOS 遗留，
// 内容在 Linux 上是 nft 规则集 —— 不新增 Paths 字段是为了压小上游 rebase 的冲突面。
func (m Manager) WriteAnchor() error {
	rendered, err := RenderAnchor(m.cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(m.paths.PFAnchor, []byte(rendered), 0o640)
}

// Enabled 在 Linux 上恒为 true：nftables 没有 pf 那样的全局开关。
func (m Manager) Enabled() (bool, error) {
	return true, nil
}

// Load 忽略 enableFirewall —— 它对应 `pfctl -e`，nftables 无等价物。
func (m Manager) Load(_ bool) error {
	return process.Run("nft", "-f", m.paths.PFAnchor)
}

// Unload 忽略 disableFirewall，理由同 Load。表不存在时不算错。
func (m Manager) Unload(_ bool) error {
	loaded, err := m.Loaded()
	if err != nil {
		return err
	}
	if !loaded {
		return nil
	}
	return process.Run("nft", "delete", "table", "inet", tableName)
}

func (m Manager) Loaded() (bool, error) {
	out, err := process.Output("nft", "list", "tables")
	if err != nil {
		return false, err
	}
	return tablesContain(string(out), tableName), nil
}

// tablesContain 解析 `nft list tables` 的输出，每行形如 `table inet opensurge`。
func tablesContain(output, name string) bool {
	for _, line := range strings.Split(output, "\n") {
		fields := strings.Fields(line)
		if len(fields) == 3 && fields[0] == "table" && fields[2] == name {
			return true
		}
	}
	return false
}
