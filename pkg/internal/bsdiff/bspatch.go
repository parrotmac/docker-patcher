package bsdiff

import (
	"os"
	"os/exec"
)

/*
Note: This is unused and untested
*/
func Patch(oldPath string, newPath string, patchPath string) error {
	cmd := exec.Command("bspatch", oldPath, newPath, patchPath)
	cmd.Stdout = os.Stdout // FIXME
	err := cmd.Run()
	if err != nil {
		return err
	}
	return nil
}
