package patcher

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"

	"github.com/icedream/go-bsdiff"
	"go.uber.org/zap"

	"github.com/parrotmac/docker-patcher/pkg/dockerutils"
)

type Config struct {
	Logger           *zap.Logger
	TempFileLocation string
	DockerWrapper    *dockerutils.Wrapper
}

type Client struct {
	logger *zap.Logger
	tempFileLocation string
	dockerClient *dockerutils.Wrapper
}

func NewClient(config *Config) (*Client, error) {
	return &Client{
		logger:           config.Logger.Named("patcher"),
		tempFileLocation: config.TempFileLocation,
		dockerClient: config.DockerWrapper,
	}, nil
}

/*
Given 'old' and 'new' image references (ID + Name:Tag), writes a patch file
*/
func (c *Client) CreatePatch(fromID string, toID string, patchFile io.Writer) error {
	ctx := context.TODO()

	fromImg, err := c.dockerClient.LookupImage(ctx, fromID)
	if err != nil {
		return err
	}

	toImg, err := c.dockerClient.LookupImage(ctx, toID)
	if err != nil {
		return err
	}

	oldTmp, err := ioutil.TempFile("", "old")
	if err != nil {
		return err
	}
	defer os.Remove(oldTmp.Name())
	defer oldTmp.Close()

	newTmp, err := ioutil.TempFile("", "new")
	if err != nil {
		return err
	}
	defer os.Remove(newTmp.Name())
	defer newTmp.Close()

	// Save old img to temp file
	err = c.dockerClient.SaveImage(ctx, fromImg.ID, oldTmp)
	if err != nil {
		return err
	}

	// Flush to disk
	err = oldTmp.Sync()
	if err != nil {
		return err
	}

	// Seek back to beginning of file
	_, err = oldTmp.Seek(0, 0)
	if err != nil {
		return err
	}

	// Save new img to temp file
	err = c.dockerClient.SaveImage(ctx, toImg.ID, newTmp)
	if err != nil {
		return err
	}

	// Flush new data to disk
	err = newTmp.Sync()
	if err != nil {
		return err
	}

	// Seek back to beginning of file
	_, err = newTmp.Seek(0, 0)
	if err != nil {
		return err
	}

	// Run diff against old(from) and new(to) files, writing result to patch file
	err = bsdiff.Diff(oldTmp, newTmp, patchFile)
	if err != nil {
		return err
	}

	return nil
}

func (c *Client) PatchDockerImage(
	ctx context.Context,
	originalIdentifier string,
	patchFile io.ReadCloser,
	targetIdentifier string,
	targetHash string,
	) error {

	defer func() {
		if err := patchFile.Close(); err != nil {
			c.logger.Warn("couldn't cleanup temporary patch file", zap.Error(err))
		}
	}()

	srcTempFile, err := ioutil.TempFile("", "src-img")
	if err != nil {
		return err
	}
	defer os.Remove(srcTempFile.Name())
	defer srcTempFile.Close()
	log.Println("src temp file:", srcTempFile.Name())

	originalImg, err := c.dockerClient.LookupImage(ctx, originalIdentifier)
	if err != nil {
		return err
	}

	// save image
	err = c.dockerClient.SaveImage(ctx, originalImg.ID, srcTempFile)
	if err != nil {
		return err
	}

	_, err = srcTempFile.Seek(0, 0)
	if err != nil {
		return err
	}

	err = srcTempFile.Sync()
	if err != nil {
		return err
	}

	outputTempFile, err := ioutil.TempFile("", "out-img")
	if err != nil {
		return err
	}
	defer os.Remove(outputTempFile.Name())
	defer outputTempFile.Close()
	c.logger.Debug("output tempfile", zap.String("filename", outputTempFile.Name()))

	// apply patch
	err = bsdiff.Patch(srcTempFile, outputTempFile, patchFile)
	if err != nil {
		log.Fatalln(err)
	}
	c.logger.Debug("patch applied to", zap.String("filename", outputTempFile.Name()))

	err = outputTempFile.Sync()
	if err != nil {
		return err
	}

	_, err = outputTempFile.Seek(0, 0)
	if err != nil {
		return err
	}

	// load image
	err = c.dockerClient.LoadImage(ctx, outputTempFile)
	if err != nil {
		return err
	}

	_, err = c.dockerClient.LookupImage(ctx, targetIdentifier)
	if err != nil {
		return fmt.Errorf("unable to generate %s: %v", targetIdentifier, err)
	}

	return nil
}
