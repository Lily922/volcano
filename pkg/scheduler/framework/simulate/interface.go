package simulate

import (
	"context"

	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"

	"volcano.sh/volcano/pkg/scheduler/api"
)

type SimulatePlugin interface {
	Name() string
	// Simulate initializes the simulation-based scheduling configuration of the plugin.
	Simulate(sharedInfoLister SchedulingSimulateSharedInfo, pluginsRegister SimulatePluginsRegister)
}

type PrePredicateSimulatePlugin interface {
	SimulatePlugin
	// PrePredicateSimulate checks whether the task meets the requirements of being scheduled.
	// It can be considered as the dry run mode of PrePredicateFn function.
	PrePredicateSimulate(ctx context.Context, task *api.TaskInfo) error
}

type PredicateSimulatePlugin interface {
	SimulatePlugin
	// PredicateSimulate checks whether the given node fits the pod.
	// It can be considered as the dry run mode of PredicateFn function
	PredicateSimulate(ctx context.Context, task *api.TaskInfo, node *api.NodeInfo) ([]*api.Status, error)
}

type AllocateSimulatePlugin interface {
	SimulatePlugin
	// AllocateSimulate is called when the pod is simulated assigned to a node.
	// It can be considered as the dry run mode of AllocateFn function.
	AllocateSimulate(ctx context.Context, task *api.TaskInfo, node *api.NodeInfo) error
}

type DeallocateSimulatePlugin interface {
	SimulatePlugin
	// DeallocateSimulate is called when the pod is simulated removed from a node or a reserved pod was failed
	// to be simulated assigned to a node.
	DeallocateSimulate(ctx context.Context, task *api.TaskInfo, node *api.NodeInfo) error
}

type SchedulingSimulate interface {
	// OpenSimulateSession is called in every simulation-based scheduling cycle,
	// to initialize configuration of the simulation-based scheduling cycle.
	OpenSimulateSession()
	// RunPrePredicateSimulatePlugins runs the set of configured PrePredicateSimulate plugins.
	RunPrePredicateSimulatePlugins(ctx context.Context, task *api.TaskInfo) error
	// RunPredicateSimulatePlugins runs the set of configured PredicateSimulate plugins.
	RunPredicateSimulatePlugins(ctx context.Context, task *api.TaskInfo, node *api.NodeInfo) ([]*api.Status, error)
	// RunAllocateSimulatePlugins runs the set of configured AllocateSimulate plugins.
	RunAllocateSimulatePlugins(ctx context.Context, task *api.TaskInfo, node *api.NodeInfo) error
	// RunDeallocateSimulatePlugins runs the set of configured DeallocateSimulate plugins.
	RunDeallocateSimulatePlugins(ctx context.Context, task *api.TaskInfo, node *api.NodeInfo) error
}

type SimulatePluginsRegister interface {
	// AddPrePredicateSimulatePlugin registers PrePredicateSimulatePlugin to the simulation-based scheduler.
	AddPrePredicateSimulatePlugin(p PrePredicateSimulatePlugin)
	// AddPredicateSimulatePlugin registers PredicateSimulatePlugin to the simulation-based scheduler.
	AddPredicateSimulatePlugin(p PredicateSimulatePlugin)
	// AddAllocateSimulatePlugin registers AllocateSimulatePlugin to the simulation-based scheduler.
	AddAllocateSimulatePlugin(p AllocateSimulatePlugin)
	// AddDeallocateSimulatePlugin registers DeallocateSimulatePlugin to the simulation-based scheduler.
	AddDeallocateSimulatePlugin(p DeallocateSimulatePlugin)
}

type SchedulingSimulateSharedInfo interface {
	KubeClient() kubernetes.Interface
	InformerFactory() informers.SharedInformerFactory
}
