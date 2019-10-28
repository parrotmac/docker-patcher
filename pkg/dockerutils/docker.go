package dockerutils

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"go.uber.org/zap"
)

type Config struct {
	Logger *zap.Logger
	Client *client.Client
}

type Wrapper struct {
	logger *zap.Logger
	Client *client.Client
}

func NewWrapper(config *Config) (*Wrapper, error) {
	return &Wrapper{
		logger: config.Logger.Named("docker-wrapper"),
		Client: config.Client,
	}, nil
}

/*
LookupImage finds an image either by name/tag or by ID. We aim to match
behavior of the Docker CLI.

Example:

$ docker images
awesome			c0ffeea1
coff33			deadb33f

When executing a command such as `docker run -it --rm coff33` the Docker
CLI will run the second image, as it matches tags before image ID partials.
 */
func (w *Wrapper)LookupImage(ctx context.Context, query string) (*types.ImageSummary, error) {
	w.logger.Debug("lookup image", zap.String("query", query))
	allImages, err := w.Client.ImageList(ctx, types.ImageListOptions{All: true})
	if err != nil {
		return nil, err
	}

	// Look for an image with a matching tag first
	for _, img := range allImages {
		for _, imgTag := range img.RepoTags {
			if imgTag == query {
				w.logger.Debug("lookup image tag match", zap.String("ID", img.ID), zap.String("Tag", imgTag))
				return &img, nil
			}
		}
	}

	// Otherwise, attempt to match an image by ID
	for _, img := range allImages {
		if strings.Contains(img.ID, ":") {

			withoutPrefix := strings.Split(img.ID, ":")[1]

			if strings.HasPrefix(img.ID, query) {
				w.logger.Debug("lookup image ID match", zap.String("ID", img.ID))
				return &img, nil
			}

			if strings.HasPrefix(withoutPrefix, query) {
				w.logger.Debug("lookup image ID match", zap.String("ID", img.ID))
				return &img, nil
			}

		} else {
				// It's not clear if an ID *must* contain a prefix such as 'sha256:'
				// if it doesn't we'll attempt to match here
			if strings.HasPrefix(img.ID, query) {
				w.logger.Debug("lookup image ID match", zap.String("ID", img.ID))
				return &img, nil
			}
		}
	}

	return nil, fmt.Errorf("unable to find an image matching %s", query)
}


func (w *Wrapper) SaveImage(ctx context.Context, imageID string, destination io.Writer) error {
	requestedIDs := []string{imageID}
	imageSaver, err := w.Client.ImageSave(ctx, requestedIDs)
	if err != nil {
		return nil
	}
	defer imageSaver.Close()

	copyCount, err := io.Copy(destination, imageSaver)
	if err != nil {
		return err
	}
	w.logger.Debug("image size", zap.String("imageID", imageID), zap.Int64("size", copyCount))
	return nil
}

func (w *Wrapper) LoadImage(ctx context.Context, source io.Reader) error {
	resp, err := w.Client.ImageLoad(ctx, source, false)
	if err != nil {
		return err
	}

	if resp.JSON {
		respBody, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		w.logger.Debug("load-image-response", zap.String("respBody", string(respBody)))
	}

	return nil
}
