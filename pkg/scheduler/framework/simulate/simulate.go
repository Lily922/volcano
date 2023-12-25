package simulate

import (
	"context"
	"reflect"
	"sync"

	"k8s.io/klog/v2"

	"volcano.sh/volcano/pkg/scheduler/api"
	"volcano.sh/volcano/pkg/scheduler/conf"
	"volcano.sh/volcano/pkg/scheduler/framework"
)

type Simulator struct {
	SchedulerConf    string
	Tiers            []conf.Tier
	SharedInfoLister SchedulingSimulateSharedInfo

	Lock                        sync.Mutex
	Plugins                     map[string]bool
	PrePredicateSimulatePlugins map[string]PrePredicateSimulatePlugin
	PredicateSimulatePlugins    map[string]PredicateSimulatePlugin
	AllocateSimulatePlugins     map[string]AllocateSimulatePlugin
	DeallocateSimulatePlugins   map[string]DeallocateSimulatePlugin
}

func NewSimulator(sharedInfoLister SchedulingSimulateSharedInfo, schedulerConf string) SchedulingSimulate {
	// TODO: 这里需要初始化模拟调度依赖的资源类型
	sharedInfoLister.InformerFactory().Policy().V1().PodDisruptionBudgets().Informer()
	sharedInfoLister.InformerFactory().Core().V1().Pods().Informer()
	sharedInfoLister.InformerFactory().Core().V1().Namespaces().Informer()
	sharedInfoLister.InformerFactory().Core().V1().Services().Informer()
	sharedInfoLister.InformerFactory().Core().V1().PersistentVolumeClaims().Informer()
	sharedInfoLister.InformerFactory().Core().V1().PersistentVolumeClaims()
	sharedInfoLister.InformerFactory().Core().V1().PersistentVolumes()
	sharedInfoLister.InformerFactory().Storage().V1().StorageClasses()
	sharedInfoLister.InformerFactory().Storage().V1().CSINodes()
	sharedInfoLister.InformerFactory().Core().V1().ResourceQuotas()

	simulator := &Simulator{
		SchedulerConf:    schedulerConf,
		SharedInfoLister: sharedInfoLister,
	}
	return simulator
}

func (s *Simulator) OpenSimulateSession() {
	s.Plugins = make(map[string]bool)
	s.PrePredicateSimulatePlugins = make(map[string]PrePredicateSimulatePlugin)
	s.PredicateSimulatePlugins = make(map[string]PredicateSimulatePlugin)
	s.AllocateSimulatePlugins = make(map[string]AllocateSimulatePlugin)
	s.DeallocateSimulatePlugins = make(map[string]DeallocateSimulatePlugin)

	s.LoadSchedulerConf()

	for _, tier := range s.Tiers {
		for _, plugin := range tier.Plugins {
			if pb, found := framework.GetPluginBuilder(plugin.Name); !found {
				klog.Errorf("Failed to get plugin %s.", plugin.Name)
			} else {
				plg := pb(plugin.Arguments)
				var simulatePlugin SimulatePlugin
				simulatePluginType := reflect.TypeOf(&simulatePlugin).Elem()
				if !reflect.TypeOf(plg).Implements(simulatePluginType) {
					klog.V(5).Infof("Plugin (%s) does not support simulation-based scheduling", plg.Name())
					continue
				}
				klog.V(5).Infof("Plugin (%s) supports simulation-based scheduling", plg.Name())
				reflect.ValueOf(&simulatePlugin).Elem().Set(reflect.ValueOf(plg))
				simulatePlugin.Simulate(s.SharedInfoLister, s)
				s.Plugins[simulatePlugin.Name()] = true
			}
		}
	}
	return
}

func (s *Simulator) AddPrePredicateSimulatePlugin(p PrePredicateSimulatePlugin) {
	s.Lock.Lock()
	defer s.Lock.Unlock()
	s.PrePredicateSimulatePlugins[p.Name()] = p
}

