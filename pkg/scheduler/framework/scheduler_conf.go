/*
Copyright 2018 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package framework

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v2"

	"volcano.sh/volcano/pkg/scheduler/conf"
)

var DefaultSchedulerConf = `
actions: "enqueue, allocate, backfill"
tiers:
- plugins:
  - name: priority
  - name: gang
  - name: conformance
- plugins:
  - name: overcommit
  - name: drf
  - name: predicates
  - name: proportion
  - name: nodeorder
`

func UnmarshalSchedulerConf(confStr string) ([]Action, []conf.Tier, []conf.Configuration, map[string]string, error) {
	var actions []Action

	schedulerConf := &conf.SchedulerConfiguration{}

	if err := yaml.Unmarshal([]byte(confStr), schedulerConf); err != nil {
		return nil, nil, nil, nil, err
	}
	// Set default settings for each plugin if not set
	for i, tier := range schedulerConf.Tiers {
		// drf with hierarchy enabled
		hdrf := false
		// proportion enabled
		proportion := false
		for j := range tier.Plugins {
			if tier.Plugins[j].Name == "drf" &&
				tier.Plugins[j].EnabledHierarchy != nil &&
				*tier.Plugins[j].EnabledHierarchy {
				hdrf = true
			}
			if tier.Plugins[j].Name == "proportion" {
				proportion = true
			}
			ApplyPluginConfDefaults(&schedulerConf.Tiers[i].Plugins[j])
		}
		if hdrf && proportion {
			return nil, nil, nil, nil, fmt.Errorf("proportion and drf with hierarchy enabled conflicts")
		}
	}

	actionNames := strings.Split(schedulerConf.Actions, ",")
	for _, actionName := range actionNames {
		if action, found := GetAction(strings.TrimSpace(actionName)); found {
			actions = append(actions, action)
		} else {
			return nil, nil, nil, nil, fmt.Errorf("failed to find Action %s, ignore it", actionName)
		}
	}

	return actions, schedulerConf.Tiers, schedulerConf.Configurations, schedulerConf.MetricsConfiguration, nil
}

func ReadSchedulerConf(confPath string) (string, error) {
	dat, err := os.ReadFile(confPath)
	if err != nil {
		return "", err
	}
	return string(dat), nil
}

// ApplyPluginConfDefaults sets option's filed to its default value if not set
func ApplyPluginConfDefaults(option *conf.PluginOption) {
	t := true

	if option.EnabledJobOrder == nil {
		option.EnabledJobOrder = &t
	}
	if option.EnabledJobReady == nil {
		option.EnabledJobReady = &t
	}
	if option.EnabledJobPipelined == nil {
		option.EnabledJobPipelined = &t
	}
	if option.EnabledJobEnqueued == nil {
		option.EnabledJobEnqueued = &t
	}
	if option.EnabledTaskOrder == nil {
		option.EnabledTaskOrder = &t
	}
	if option.EnabledPreemptable == nil {
		option.EnabledPreemptable = &t
	}
	if option.EnabledReclaimable == nil {
		option.EnabledReclaimable = &t
	}
	if option.EnabledQueueOrder == nil {
		option.EnabledQueueOrder = &t
	}
	if option.EnabledPredicate == nil {
		option.EnabledPredicate = &t
	}
	if option.EnabledBestNode == nil {
		option.EnabledBestNode = &t
	}
	if option.EnabledNodeOrder == nil {
		option.EnabledNodeOrder = &t
	}
	if option.EnabledTargetJob == nil {
		option.EnabledTargetJob = &t
	}
	if option.EnabledReservedNodes == nil {
		option.EnabledReservedNodes = &t
	}
	if option.EnabledVictim == nil {
		option.EnabledVictim = &t
	}
	if option.EnabledJobStarving == nil {
		option.EnabledJobStarving = &t
	}
	if option.EnabledOverused == nil {
		option.EnabledOverused = &t
	}
	if option.EnabledAllocatable == nil {
		option.EnabledAllocatable = &t
	}
}
