package main

import (
	"os"

	"github.com/jessevdk/go-flags"
	"github.com/sirupsen/logrus"

	"github.com/parrotmac/docker-patcher/pkg/patcher"
)

type diffJob struct {
	oldID         string
	newID         string
	patchFilePath string
}

var cliOptions struct {
	NewTag string `short:"t" long:"new-tag" description:"Repo:tag name for new image (e.g. nginx:1.15.12 or example.com/cool_thing:1.2.3)"`
	Help   bool   `short:"h" long:"help" description:"Display help"`
}

const helpMessage = `Docker Image Diff
Usage:
	didiff original_id new_id patch_file
  	didiff -h | --help

Options:
	-h --help		Show this message

Description:
	Docker Image Diff builds a patch file describing the binary diff of two Docker images. ` + "`original_id and new_id`" + `
	should be the docker image IDs of the original/from image and new/to image, respectively. These IDs can be in either
	long (sha256:xxxxxxxxxxxxxxxxxx...) or short (01234567890ab) form.`

func setupDiff() *diffJob {
	parser := flags.NewParser(&cliOptions, 0)
	parser.Usage = helpMessage
	args, err := parser.Parse()
	if err != nil {
		logrus.Fatalln(err)
	}

	if cliOptions.Help {
		parser.WriteHelp(os.Stdout)
		os.Exit(0)
	}

	if len(args) != 3 {
		parser.WriteHelp(os.Stdout)
		os.Exit(1)
	}
	fromID := args[0]
	toID := args[1]
	patchFilePath := args[2]

	return &diffJob{
		oldID:         fromID,
		newID:         toID,
		patchFilePath: patchFilePath,
	}
}

func main() {
	job := setupDiff()

	patchFile, err := os.Create(job.patchFilePath)
	if err != nil {
		logrus.Fatalln("unable to open patch file:", err)
	}
	defer func() {
		err = patchFile.Close()
		if err != nil {
			logrus.Errorln(err)
		}
	}()

	err = patcher.CreatePatch(job.oldID, job.newID, patchFile)
	if err != nil {
		logrus.Fatalln("unable to create patch:", err)
	}
}