func (s *Simulator) AddPredicateSimulatePlugin(p PredicateSimulatePlugin) {
	s.Lock.Lock()
	defer s.Lock.Unlock()
	s.PredicateSimulatePlugins[p.Name()] = p
}

func (s *Simulator) AddAllocateSimulatePlugin(p AllocateSimulatePlugin) {
	s.Lock.Lock()
	defer s.Lock.Unlock()
	s.AllocateSimulatePlugins[p.Name()] = p
}

func (s *Simulator) AddDeallocateSimulatePlugin(p DeallocateSimulatePlugin) {
	s.Lock.Lock()
	defer s.Lock.Unlock()
	s.DeallocateSimulatePlugins[p.Name()] = p
}

func (s *Simulator) RunPrePredicateSimulatePlugins(ctx context.Context, task *api.TaskInfo) error {
	for _, tier := range s.Tiers {
		for _, plugin := range tier.Plugins {
			if !s.IsPluginSimulateSchedulingEnable(plugin) {
				continue
			}
			p, found := s.PrePredicateSimulatePlugins[plugin.Name]
			if !found {
				continue
			}
			err := p.PrePredicateSimulate(ctx, task)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *Simulator) RunPredicateSimulatePlugins(ctx context.Context, task *api.TaskInfo, node *api.NodeInfo) ([]*api.Status, error) {
	predicateStatus := make([]*api.Status, 0)
	for _, tier := range s.Tiers {
		for _, plugin := range tier.Plugins {
			if !s.IsPluginSimulateSchedulingEnable(plugin) {
				continue
			}
			p, found := s.PredicateSimulatePlugins[plugin.Name]
			if !found {
				continue
			}
			status, err := p.PredicateSimulate(ctx, task, node)
			predicateStatus = append(predicateStatus, status...)
			if err != nil {
				return predicateStatus, err
			}
		}
	}
	return predicateStatus, nil
}

func (s *Simulator) RunAllocateSimulatePlugins(ctx context.Context, task *api.TaskInfo, node *api.NodeInfo) error {
	for _, tier := range s.Tiers {
		for _, plugin := range tier.Plugins {
			if !s.IsPluginSimulateSchedulingEnable(plugin) {
				continue
			}
			p, found := s.AllocateSimulatePlugins[plugin.Name]
			if !found {
				continue
			}
			err := p.AllocateSimulate(ctx, task, node)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *Simulator) RunDeallocateSimulatePlugins(ctx context.Context, task *api.TaskInfo, node *api.NodeInfo) error {
	for _, tier := range s.Tiers {
		for _, plugin := range tier.Plugins {
			if !s.IsPluginSimulateSchedulingEnable(plugin) {
				continue
			}
			p, found := s.DeallocateSimulatePlugins[plugin.Name]
			if !found {
				continue
			}
			err := p.DeallocateSimulate(ctx, task, node)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *Simulator) LoadSchedulerConf() {
	s.Tiers = []conf.Tier{}
	config := framework.DefaultSchedulerConf
	if len(s.SchedulerConf) != 0 {
		if con, err := framework.ReadSchedulerConf(s.SchedulerConf); err != nil {
			klog.Errorf("Failed to read scheduler configuration '%s', using previous configuration: %v",
				s.SchedulerConf, err)
			return
		} else {
			config = con
		}
	}

	_, plugins, _, _, err := framework.UnmarshalSchedulerConf(config)
	if err != nil {
		klog.Errorf("scheduler config %s is invalid: %v", config, err)
		return
	}
	s.Tiers = plugins
}

func (s *Simulator) IsPluginSimulateSchedulingEnable(option conf.PluginOption) bool {
	if option.EnabledPredicate == nil || !*option.EnabledPredicate {
		return false
	}

	// TODO check enabled simulation-based Predicate ?
	if _, ok := s.Plugins[option.Name]; !ok {
		return false
	}
	return true
}
