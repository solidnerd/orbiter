package core

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/Sirupsen/logrus"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/swarm"
	"github.com/docker/docker/client"
	"github.com/gianarb/orbiter/autoscaler"
	"github.com/gianarb/orbiter/provider"
)

// This function use diferent strategies to get information from
// the system itself to configure the autoloader.
// They can be environment variables for example or other systems.
func Autodetect(core *Core) error {
	autoDetectSwarmMode(core)
	if len(core.Autoscalers) == 0 {
		return errors.New("we didn't detect any autoscaling group")
	}
	return nil
}

func autoDetectSwarmMode(c *Core) {
	ctx := context.Background()
	dockerClient, err := client.NewEnvClient()
	if err != nil {
		logrus.WithField("error", err).Debug("Problem communication with Docker")
		return
	}
	info, err := dockerClient.Info(ctx)
	if err != nil {
		logrus.WithField("error", err).Debug("We didn't detect any Docker Swarm running")
		return
	}
	if info.Swarm.NodeID == "" {
		logrus.Debug("We didn't detect any Docker Swarm running")
		return
	}
	services, err := dockerClient.ServiceList(ctx, types.ServiceListOptions{})
	if err != nil {
		logrus.WithField("error", err).Debug("Bad comunication with Docker.")
		return
	}
	prov, _ := provider.NewSwarmProvider(map[string]string{})
	for _, service := range services {
		s, err := getAutoscalerByService(prov, service.Spec.Annotations)
		if err != nil {
			continue
		}
		c.Autoscalers[fmt.Sprintf("autodetect_swarm/%s", service.Spec.Annotations.Name)] = s
	}
}

func getAutoscalerByService(p autoscaler.Provider, an swarm.Annotations) (autoscaler.Autoscaler, error) {
	_, e := an.Labels["orbiter"]
	if e == false {
		return autoscaler.Autoscaler{}, errors.New("")
	}
	up := convertStringLabelToInt("orbiter.up", an.Labels)
	down := convertStringLabelToInt("orbiter.down", an.Labels)
	as := autoscaler.NewAutoscaler(p, an.Name, up, down)
	logrus.Debugf("autodetect_swarm/%s added to orbiter. UP %d, DOWN %d", an.Name, up, down)
	return as, nil
}

func convertStringLabelToInt(labelName string, labels map[string]string) int {
	row, e := labels[labelName]
	if e == true {
		i, err := strconv.ParseInt(row, 10, 64)
		if err != nil {
			return 1
		}
		return int(i)
	}
	return 1
}
