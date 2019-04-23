package patcher

import (
	"context"
	"errors"
	"fmt"
	"github.com/parrotmac/docker-patcher/pkg/internal/utils"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strings"

	"github.com/icedream/go-bsdiff"
	"github.com/sirupsen/logrus"

	"github.com/parrotmac/docker-patcher/pkg/internal/docker_api"
)

/*
Given 'old' and 'new' image references (ID + Name:Tag), writes a patch file
TODO(isaac) Dedupe from/to (old/new) logic
*/
func CreatePatch(fromID string, toID string, patchFile io.Writer) error {
	if docker_api.ShortenLongID(fromID) == docker_api.ShortenLongID(toID) {
		return errors.New("IDs cannot be the same")
	}

	ctx := context.TODO()
	dockerClient, err := docker_api.NewDefaultAPIClient()
	if err != nil {
		return err
	}

	oldImgAvailable := false
	newImgAvailable := false

	images, err := dockerClient.ListImages(ctx)
	for _, img := range images {
		if img.ID == fromID {
			oldImgAvailable = true
		}
		if img.ID == toID {
			newImgAvailable = true
		}
	}

	if !oldImgAvailable {
		return fmt.Errorf("image with reference '%s' in not availalbe (it may need to be pulled)", fromID)
	}

	if !newImgAvailable {
		return fmt.Errorf("image with reference '%s' in not availalbe (it may need to be pulled)", toID)
	}

	oldTmp, err := ioutil.TempFile("", "old")
	if err != nil {
		return err
	}

	newTmp, err := ioutil.TempFile("", "new")
	if err != nil {
		return err
	}

	defer func() {
		err = oldTmp.Close()
		if err != nil {
			logrus.Warnln(err)
		}

		err = os.Remove(oldTmp.Name())
		if err != nil {
			logrus.Warnln(err)
		}

		err = newTmp.Close()
		if err != nil {
			logrus.Warnln(err)
		}

		err = os.Remove(newTmp.Name())
		if err != nil {
			logrus.Warnln(err)
		}
	}()

	// Save old img to temp file
	err = dockerClient.SaveImage(ctx, fromID, oldTmp)
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
	err = dockerClient.SaveImage(ctx, toID, newTmp)
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

func PatchDockerImage(originalIdentifier string, patchFile io.ReadCloser, targetIdentifier, targetHash string) error {
	// Setup
	ctx := context.TODO()
	dockerClient, err := docker_api.NewDefaultAPIClient()
	if err != nil {
		return err
	}

	if len(originalIdentifier) > 12 && strings.Contains(originalIdentifier, ":") {
		originalIdentifier = strings.Split(originalIdentifier, ":")[1][0:12]
	}

	if len(targetIdentifier) > 12 && strings.Contains(targetIdentifier, ":") {
		targetIdentifier = strings.Split(targetIdentifier, ":")[1][0:12]
	}

	defer func() {
		if err = patchFile.Close(); err != nil {
			logrus.Warnln(err)
		}
	}()

	srcTempFile, err := ioutil.TempFile("", "src-img")
	if err != nil {
		return err
	}
	log.Println("src temp file:", srcTempFile.Name())

	// lookup image
	preLoadImages, err := dockerClient.ListImages(ctx)
	if err != nil {
		return err
	}

	hasImg := false
	for _, img := range preLoadImages {
		if strings.HasPrefix(strings.Split(img.ID, ":")[1], originalIdentifier) {
			hasImg = true
			break
		}
	}
	if !hasImg {
		return fmt.Errorf("img %s doesn't exist", originalIdentifier)
	}

	// save image
	err = dockerClient.SaveImage(ctx, originalIdentifier, srcTempFile)
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
	logrus.Debugln("output temp file:", outputTempFile.Name())

	// apply patch
	err = bsdiff.Patch(srcTempFile, outputTempFile, patchFile)
	if err != nil {
		log.Fatalln(err)
	}
	logrus.Debugln("patch applied to", outputTempFile.Name())

	err = outputTempFile.Sync()
	if err != nil {
		return err
	}

	_, err = outputTempFile.Seek(0, 0)
	if err != nil {
		log.Fatalln("Unable to seek:", err)
	}

	// verify sha sum
	// TODO(isaac) Should verification happen? Might be superfluous
	outputShaSum, _ := utils.CalculateFileSha256Sum(outputTempFile.Name())
	logrus.Debugln("New image SHA sum:", outputShaSum)
	// TODO: Compare against a known good value. This would prevent (or even attempting) loading a malformed image

	// remove original tar
	err = os.Remove(srcTempFile.Name())
	if err != nil {
		logrus.Warnf("unable to cleanup temporary file: %+v", err)
	}

	// load image
	err = dockerClient.LoadImage(ctx, outputTempFile)
	if err != nil {
		log.Fatalln(err)
	}

	// remove output tar
	err = os.Remove(outputTempFile.Name())
	if err != nil {
		logrus.Warnf("unable to cleanup temporary file: %+v", err)
	}

	// verify identifier
	postLoadImages, err := dockerClient.ListImages(ctx)
	if err != nil {
		// TODO Return on err? Load may have been successful
		logrus.Warnln(err)
	}
	for _, img := range postLoadImages {
		if strings.HasPrefix(strings.Split(img.ID, ":")[1], targetIdentifier) {
			logrus.Infoln("Patch was successful. %s is now available.", targetIdentifier)
			return nil
		}
	}
	return fmt.Errorf("unable to patch %s to %s", originalIdentifier, targetIdentifier)
}

func TagImage(imgID, tag string) error {
	ctx := context.TODO()
	dockerClient, err := docker_api.NewDefaultAPIClient()
	if err != nil {
		return err
	}
	return dockerClient.TagImage(ctx, imgID, tag)
}
