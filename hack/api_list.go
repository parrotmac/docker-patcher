package main

import (
	"context"
	"log"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"go.uber.org/zap"

)

/*
Example of Docker API to list images & associated tags
 */

func main() {
	logger := zap.NewExample()

	cx, err := client.NewEnvClient()
	if err != nil {
		log.Fatalln(err)
	}

	results, err := cx.ImageList(context.TODO(), types.ImageListOptions{
		All:     true,
		Filters: filters.Args{},
	})
	if err != nil {
		logger.Fatal("failed to list images", zap.Error(err))
	}

	for _, res := range results {
		logger.Info("img",
			zap.String("ID", res.ID),
			zap.String("ParentID", res.ParentID),
			zap.Int64("Size", res.Size),
			zap.Int64("Container", res.Containers),
			zap.Int64("Created", res.Created),
			zap.Strings("RepoDigests", res.RepoDigests),
			zap.Strings("RepoTags", res.RepoTags),
		)
		for labelKey, labelValue := range res.Labels {
			logger.Info("labels for",
				zap.String("id", res.ID),
				zap.String("key", labelKey),
				zap.String("value", labelValue),
			)
		}
	}
}
