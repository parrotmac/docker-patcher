package bsdiff

import (
	"os"
	"os/exec"
)

/*
Note: This is unused and untested
*/
func Diff(oldPath string, newPath string, patchPath string) error {
	cmd := exec.Command("bsdiff", oldPath, newPath, patchPath)
	cmd.Stdout = os.Stdout // FIXME
	err := cmd.Run()
	if err != nil {
		return err
	}
	return nil
}
