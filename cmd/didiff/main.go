package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/docker/docker/client"
	"github.com/jessevdk/go-flags"
	"go.uber.org/zap"

	"github.com/parrotmac/docker-patcher/pkg/dockerutils"
	"github.com/parrotmac/docker-patcher/pkg/patcher"
)

type didiffArgs struct {
	action string
	dockerURL string
	tempFileLocation string
	diffJob *diffJob
	patchJob *patchJob
}

type diffJob struct {
	oldID         string
	newID         string
	patchFilePath string
}

type patchJob struct {
	oldID         string
	newID         string
	newRepoTag    string
	patchFilePath string
}

var cliOptions struct {
	DockerHostURL string `short:"d" long:"docker-host" description:"URL of Docker daemon (e.g. '/var/run/docker.sock' or 'localhost:2375')"`
	TempDirectoryLocation string `short:"e" long:"temp-directory" description:"Location used for temporary files. Defaults to /tmp"`
	NewTag string `short:"t" long:"new-tag" description:"Repo:tag name for new image (e.g. nginx:1.15.12 or example.com/cool_thing:1.2.3)"`
	Help   bool   `short:"h" long:"help" description:"Display help"`
}

const helpMessage = `Docker Image Diffing & Patching
Usage:
	didiff create original_id new_id patch_file
	didiff apply [--new-tag] original_id new_id patch_file
  	didiff -h | --help

Options:
	-h --help			Show this message
	-t --new-tag		repo:tag for new image (e.g. nginx:1.15.12 or example.com/cool_thing:1.2.3) (apply only)
	-d --docker-host	URL of Docker daemon (e.g. '/var/run/docker.sock' or 'localhost:2375')
	-e --temp-directory Location used for temporary files. Defaults to /tmp

Description:
	Docker Image Diff builds a patch file describing the binary diff of two Docker images. ` + "`original_id and new_id`" + `
	should be the docker image IDs of the original/from image and new/to image, respectively. These IDs can be in either
	long (sha256:xxxxxxxxxxxxxxxxxx...) or short (01234567890ab) form.

	Docker Image Patcher builds a new Docker image using an original image and a patch file. ` + "`original_id and new_id`" + `
	should be the docker image IDs of the original/from image and new/to image, respectively. These IDs can be in either
	long (sha256:xxxxxxxxxxxxxxxxxx...) or short (01234567890ab) form. ` + "`patch_file`" + ` is the path to the patch file.

	-t --new-tag Optionally provide a tag to be applied to the image once created and loaded.`

const (
	actionCreate = "create"
	actionApply = "apply"
)

func setupDidiff(logger *zap.Logger) (*didiffArgs, error) {
	parser := flags.NewParser(&cliOptions, 0)
	parser.Usage = helpMessage
	args, err := parser.Parse()
	if err != nil {
		return nil, fmt.Errorf("bad action %v", err)
	}

	if cliOptions.Help {
		parser.WriteHelp(os.Stdout)
		os.Exit(0)
	}

	job := &didiffArgs{}

	if len(args) < 1 {
		parser.WriteHelp(os.Stdout)
		return nil, errors.New("must specify an action [create|action]")
	}

	actionValue := args[0]
	logger.Debug("requested action", zap.String("action", actionValue))
	switch actionValue {
	case actionCreate:
		job.action = actionCreate

		if len(args) != 4 {
			return nil, errors.New("bad arguments")
		}

		fromID := args[1]
		toID := args[2]
		patchFilePath := args[3]

		job.diffJob = &diffJob{
			oldID:         fromID,
			newID:         toID,
			patchFilePath: patchFilePath,
		}

	case actionApply:
		job.action = actionApply

		if len(args) != 4 {
			return nil, errors.New("bad arguments")
		}

		fromID := args[1]
		toID := args[2]
		patchFilePath := args[3]

		newImageTag := cliOptions.NewTag

		job.patchJob = &patchJob{
			oldID:         fromID,
			newID:         toID,
			patchFilePath: patchFilePath,
			newRepoTag:    newImageTag,
		}

	default:
		return nil, fmt.Errorf("bad action %v", actionValue)
	}

	return job, nil
}

func main() {
	// Setup logger
	logger := zap.NewExample()

	// Setup context
	ctx := context.TODO()

	// Parse options
	job, err := setupDidiff(logger)
	if err != nil {
		// TODO
		log.Println(err)
		log.Println(helpMessage)
		os.Exit(1)
	}

	// Init docker client
	// TODO: Hook up client to arguments
	dockerClient, err := client.NewEnvClient()
	if err != nil {
		logger.Fatal("failed to init docker client", zap.Error(err))
	}

	// Init wrapper for Docker actions
	dockerWrapper, err := dockerutils.NewWrapper(&dockerutils.Config{
		Logger: logger,
		Client: dockerClient,
	})
	if err != nil {
		logger.Fatal("failed to init docker wrapper", zap.Error(err))
	}

	// Create patching client
	patchClient, err := patcher.NewClient(&patcher.Config{
		Logger:           logger,
		TempFileLocation: "",
		DockerWrapper:    dockerWrapper,
	})
	if err != nil {
		logger.Fatal("failed to init patch client", zap.Error(err))
	}

	// Perform specified action
	if job.action == actionCreate {
		if job.diffJob == nil {
			logger.Fatal("unable to get values for operation")
		}
		patchJob := job.diffJob

		patchFile, err := os.Create(patchJob.patchFilePath)
		if err != nil {
			logger.Fatal("unable to open patch file", zap.Error(err))
		}

		defer func() {
			err = patchFile.Close()
			if err != nil {
				logger.Warn("unable to close patch file", zap.Error(err))
			}
		}()


		if err = patchClient.CreatePatch(patchJob.oldID, patchJob.newID, patchFile); err != nil {
			logger.Fatal("unable to create patch", zap.Error(err))
		}
		return
	}

	// Perform specified action
	if job.action == actionApply {
		if job.patchJob == nil {
			logger.Fatal("unable to get values for operation")
		}
		patchJob := job.patchJob

		patchFile, err := os.Open(patchJob.patchFilePath)
		if err != nil {
			logger.Fatal("unable to open patch file", zap.Error(err))
		}
		defer func() {
			if err = patchFile.Close(); err != nil {
				logger.Warn("unable to close patch file", zap.Error(err))
			}
		}()

		err = patchClient.PatchDockerImage(ctx, patchJob.oldID, patchFile, patchJob.newID, "")
		if err != nil {
			logger.Fatal("unable to patch image", zap.Error(err))
		}

		// FIXME: Replace removed API call
		//if err := patcher.TagImage(patchJob.newID, patchJob.newRepoTag); err != nil {
		//	logger.Fatal("unable to patch", zap.Error(err))
		//}

		return
	}

	logger.Fatal("unable to perform specified action")
}
