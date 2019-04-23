package main

import (
	"os"

	"github.com/jessevdk/go-flags"
	"github.com/sirupsen/logrus"

	"github.com/parrotmac/docker-patcher/pkg/patcher"
)

type patchJob struct {
	oldID         string
	newID         string
	newRepoTag    string
	patchFilePath string
}

var cliOptions struct {
	NewTag string `short:"t" long:"new-tag" description:"Repo:tag name for new image (e.g. nginx:1.15.12 or example.com/cool_thing:1.2.3)"`
	Help   bool   `short:"h" long:"help" description:"Display help"`
}

const helpMessage = `Docker Image Patcher
Usage:
	dipatch [--new-tag] original_id new_id patch_file
  	dipatch -h | --help

Options:
	-h --help		Show this message
	-t --new-tag	repo:tag for new image (e.g. nginx:1.15.12 or example.com/cool_thing:1.2.3)

Description:
	Docker Image Patcher builds a new Docker image using an original image and a patch file. ` + "`original_id and new_id`" + `
	should be the docker image IDs of the original/from image and new/to image, respectively. These IDs can be in either
	long (sha256:xxxxxxxxxxxxxxxxxx...) or short (01234567890ab) form. ` + "`patch_file`" + ` is the path to the patch file.
	
	-t --new-tag Optionally provide a tag to be applied to the image once created and loaded.`

func setupPatch() *patchJob {
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

	newImageTag := cliOptions.NewTag
	if newImageTag == "" {
		logrus.Warnln("new repo:tag was not specified (-t)")
	}

	return &patchJob{
		oldID:         fromID,
		newID:         toID,
		patchFilePath: patchFilePath,
		newRepoTag:    newImageTag,
	}
}

func main() {
	job := setupPatch()

	patchFile, err := os.Open(job.patchFilePath)
	if err != nil {
		logrus.Fatalln("unable to open patch file:", err)
	}
	defer func() {
		if err = patchFile.Close(); err != nil {
			logrus.Warnln(err)
		}
	}()

	err = patcher.PatchDockerImage(job.oldID, patchFile, job.newID, "")
	if err != nil {
		logrus.Fatalln(err)
	}

	err = patcher.TagImage(job.newID, job.newRepoTag)
	if err != nil {
		logrus.Fatalln(err)
	}
}
