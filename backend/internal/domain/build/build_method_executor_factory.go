package build

import (
	"fmt"

	tektonclient "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	"go.uber.org/zap"
	"k8s.io/client-go/kubernetes"
)

// InfrastructureAwareExecutorFactory creates executors based on infrastructure type
type InfrastructureAwareExecutorFactory struct {
	localFactory       BuildMethodExecutorFactory
	tektonFactory      *TektonExecutorFactory
	infrastructureType string
}

// NewInfrastructureAwareExecutorFactory creates a factory that chooses executors based on infrastructure
func NewInfrastructureAwareExecutorFactory(
	localFactory BuildMethodExecutorFactory,
	tektonFactory *TektonExecutorFactory,
	infrastructureType string,
) BuildMethodExecutorFactory {
	return &InfrastructureAwareExecutorFactory{
		localFactory:       localFactory,
		tektonFactory:      tektonFactory,
		infrastructureType: infrastructureType,
	}
}

// CreateExecutor creates an executor based on infrastructure type
func (f *InfrastructureAwareExecutorFactory) CreateExecutor(method BuildMethod) (BuildMethodExecutor, error) {
	if f.infrastructureType == "kubernetes" && f.tektonFactory != nil {
		return f.tektonFactory.CreateExecutor(method)
	}
	// Default to local execution
	return f.localFactory.CreateExecutor(method)
}

// GetSupportedMethods returns supported methods (same for both infrastructures)
func (f *InfrastructureAwareExecutorFactory) GetSupportedMethods() []BuildMethod {
	return f.localFactory.GetSupportedMethods()
}

// TektonExecutorFactory creates Tekton-based executors
type TektonExecutorFactory struct {
	k8sClient      kubernetes.Interface
	tektonClient   tektonclient.Interface
	logger         *zap.Logger
	namespaceMgr   NamespaceManager
	pipelineMgr    PipelineManager
	templateEngine TemplateEngine
	service        BuildExecutionService
	configRepo     BuildMethodConfigRepository
	buildRepo      Repository
	clientProvider TektonClientProvider
	registryAuth   RegistryDockerConfigResolver
	repositoryAuth RepositoryGitAuthResolver
	executor       BuildMethodExecutor
}

// NewTektonExecutorFactory creates a Tekton executor factory
func NewTektonExecutorFactory(
	k8sClient kubernetes.Interface,
	tektonClient tektonclient.Interface,
	logger *zap.Logger,
	namespaceMgr NamespaceManager,
	pipelineMgr PipelineManager,
	templateEngine TemplateEngine,
	service BuildExecutionService,
	configRepo BuildMethodConfigRepository,
	buildRepo Repository,
	clientProvider TektonClientProvider,
	registryAuth RegistryDockerConfigResolver,
	repositoryAuth RepositoryGitAuthResolver,
) *TektonExecutorFactory {
	factory := &TektonExecutorFactory{
		k8sClient:      k8sClient,
		tektonClient:   tektonClient,
		logger:         logger,
		namespaceMgr:   namespaceMgr,
		pipelineMgr:    pipelineMgr,
		templateEngine: templateEngine,
		service:        service,
		configRepo:     configRepo,
		buildRepo:      buildRepo,
		clientProvider: clientProvider,
		registryAuth:   registryAuth,
		repositoryAuth: repositoryAuth,
	}
	factory.executor = NewMethodTektonExecutor(
		factory.k8sClient,
		factory.tektonClient,
		factory.logger,
		factory.namespaceMgr,
		factory.pipelineMgr,
		factory.templateEngine,
		factory.service,
		factory.configRepo,
		factory.buildRepo,
		factory.clientProvider,
		factory.registryAuth,
		factory.repositoryAuth,
	)
	return factory
}

// CreateExecutor creates a Tekton-based executor
func (f *TektonExecutorFactory) CreateExecutor(method BuildMethod) (BuildMethodExecutor, error) {
	if f.executor == nil {
		f.executor = NewMethodTektonExecutor(
			f.k8sClient,
			f.tektonClient,
			f.logger,
			f.namespaceMgr,
			f.pipelineMgr,
			f.templateEngine,
			f.service,
			f.configRepo,
			f.buildRepo,
			f.clientProvider,
			f.registryAuth,
			f.repositoryAuth,
		)
	}
	return f.executor, nil
}

// GetSupportedMethods returns a list of supported build methods
func (f *TektonExecutorFactory) GetSupportedMethods() []BuildMethod {
	return []BuildMethod{
		BuildMethodPacker,
		BuildMethodBuildx,
		BuildMethodKaniko,
		BuildMethodDocker,
	}
}

// LocalExecutorFactory creates local executors for different build methods
type LocalExecutorFactory struct {
	service BuildExecutionService
}

// NewBuildMethodExecutorFactory creates a local executor factory
func NewBuildMethodExecutorFactory(service BuildExecutionService) BuildMethodExecutorFactory {
	return &LocalExecutorFactory{
		service: service,
	}
}

// CreateExecutor creates an executor for the given build method
func (f *LocalExecutorFactory) CreateExecutor(method BuildMethod) (BuildMethodExecutor, error) {
	switch method {
	case BuildMethodPacker:
		return NewMethodPackerExecutor(f.service), nil
	case BuildMethodBuildx:
		return NewMethodBuildxExecutor(f.service), nil
	case BuildMethodKaniko:
		return NewMethodKanikoExecutor(f.service), nil
	case BuildMethodDocker:
		return NewMethodDockerExecutor(f.service), nil
	case BuildMethodPaketo:
		return NewMethodPaketoExecutor(f.service), nil
	case BuildMethodNix:
		return NewMethodNixExecutor(f.service), nil
	default:
		return nil, fmt.Errorf("unsupported build method: %s", method)
	}
}

// GetSupportedMethods returns a list of supported build methods
func (f *LocalExecutorFactory) GetSupportedMethods() []BuildMethod {
	return []BuildMethod{
		BuildMethodPacker,
		BuildMethodBuildx,
		BuildMethodKaniko,
		BuildMethodDocker,
		BuildMethodPaketo,
		BuildMethodNix,
	}
}
