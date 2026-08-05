package macosnetwork

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

var runCommand = run

func run(ctx context.Context, binary string, args ...string) (string, error) {
	commandCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(commandCtx, binary, args...)
	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("%s %s: %w: %s", binary, strings.Join(args, " "), err, strings.TrimSpace(output.String()))
	}
	return output.String(), nil
}
