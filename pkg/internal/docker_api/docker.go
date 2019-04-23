package docker_api

import (
	"context"
	"io"
	"io/ioutil"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/sirupsen/logrus"
)

type APIClient struct {
	Client *client.Client
}

func NewDefaultAPIClient() (*APIClient, error) {
	cx, err := client.NewClientWithOpts(client.FromEnv, client.WithVersion("1.39"))
	if err != nil {
		return nil, err
	}
	return &APIClient{
		Client: cx,
	}, nil
}

func ShortenLongID(longID string) string {
	parts := strings.Split(longID, ":")
	return parts[1][0:12]
}

func (a *APIClient) ListImages(ctx context.Context) ([]types.ImageSummary, error) {
	return a.Client.ImageList(ctx, types.ImageListOptions{
		All: true,
	})
}

func (a *APIClient) SaveImage(ctx context.Context, imageID string, destination io.Writer) error {
	requestedIDs := []string{imageID}
	imageSaver, err := a.Client.ImageSave(ctx, requestedIDs)
	if err != nil {
		return nil
	}
	defer func() {
		if err = imageSaver.Close(); err != nil {
			logrus.Warnln(err)
		}
	}()

	copyCount, err := io.Copy(destination, imageSaver)
	if err != nil {
		return err
	}
	logrus.Debugf("Saved %d bytes", copyCount)
	return nil
}

func (a *APIClient) LoadImage(ctx context.Context, source io.ReadCloser) error {
	defer func() {
		if err := source.Close(); err != nil {
			logrus.Warnln(err)
		}
	}()

	resp, err := a.Client.ImageLoad(ctx, source, false)
	if err != nil {
		return err
	}
	if resp.JSON {
		respBody, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		logrus.Debugln(string(respBody))
	}

	return nil
}

func (a *APIClient) TagImage(ctx context.Context, imageID, tag string) error {
	return a.Client.ImageTag(ctx, imageID, tag)
}
