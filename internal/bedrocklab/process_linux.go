package bedrocklab

import (
	"context"
	"errors"
	"os/exec"
)

func hideCommand(cmd *exec.Cmd) {}
func CopyReadableExecutable(ctx context.Context, source, root string) (string, error) {
	return "", errors.New("package-context copying requires Windows")
}
