package vm

import (
	_ "embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/accuknox/accuknox-cli-v2/pkg/logger"
)

//go:embed rra
var rraBinary []byte

var fileName = "accuknox-rra_%s_result.json"

func PrepareRRACommand(profile, benchmark, authToken, label, url, tenantID, clusterName, clusterID string) string {
	command := fmt.Sprintf(`analyze \
    --profile %s \
    --benchmark %s \
    --auth-token %s \
    --url %s \
    --tenant-id %s \
    --cluster-name %s \
    --label %s`,
		profile, benchmark, authToken, url, tenantID, clusterName, label,
	)
	if clusterID != "" {
		command += fmt.Sprintf(" --cluster-id %s", clusterID)
	}
	command += " --json"

	return command
}
func ExecCommand(command, path, benchmark string, save bool) error {
	tmpDir, _ := os.MkdirTemp("", "rra")
	defer os.RemoveAll(tmpDir)

	rraPath := filepath.Join(tmpDir, "rra")
	err := os.WriteFile(rraPath, rraBinary, 0700) // #nosec G306 need perms for write and execute
	if err != nil {
		return fmt.Errorf("failed to write RRA binary: %v", err)
	}
	command = fmt.Sprintf("%s %s", rraPath, strings.TrimSpace(command))

	cmd := exec.Command("/bin/sh", "-c", command)

	filePath := ""

	if save {
		filePath = filepath.Join(path, fmt.Sprintf(fileName, benchmark))
		file, err := os.Create(filePath)
		if err != nil {
			return fmt.Errorf("failed to create output file: %v", err)
		}
		logger.Info1("Output file created at %s", filePath)
		defer file.Close()
		cmd.Stdout = file
		cmd.Stderr = file
	} else {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}
	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("Unable to execute RRA: %s", err.Error())
	}
	return nil
}
