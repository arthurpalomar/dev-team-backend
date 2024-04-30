package internal

import (
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
)

func FindContainerByName(name string) (*types.Container, error) {
	filtersArgs := filters.NewArgs()
	filtersArgs.Add("name", name)

	containers, err := Docker.Client.ContainerList(Docker.Ctx, types.ContainerListOptions{
		All:     true,
		Filters: filtersArgs,
	})
	if err != nil {
		return nil, err
	}

	if len(containers) == 0 {
		return nil, nil
	}
	return &containers[0], nil
}
