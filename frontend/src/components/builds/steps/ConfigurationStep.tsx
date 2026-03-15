import { buildService } from '@/services/buildService'
import { infrastructureService } from '@/services/infrastructureService'
import { projectService } from '@/services/projectService'
import { registryAuthClient } from '@/api/registryAuthClient'
import HelpTooltip from '@/components/common/HelpTooltip'
import { BuildConfig, BuildContextSuggestionsResponse, InfrastructureProvider, InfrastructureRecommendation, InfrastructureType, ProjectBuildSettings, ProjectSource, WizardState } from '@/types'
import React, { useEffect, useMemo, useState } from 'react'
import { Link } from 'react-router-dom'
import InfrastructureSelector from '../InfrastructureSelector'
import BuildxConfigForm from './BuildxConfigForm'
import DockerConfigForm from './DockerConfigForm'
import KanikoConfigForm from './KanikoConfigForm'
import NixConfigForm from './NixConfigForm'
import PackerConfigForm from './PackerConfigForm'
import PaketoConfigForm from './PaketoConfigForm'

interface ConfigurationStepProps {
    wizardState: WizardState
    onUpdate: (updates: Partial<WizardState>) => void
}

const ConfigurationStep: React.FC<ConfigurationStepProps> = ({
    wizardState,
    onUpdate
}) => {
    const [activeTab, setActiveTab] = useState<'method' | 'registry' | 'infrastructure' | 'common'>('method')
    const [infrastructureRecommendation, setInfrastructureRecommendation] = useState<InfrastructureRecommendation | null>(null)
    const [recommendationLoading, setRecommendationLoading] = useState(false)
    const [recommendationError, setRecommendationError] = useState<string | null>(null)
    const [lastRecommendationHash, setLastRecommendationHash] = useState<string | null>(null)
    const [registryAuthOptions, setRegistryAuthOptions] = useState<Array<{ id: string; label: string }>>([])
    const [registryAuthLoading, setRegistryAuthLoading] = useState(false)
    const [registryAuthError, setRegistryAuthError] = useState<string | null>(null)
    const [contextSuggestions, setContextSuggestions] = useState<BuildContextSuggestionsResponse | null>(null)
    const [contextSuggestionsLoading, setContextSuggestionsLoading] = useState(false)
    const [contextSuggestionsError, setContextSuggestionsError] = useState<string | null>(null)
    const [contextGuardWarning, setContextGuardWarning] = useState<string | null>(null)
    const [showContextSuggestions, setShowContextSuggestions] = useState(false)
    const [selectedSuggestedContext, setSelectedSuggestedContext] = useState<string>('.')
    const [selectedSuggestedDockerfile, setSelectedSuggestedDockerfile] = useState<string>('')
    const [availableProviders, setAvailableProviders] = useState<InfrastructureProvider[]>([])
    const [projectSources, setProjectSources] = useState<ProjectSource[]>([])
    const [projectSourcesLoading, setProjectSourcesLoading] = useState(false)
    const [projectBuildSettings, setProjectBuildSettings] = useState<ProjectBuildSettings | null>(null)
    const selectedInfrastructure = wizardState.infrastructureType ?? null
    const selectedProviderId = wizardState.infrastructureProviderId ?? null
    const supportsContextSuggestions = wizardState.buildMethod === 'container'
        || wizardState.buildMethod === 'buildx'
        || wizardState.buildMethod === 'kaniko'
        || wizardState.buildMethod === 'paketo'
    const supportsDockerfileSuggestions = wizardState.buildMethod === 'container'
        || wizardState.buildMethod === 'buildx'
        || wizardState.buildMethod === 'kaniko'

    const updateBuildConfig = (updates: Partial<BuildConfig>) => {
        onUpdate({
            buildConfig: {
                ...wizardState.buildConfig,
                ...updates
            }
        })
    }

    useEffect(() => {
        const projectId = wizardState.selectedProject?.id
        if (!projectId) {
            setProjectSources([])
            return
        }
        let mounted = true
        const run = async () => {
            try {
                setProjectSourcesLoading(true)
                const sources = await projectService.listProjectSources(projectId)
                if (!mounted) return
                setProjectSources(sources)
                if (!wizardState.buildConfig?.sourceId) {
                    const defaultSource = sources.find((s) => s.isDefault && s.isActive) || sources.find((s) => s.isActive)
                    if (defaultSource) {
                        updateBuildConfig({ sourceId: defaultSource.id })
                    }
                }
            } catch {
                if (!mounted) return
                setProjectSources([])
            } finally {
                if (mounted) setProjectSourcesLoading(false)
            }
        }
        run()
        return () => {
            mounted = false
        }
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [wizardState.selectedProject?.id])

    useEffect(() => {
        const projectId = wizardState.selectedProject?.id
        if (!projectId) {
            setProjectBuildSettings(null)
            return
        }
        let mounted = true
        const run = async () => {
            try {
                const settings = await projectService.getProjectBuildSettings(projectId)
                if (!mounted) return
                setProjectBuildSettings(settings)
            } catch {
                if (!mounted) return
                setProjectBuildSettings(null)
            }
        }
        run()
        return () => {
            mounted = false
        }
    }, [wizardState.selectedProject?.id])

    // Handlers for form components
    const handleDockerConfig = (config: any) => {
        updateBuildConfig({
            dockerfile: config.dockerfile,
            buildContext: config.build_context,
            registryRepo: config.registry_repo,
            target: config.target_stage,
            buildArgs: config.build_args,
            environment: config.environment_vars
        })
    }

    const handleBuildxConfig = (config: any) => {
        updateBuildConfig({
            dockerfile: config.dockerfile,
            buildContext: config.build_context,
            registryRepo: config.registry_repo,
            platforms: config.platforms,
            target: config.target,
            buildArgs: config.build_args,
            cache: !config.no_cache && !!(config.cache?.from || config.cache?.to),
            cacheRepo: config.cache?.to,
            cacheFrom: config.cache_from,
            secrets: config.secrets
        })
    }

    const handleKanikoConfig = (config: any) => {
        updateBuildConfig({
            dockerfile: config.dockerfile,
            buildContext: config.build_context,
            target: config.target,
            buildArgs: config.build_args,
            cache: !!config.cache_repo,
            cacheRepo: config.cache_repo,
            registryRepo: config.registry_repo,
            skipUnusedStages: config.skip_unused_stages
        })
    }

    const handleTestRegistryAuth = async (registryAuthId: string, registryRepo: string): Promise<{ success: boolean; message: string }> => {
        try {
            const response = await registryAuthClient.testRegistryAuthPermissions(registryAuthId, {
                registry_repo: registryRepo,
            })
            return {
                success: response.success,
                message: response.message,
            }
        } catch (error) {
            const message = error instanceof Error ? error.message : 'Failed to test registry authentication'
            return {
                success: false,
                message,
            }
        }
    }

    const handleNixConfig = (config: any) => {
        updateBuildConfig({
            nixExpression: config.nix_expression,
            flakeUri: config.flake_uri,
            nixAttributes: config.attributes,
            nixOutputs: config.outputs,
            nixCacheDir: config.cache_dir,
            nixPure: config.pure,
            nixShowTrace: config.show_trace
        })
    }

    const requiresRegistryAuth = wizardState.buildMethod === 'container'
        || wizardState.buildMethod === 'buildx'
        || wizardState.buildMethod === 'kaniko'
        || wizardState.buildMethod === 'paketo'
    const showRegistryTab = requiresRegistryAuth
        && wizardState.buildMethod !== 'kaniko'
        && wizardState.buildMethod !== 'buildx'

    const handlePackerConfig = (config: any) => {
        updateBuildConfig({
            packerTemplate: config.template,
            variables: config.variables
        })
    }

    const handlePaketoConfig = (config: any) => {
        updateBuildConfig({
            paketoConfig: {
                builder: config.builder,
                buildpacks: config.buildpacks,
                env: config.env,
                buildArgs: config.build_args,
            }
        })
    }

    const currentConfigHash = useMemo(
        () => JSON.stringify({
            buildMethod: wizardState.buildMethod,
            buildConfig: wizardState.buildConfig,
            projectId: wizardState.selectedProject?.id,
        }),
        [wizardState.buildMethod, wizardState.buildConfig, wizardState.selectedProject]
    )

    const fetchInfrastructureRecommendation = async () => {
        if (!wizardState.buildMethod || !wizardState.selectedProject) return

        try {
            setRecommendationLoading(true)
            setRecommendationError(null)
            const recommendation = await buildService.getInfrastructureRecommendation({
                build_method: wizardState.buildMethod,
                project_id: wizardState.selectedProject.id,
                config: wizardState.buildConfig
            })
            setInfrastructureRecommendation(recommendation)
            setLastRecommendationHash(currentConfigHash)
            // Auto-select the recommended infrastructure if none selected
            if (!selectedInfrastructure) {
                onUpdate({ infrastructureType: recommendation.recommended_infrastructure })
            }
        } catch (error) {
            console.error('Failed to fetch infrastructure recommendation:', error)
            setRecommendationError('Failed to fetch infrastructure recommendation')
        } finally {
            setRecommendationLoading(false)
        }
    }

    // Fetch once when user opens the infrastructure tab and there is no recommendation yet
    useEffect(() => {
        if (activeTab !== 'infrastructure') return
        if (infrastructureRecommendation) return
        fetchInfrastructureRecommendation()
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [activeTab])

    // Also prefetch recommendation in background so users can proceed
    // without needing to open the Infrastructure tab first.
    useEffect(() => {
        if (!wizardState.buildMethod || !wizardState.selectedProject?.id) return
        if (infrastructureRecommendation || recommendationLoading) return
        fetchInfrastructureRecommendation()
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [wizardState.buildMethod, wizardState.selectedProject?.id, infrastructureRecommendation, recommendationLoading])

    // Load available providers once for auto-selection logic.
    useEffect(() => {
        let mounted = true
        const run = async () => {
            try {
                const providers = await infrastructureService.getAvailableOptions()
                if (!mounted) return
                setAvailableProviders(providers)
            } catch {
                if (!mounted) return
                setAvailableProviders([])
            }
        }
        run()
        return () => {
            mounted = false
        }
    }, [])

    // Keep provider selection in sync even when Infrastructure tab hasn't been visited.
    useEffect(() => {
        if (!selectedInfrastructure) return
        const isBuildReadyProvider = (provider: any) =>
            provider.provider_type === selectedInfrastructure &&
            provider.status === 'online' &&
            provider.is_schedulable &&
            provider.config?.tekton_enabled === true
        const providersForType = availableProviders.filter(
            provider => isBuildReadyProvider(provider)
        )

        if (providersForType.length === 0) {
            if (selectedProviderId !== null) {
                onUpdate({ infrastructureProviderId: null })
            }
            return
        }

        const selectedExists = providersForType.some(provider => provider.id === selectedProviderId)
        const nextProviderId = selectedExists ? selectedProviderId : providersForType[0].id
        if (nextProviderId !== selectedProviderId) {
            onUpdate({ infrastructureProviderId: nextProviderId })
        }
    }, [availableProviders, onUpdate, selectedInfrastructure, selectedProviderId])

    useEffect(() => {
        const projectId = wizardState.selectedProject?.id
        if (!projectId || !requiresRegistryAuth) {
            setRegistryAuthOptions([])
            setRegistryAuthError(null)
            return
        }
        let mounted = true
        const run = async () => {
            setRegistryAuthLoading(true)
            setRegistryAuthError(null)
            try {
                const response = await registryAuthClient.listRegistryAuth(projectId, true)
                if (!mounted) return
                const options = (response.registry_auth || []).map((auth) => ({
                    id: auth.id,
                    label: `${auth.name} (${auth.scope}) — ${auth.id}`,
                }))
                setRegistryAuthOptions(options)
            } catch (error) {
                if (!mounted) return
                setRegistryAuthError('Failed to load registry authentications')
            } finally {
                if (mounted) setRegistryAuthLoading(false)
            }
        }
        run()
        return () => {
            mounted = false
        }
    }, [wizardState.selectedProject?.id, requiresRegistryAuth])

    useEffect(() => {
        if (showRegistryTab && wizardState.validationErrors?.registryAuth && activeTab !== 'registry') {
            setActiveTab('registry')
        }
    }, [wizardState.validationErrors?.registryAuth, activeTab, showRegistryTab])

    useEffect(() => {
        if (activeTab === 'registry' && !showRegistryTab) {
            setActiveTab('method')
        }
    }, [activeTab, showRegistryTab])

    useEffect(() => {
        const projectId = wizardState.selectedProject?.id
        const branch = wizardState.selectedProject?.branch
        if (!projectId || !supportsContextSuggestions) {
            setContextSuggestions(null)
            setContextSuggestionsError(null)
            setContextSuggestionsLoading(false)
            setShowContextSuggestions(false)
            return
        }
        if (!showContextSuggestions) {
            return
        }

        let mounted = true
        const run = async () => {
            setContextSuggestionsLoading(true)
            setContextSuggestionsError(null)
            try {
                const response = await buildService.getBuildContextSuggestions(projectId, branch || undefined)
                if (!mounted) return
                setContextSuggestions(response)
            } catch (error) {
                if (!mounted) return
                setContextSuggestions(null)
                setContextSuggestionsError('Unable to inspect repository structure. You can still enter context and Dockerfile path manually.')
            } finally {
                if (mounted) setContextSuggestionsLoading(false)
            }
        }
        run()
        return () => {
            mounted = false
        }
    }, [wizardState.selectedProject?.id, wizardState.selectedProject?.branch, supportsContextSuggestions, showContextSuggestions])

    const handleInfrastructureSelect = (infrastructure: InfrastructureType) => {
        onUpdate({ infrastructureType: infrastructure })
    }

    const handleProviderSelect = (providerId: string | null) => {
        onUpdate({ infrastructureProviderId: providerId })
    }

    const readDockerfilePath = (): string | null => {
        const dockerfile = wizardState.buildConfig?.dockerfile as any
        if (!dockerfile) return null
        if (typeof dockerfile === 'string') return dockerfile
        if (dockerfile.source === 'path') return dockerfile.path || null
        return null
    }

    const normalizePath = (path?: string | null): string => {
        const value = (path || '.').trim().replace(/\\/g, '/').replace(/^.\//, '').replace(/\/+$/, '')
        return value.length > 0 ? value : '.'
    }

    const toPathRelativeToContext = (fullPath: string, context: string): string => {
        if (!fullPath) return fullPath
        if (!context || context === '.') return fullPath
        const prefix = `${context}/`
        if (fullPath === context) return '.'
        if (fullPath.startsWith(prefix)) {
            return fullPath.slice(prefix.length) || '.'
        }
        return fullPath
    }

    const applyTopSuggestedPaths = () => {
        if (!contextSuggestions) return
        const bestContext = contextSuggestions.contexts[0]?.path || '.'
        const bestDockerfile = contextSuggestions.dockerfiles.find((d) => d.context === bestContext) || contextSuggestions.dockerfiles[0]
        updateBuildConfig({
            buildContext: bestContext,
            ...(supportsDockerfileSuggestions && bestDockerfile
                ? { dockerfile: { source: 'path', path: toPathRelativeToContext(bestDockerfile.path, bestContext) } }
                : {}),
        })
        setContextGuardWarning(null)
    }

    const applySelectedSuggestedPaths = () => {
        if (!contextSuggestions) return
        const context = selectedSuggestedContext || contextSuggestions.contexts[0]?.path || '.'
        const dockerfileMatch = contextSuggestions.dockerfiles.find((d) => d.path === selectedSuggestedDockerfile)
        updateBuildConfig({
            buildContext: context,
            ...(supportsDockerfileSuggestions && dockerfileMatch
                ? {
                    dockerfile: {
                        source: 'path',
                        path: toPathRelativeToContext(dockerfileMatch.path, dockerfileMatch.context || context),
                    },
                }
                : {}),
        })
        setContextGuardWarning(null)
    }

    useEffect(() => {
        if (!contextSuggestions) {
            setSelectedSuggestedContext('.')
            setSelectedSuggestedDockerfile('')
            return
        }
        const suggestedContext = contextSuggestions.contexts[0]?.path || '.'
        const dockerfile = contextSuggestions.dockerfiles.find((d) => d.context === suggestedContext) || contextSuggestions.dockerfiles[0]
        setSelectedSuggestedContext(suggestedContext)
        setSelectedSuggestedDockerfile(dockerfile?.path || '')
    }, [contextSuggestions])

    useEffect(() => {
        if (!contextSuggestions || !supportsDockerfileSuggestions) return
        const forContext = contextSuggestions.dockerfiles.filter((d) => (d.context || '.') === selectedSuggestedContext)
        if (forContext.length === 0) return
        const exists = forContext.some((d) => d.path === selectedSuggestedDockerfile)
        if (!exists) {
            setSelectedSuggestedDockerfile(forContext[0].path)
        }
    }, [contextSuggestions, selectedSuggestedContext, selectedSuggestedDockerfile, supportsDockerfileSuggestions])

    useEffect(() => {
        if (!supportsContextSuggestions || !contextSuggestions) {
            setContextGuardWarning(null)
            return
        }

        const currentContext = normalizePath(wizardState.buildConfig?.buildContext || '.')
        const availableContexts = new Set(contextSuggestions.contexts.map((c) => normalizePath(c.path)))
        if (!availableContexts.has(currentContext)) {
            setContextGuardWarning(
                `Current Build Context "${currentContext}" was not found in repository suggestions for this project/ref.`
            )
            return
        }

        if (supportsDockerfileSuggestions) {
            const dockerfilePath = readDockerfilePath()
            if (dockerfilePath) {
                const expectedRelativePaths = new Set(
                    contextSuggestions.dockerfiles
                        .filter((d) => normalizePath(d.context) === currentContext)
                        .map((d) => normalizePath(toPathRelativeToContext(d.path, d.context)))
                )
                if (expectedRelativePaths.size > 0 && !expectedRelativePaths.has(normalizePath(dockerfilePath))) {
                    setContextGuardWarning(
                        `Dockerfile path "${dockerfilePath}" does not match suggested paths for context "${currentContext}".`
                    )
                    return
                }
            }
        }

        setContextGuardWarning(null)
    }, [contextSuggestions, supportsContextSuggestions, supportsDockerfileSuggestions, wizardState.buildConfig?.buildContext, wizardState.buildConfig?.dockerfile])

    // Map BuildType to form component
    const getFormComponent = (buildMethod: string) => {
        switch (buildMethod) {
            case 'packer':
                return (
                    <PackerConfigForm
                        onChange={handlePackerConfig}
                        isLoading={false}
                    />
                )
            case 'buildx':
                return (
                    <BuildxConfigForm
                        onChange={handleBuildxConfig}
                        isLoading={false}
                        registryAuthOptions={registryAuthOptions}
                        registryAuthLoading={registryAuthLoading}
                        registryAuthError={registryAuthError}
                        registryAuthValidationError={wizardState.validationErrors?.registryAuth}
                        selectedRegistryAuthId={wizardState.buildConfig?.registryAuthId}
                        onRegistryAuthChange={(registryAuthId) => updateBuildConfig({ registryAuthId })}
                        onTestRegistryAuth={handleTestRegistryAuth}
                        imageTags={wizardState.buildConfig?.tags || []}
                        onImageTagsChange={(tags) => updateBuildConfig({ tags })}
                        initialValue={{
                            dockerfile: typeof wizardState.buildConfig?.dockerfile === 'string'
                                ? { source: 'path', path: wizardState.buildConfig.dockerfile }
                                : wizardState.buildConfig?.dockerfile,
                            build_context: wizardState.buildConfig?.buildContext || '.',
                            registry_repo: wizardState.buildConfig?.registryRepo || '',
                            platforms: wizardState.buildConfig?.platforms,
                            build_args: wizardState.buildConfig?.buildArgs,
                            secrets: wizardState.buildConfig?.secrets,
                        }}
                    />
                )
            case 'kaniko':
                return (
                    <KanikoConfigForm
                        onChange={handleKanikoConfig}
                        isLoading={false}
                        registryAuthOptions={registryAuthOptions}
                        registryAuthLoading={registryAuthLoading}
                        registryAuthError={registryAuthError}
                        registryAuthValidationError={wizardState.validationErrors?.registryAuth}
                        selectedRegistryAuthId={wizardState.buildConfig?.registryAuthId}
                        onRegistryAuthChange={(registryAuthId) => updateBuildConfig({ registryAuthId })}
                        onTestRegistryAuth={handleTestRegistryAuth}
                        imageTags={wizardState.buildConfig?.tags || []}
                        onImageTagsChange={(tags) => updateBuildConfig({ tags })}
                        initialValue={{
                            dockerfile: wizardState.buildConfig?.dockerfile,
                            build_context: wizardState.buildConfig?.buildContext || '.',
                            registry_repo: wizardState.buildConfig?.registryRepo || '',
                            cache_repo: wizardState.buildConfig?.cacheRepo,
                            build_args: wizardState.buildConfig?.buildArgs,
                            skip_unused_stages: wizardState.buildConfig?.skipUnusedStages,
                        }}
                    />
                )
            case 'container':
            case 'docker':
                return (
                    <DockerConfigForm
                        buildId=""
                        onChange={handleDockerConfig}
                        isLoading={false}
                        initialValue={{
                            dockerfile: wizardState.buildConfig?.dockerfile as any,
                            build_context: wizardState.buildConfig?.buildContext || '.',
                            registry_repo: wizardState.buildConfig?.registryRepo || '',
                            target_stage: wizardState.buildConfig?.target,
                            build_args: wizardState.buildConfig?.buildArgs,
                            environment_vars: wizardState.buildConfig?.environment,
                        }}
                    />
                )
            case 'paketo':
                return (
                    <PaketoConfigForm
                        onChange={handlePaketoConfig}
                        isLoading={false}
                        initialValue={{
                            builder: wizardState.buildConfig?.paketoConfig?.builder,
                            buildpacks: wizardState.buildConfig?.paketoConfig?.buildpacks,
                            env: wizardState.buildConfig?.paketoConfig?.env,
                            build_args: wizardState.buildConfig?.paketoConfig?.buildArgs,
                        }}
                    />
                )
            case 'nix':
                return (
                    <NixConfigForm
                        onChange={handleNixConfig}
                        isLoading={false}
                        initialValue={{
                            nix_expression: wizardState.buildConfig?.nixExpression,
                            flake_uri: wizardState.buildConfig?.flakeUri,
                            attributes: wizardState.buildConfig?.nixAttributes || [],
                            outputs: wizardState.buildConfig?.nixOutputs,
                            cache_dir: wizardState.buildConfig?.nixCacheDir,
                            pure: wizardState.buildConfig?.nixPure,
                            show_trace: wizardState.buildConfig?.nixShowTrace,
                        }}
                    />
                )
            default:
                return <div>Unsupported build method</div>
        }
    }

    const renderBuildMethodConfig = () => {
        const buildMethod = wizardState.buildMethod

        if (!buildMethod) return null

        return (
            <div className="space-y-6">
                {supportsContextSuggestions && (
                    <>
                        <div className="flex items-center justify-between rounded-lg border border-indigo-200 bg-indigo-50 px-4 py-3 dark:border-indigo-700 dark:bg-indigo-900/20">
                            <div>
                                <h4 className="text-sm font-semibold text-indigo-900 dark:text-indigo-200">Repository Structure Suggestions</h4>
                                <p className="mt-1 text-xs text-indigo-800 dark:text-indigo-300">
                                    Open this panel to inspect suggested build context and Dockerfile paths.
                                </p>
                                {contextGuardWarning && !showContextSuggestions && (
                                    <p className="mt-1 text-xs text-amber-700 dark:text-amber-300">Configuration warning available. Open suggestions to review.</p>
                                )}
                            </div>
                            <button
                                type="button"
                                onClick={() => setShowContextSuggestions((prev) => !prev)}
                                className="rounded-md border border-indigo-300 bg-white px-3 py-1.5 text-xs font-medium text-indigo-700 hover:bg-indigo-100 dark:border-indigo-600 dark:bg-slate-900 dark:text-indigo-300 dark:hover:bg-indigo-900/30"
                            >
                                {showContextSuggestions ? 'Hide Suggestions' : 'View Suggestions'}
                            </button>
                        </div>
                        {showContextSuggestions && (
                            <div className="rounded-lg border border-indigo-200 bg-indigo-50 p-4 dark:border-indigo-700 dark:bg-indigo-900/20">
                                {supportsDockerfileSuggestions ? (
                                    <p className="text-xs text-indigo-800 dark:text-indigo-300">
                                        Suggestions are derived from repository layout to reduce build context and Dockerfile path errors.
                                    </p>
                                ) : (
                                    <p className="text-xs text-indigo-800 dark:text-indigo-300">
                                        Suggestions are derived from repository layout to help choose the right source root for this build method.
                                    </p>
                                )}
                                {contextSuggestionsLoading && (
                                    <p className="mt-2 text-xs text-indigo-700 dark:text-indigo-300">Inspecting repository structure...</p>
                                )}
                                {contextSuggestionsError && (
                                    <p className="mt-2 text-xs text-amber-700 dark:text-amber-300">{contextSuggestionsError}</p>
                                )}
                                {contextSuggestions?.note && (
                                    <p className="mt-2 text-xs text-amber-700 dark:text-amber-300">{contextSuggestions.note}</p>
                                )}
                                {!contextSuggestionsLoading && contextSuggestions && (
                                    <div className="mt-3 rounded-md border border-indigo-200 bg-white/70 p-3 dark:border-indigo-700 dark:bg-slate-900/40">
                                        <div className="grid grid-cols-1 gap-3 md:grid-cols-2">
                                            <div>
                                                <label className="block text-[11px] font-medium uppercase tracking-wide text-indigo-800 dark:text-indigo-300">
                                                    {supportsDockerfileSuggestions ? 'Build Context' : 'Suggested Source Root'}
                                                </label>
                                                <select
                                                    value={selectedSuggestedContext}
                                                    onChange={(e) => setSelectedSuggestedContext(e.target.value)}
                                                    className="mt-1 block w-full rounded border border-indigo-300 bg-white px-2 py-1.5 text-xs text-indigo-900 focus:border-indigo-500 focus:outline-none dark:border-indigo-600 dark:bg-slate-900 dark:text-indigo-100"
                                                >
                                                    {contextSuggestions.contexts.slice(0, 8).map((context) => (
                                                        <option key={`ctx-${context.path}`} value={context.path}>
                                                            {context.path}
                                                        </option>
                                                    ))}
                                                </select>
                                            </div>
                                            {supportsDockerfileSuggestions && (
                                                <div>
                                                    <label className="block text-[11px] font-medium uppercase tracking-wide text-indigo-800 dark:text-indigo-300">
                                                        Dockerfile Path
                                                    </label>
                                                    <select
                                                        value={selectedSuggestedDockerfile}
                                                        onChange={(e) => setSelectedSuggestedDockerfile(e.target.value)}
                                                        className="mt-1 block w-full rounded border border-indigo-300 bg-white px-2 py-1.5 text-xs text-indigo-900 focus:border-indigo-500 focus:outline-none dark:border-indigo-600 dark:bg-slate-900 dark:text-indigo-100"
                                                    >
                                                        {contextSuggestions.dockerfiles
                                                            .filter((dockerfile) => (dockerfile.context || '.') === selectedSuggestedContext)
                                                            .slice(0, 8)
                                                            .map((dockerfile) => (
                                                                <option key={`df-${dockerfile.path}`} value={dockerfile.path}>
                                                                    {dockerfile.path}
                                                                </option>
                                                            ))}
                                                    </select>
                                                </div>
                                            )}
                                        </div>
                                        <div className="mt-3 flex items-center gap-2">
                                            <button
                                                type="button"
                                                onClick={applySelectedSuggestedPaths}
                                                className="rounded border border-indigo-300 bg-white px-2.5 py-1.5 text-xs font-medium text-indigo-700 hover:bg-indigo-100 dark:border-indigo-600 dark:bg-slate-900 dark:text-indigo-300 dark:hover:bg-indigo-900/30"
                                            >
                                                Use Suggestions
                                            </button>
                                            <button
                                                type="button"
                                                onClick={applyTopSuggestedPaths}
                                                className="rounded border border-indigo-200 bg-indigo-50 px-2.5 py-1.5 text-xs font-medium text-indigo-700 hover:bg-indigo-100 dark:border-indigo-700 dark:bg-indigo-900/30 dark:text-indigo-300 dark:hover:bg-indigo-900/50"
                                            >
                                                Use Best Match
                                            </button>
                                        </div>
                                    </div>
                                )}
                                {contextGuardWarning && (
                                    <div className="mt-2 rounded border border-amber-300 bg-amber-100/70 p-2 dark:border-amber-700 dark:bg-amber-900/30">
                                        <p className="text-xs text-amber-800 dark:text-amber-200">{contextGuardWarning}</p>
                                        <button
                                            type="button"
                                            onClick={applyTopSuggestedPaths}
                                            className="mt-2 rounded border border-amber-400 bg-white px-2 py-1 text-xs font-medium text-amber-900 hover:bg-amber-50 dark:border-amber-600 dark:bg-slate-900 dark:text-amber-200 dark:hover:bg-amber-900/20"
                                        >
                                            Apply suggested paths
                                        </button>
                                    </div>
                                )}
                            </div>
                        )}
                    </>
                )}
                <div className="bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-800 rounded-lg p-4">
                    <h4 className="text-sm font-medium text-blue-800 dark:text-blue-200">
                        {buildMethod.toUpperCase()} Build Configuration
                    </h4>
                    <p className="mt-1 text-sm text-blue-700 dark:text-blue-300">
                        {buildMethod === 'container' && 'Standard Docker build with multi-stage support'}
                        {buildMethod === 'buildx' && 'Advanced Docker build with multi-platform and cache support'}
                        {buildMethod === 'kaniko' && 'Google Kaniko for building in unprivileged environments'}
                        {buildMethod === 'paketo' && 'Paketo Buildpacks for cloud-native applications'}
                        {buildMethod === 'packer' && 'HashiCorp Packer for machine image builds'}
                        {buildMethod === 'nix' && 'Nix package manager for reproducible builds'}
                    </p>
                </div>

                {getFormComponent(buildMethod)}
            </div>
        )
    }

    const renderRegistryAuthTab = () => (
        <div className="space-y-6">
            <div className="rounded-lg border border-amber-200 dark:border-amber-700 bg-amber-50 dark:bg-amber-900/20 p-4">
                <h4 className="text-sm font-semibold text-amber-900 dark:text-amber-100">
                    Registry Authentication
                </h4>
                <p className="mt-1 text-sm text-amber-800 dark:text-amber-200">
                    {requiresRegistryAuth
                        ? `Required for ${wizardState.buildMethod} builds to push or pull images.`
                        : 'Optional for this build method, but recommended when publishing images.'}
                </p>
                <p className="mt-2 text-xs text-amber-800 dark:text-amber-300">
                    Runtime precedence: selected auth, then project default registry auth, then tenant default.
                </p>
                <p className="mt-1 text-xs text-amber-800 dark:text-amber-300">
                    For YAML config, use the UUID shown in the dropdown (`build_config.registry_auth_id`).
                </p>
                <div className="mt-3">
                    <label className="block text-sm font-medium text-slate-700 dark:text-slate-300">
                        <span className="inline-flex items-center gap-2">
                            Select Registry Credential
                            <HelpTooltip text="Registry auth precedence: selected auth, then project default registry auth, then tenant default." />
                        </span>
                    </label>
                    <select
                        value={wizardState.buildConfig?.registryAuthId || ''}
                        onChange={(e) => updateBuildConfig({ registryAuthId: e.target.value || undefined })}
                        className="mt-1 block w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-md shadow-sm focus:outline-none focus:ring-blue-500 focus:border-blue-500 dark:bg-slate-700 dark:text-white"
                        disabled={registryAuthLoading}
                    >
                        <option value="">
                            {registryAuthLoading ? 'Loading registry auth...' : 'Select registry authentication'}
                        </option>
                        {registryAuthOptions.map((option) => (
                            <option key={option.id} value={option.id}>
                                {option.label}
                            </option>
                        ))}
                    </select>
                    {registryAuthError && (
                        <p className="mt-1 text-sm text-red-600 dark:text-red-400">{registryAuthError}</p>
                    )}
                    {!registryAuthLoading && registryAuthOptions.length === 0 && (
                        <p className="mt-1 text-sm text-amber-700 dark:text-amber-300">
                            No registry authentication found for this project/tenant. Create one before creating builds.
                        </p>
                    )}
                </div>
            </div>
        </div>
    )

    const renderSourceBinding = () => (
        <div className="rounded-lg border border-slate-200 bg-slate-50 p-4 dark:border-slate-700 dark:bg-slate-900/40">
            <h4 className="text-sm font-semibold text-slate-800 dark:text-slate-100">Source Binding</h4>
            <p className="mt-1 text-sm text-slate-600 dark:text-slate-300">
                Choose which project source and git ref policy should be used for this build.
            </p>
            <p className="mt-2 text-xs text-slate-600 dark:text-slate-400">
                Repository auth for build-as-code fetch follows source auth first, then project active auth, then anonymous access.
            </p>
            <div className="mt-2 text-xs text-slate-600 dark:text-slate-400">
                YAML validation policy:
                <span className="ml-1 font-medium text-slate-800 dark:text-slate-200">
                    {projectBuildSettings?.buildConfigOnError === 'fallback_to_ui'
                        ? 'Fallback to saved UI config on YAML error'
                        : 'Strict (fail build on YAML error)'}
                </span>
                <span className="ml-2">
                    Change in Project Settings -&gt; Build Settings.
                </span>
            </div>
            {wizardState.selectedProject?.id && (
                <div className="mt-3">
                    <Link
                        to={`/projects/${wizardState.selectedProject.id}?tab=build-settings`}
                        target="_blank"
                        rel="noopener noreferrer"
                        className="inline-flex items-center rounded-md border border-blue-300 dark:border-blue-700 px-3 py-1.5 text-xs font-medium text-blue-700 dark:text-blue-300 bg-white dark:bg-slate-800 hover:bg-blue-50 dark:hover:bg-blue-900/20"
                    >
                        Open Build Settings (new tab)
                    </Link>
                </div>
            )}
            <div className="mt-4 grid grid-cols-1 gap-4 md:grid-cols-3">
                <div>
                    <label className="block text-sm font-medium text-slate-700 dark:text-slate-300">
                        <span className="inline-flex items-center gap-2">
                            Source
                            <HelpTooltip text="Repository auth precedence for build-as-code: selected source auth, then project active auth, then anonymous." />
                        </span>
                    </label>
                    <select
                        value={wizardState.buildConfig?.sourceId || ''}
                        onChange={(e) => updateBuildConfig({ sourceId: e.target.value })}
                        disabled={projectSourcesLoading || !wizardState.selectedProject?.id}
                        className="mt-1 block w-full rounded-md border border-slate-300 bg-white px-3 py-2 text-sm text-slate-900 focus:border-blue-500 focus:outline-none disabled:cursor-not-allowed disabled:bg-slate-100 dark:border-slate-600 dark:bg-slate-800 dark:text-slate-100 dark:focus:border-blue-400 dark:disabled:bg-slate-700"
                    >
                        <option value="">
                            {!wizardState.selectedProject?.id
                                ? 'Select a project first'
                                : (projectSourcesLoading ? 'Loading sources...' : 'Select source')}
                        </option>
                        {projectSources.filter((s) => s.isActive).map((source) => (
                            <option key={source.id} value={source.id}>
                                {source.name} ({source.defaultBranch})
                            </option>
                        ))}
                    </select>
                    {wizardState.validationErrors?.sourceId && (
                        <p className="mt-1 text-sm text-red-600 dark:text-red-400">{wizardState.validationErrors.sourceId}</p>
                    )}
                </div>
                <div>
                    <label className="block text-sm font-medium text-slate-700 dark:text-slate-300">Ref Policy</label>
                    <select
                        value={wizardState.buildConfig?.refPolicy || 'source_default'}
                        onChange={(e) => updateBuildConfig({ refPolicy: e.target.value as BuildConfig['refPolicy'] })}
                        className="mt-1 block w-full rounded-md border border-slate-300 bg-white px-3 py-2 text-sm text-slate-900 focus:border-blue-500 focus:outline-none dark:border-slate-600 dark:bg-slate-800 dark:text-slate-100 dark:focus:border-blue-400"
                    >
                        <option value="source_default">Use source default branch</option>
                        <option value="fixed">Use fixed ref</option>
                        <option value="event_ref">Use webhook event ref</option>
                    </select>
                </div>
                <div>
                    <label className="block text-sm font-medium text-slate-700 dark:text-slate-300">Fixed Ref</label>
                    <input
                        type="text"
                        value={wizardState.buildConfig?.fixedRef || ''}
                        onChange={(e) => updateBuildConfig({ fixedRef: e.target.value })}
                        disabled={(wizardState.buildConfig?.refPolicy || 'source_default') !== 'fixed'}
                        placeholder="main or refs/tags/v1.0.0"
                        className="mt-1 block w-full rounded-md border border-slate-300 bg-white px-3 py-2 text-sm text-slate-900 focus:border-blue-500 focus:outline-none disabled:cursor-not-allowed disabled:bg-slate-100 dark:border-slate-600 dark:bg-slate-800 dark:text-slate-100 dark:focus:border-blue-400 dark:disabled:bg-slate-700"
                    />
                    {wizardState.validationErrors?.fixedRef && (
                        <p className="mt-1 text-sm text-red-600 dark:text-red-400">{wizardState.validationErrors.fixedRef}</p>
                    )}
                </div>
            </div>
        </div>
    )

    const renderCommonSettings = () => (
        <div className="space-y-6">
            {wizardState.buildMethod !== 'kaniko' && wizardState.buildMethod !== 'buildx' && (
                <div>
                    <label className="block text-sm font-medium text-slate-700 dark:text-slate-300">
                        Image Tags
                    </label>
                    <input
                        type="text"
                        value={wizardState.buildConfig?.tags?.join(', ') || ''}
                        onChange={(e) => updateBuildConfig({
                            tags: e.target.value.split(',').map(tag => tag.trim()).filter(tag => tag)
                        })}
                        placeholder="latest, v1.0.0, production"
                        className="mt-1 block w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-md shadow-sm focus:outline-none focus:ring-blue-500 focus:border-blue-500 dark:bg-slate-700 dark:text-white"
                    />
                    <p className="mt-1 text-sm text-slate-500 dark:text-slate-400">
                        Comma-separated list of tags for the built image.
                    </p>
                </div>
            )}

            <div>
                <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-3">
                    Environment Variables
                </label>
                <div className="space-y-2">
                    {Object.entries(wizardState.buildConfig?.environment || {}).map(([key, value], index) => (
                        <div key={index} className="flex space-x-2">
                            <input
                                type="text"
                                placeholder="KEY"
                                value={key}
                                onChange={(e) => {
                                    const newEnv = { ...wizardState.buildConfig?.environment }
                                    delete newEnv[key]
                                    newEnv[e.target.value] = value
                                    updateBuildConfig({ environment: newEnv })
                                }}
                                className="flex-1 px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-md shadow-sm focus:outline-none focus:ring-blue-500 focus:border-blue-500 dark:bg-slate-700 dark:text-white font-mono"
                            />
                            <input
                                type="text"
                                placeholder="VALUE"
                                value={value as string}
                                onChange={(e) => updateBuildConfig({
                                    environment: {
                                        ...wizardState.buildConfig?.environment,
                                        [key]: e.target.value
                                    }
                                })}
                                className="flex-1 px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-md shadow-sm focus:outline-none focus:ring-blue-500 focus:border-blue-500 dark:bg-slate-700 dark:text-white font-mono"
                            />
                            <button
                                onClick={() => {
                                    const newEnv = { ...wizardState.buildConfig?.environment }
                                    delete newEnv[key]
                                    updateBuildConfig({ environment: newEnv })
                                }}
                                className="px-3 py-2 text-red-600 hover:text-red-800 dark:text-red-400 dark:hover:text-red-300"
                            >
                                ✕
                            </button>
                        </div>
                    ))}
                    <button
                        onClick={() => updateBuildConfig({
                            environment: {
                                ...wizardState.buildConfig?.environment,
                                '': ''
                            }
                        })}
                        className="text-blue-600 hover:text-blue-800 dark:text-blue-400 dark:hover:text-blue-300 text-sm font-medium"
                    >
                        + Add Environment Variable
                    </button>
                </div>
                <p className="mt-2 text-sm text-slate-500 dark:text-slate-400">
                    Runtime environment variables (may not apply to all build methods)
                </p>
            </div>
        </div>
    )

    const renderInfrastructureSettings = () => (
        <div className="space-y-6">
            <div className="bg-green-50 dark:bg-green-900/20 border border-green-200 dark:border-green-800 rounded-lg p-4">
                <h4 className="text-sm font-medium text-green-800 dark:text-green-200">
                    Infrastructure Selection
                </h4>
                <p className="mt-1 text-sm text-green-700 dark:text-green-300">
                    Choose the infrastructure where your build will run. Recommendations are based on provider availability and build preferences.
                </p>
            </div>

            {infrastructureRecommendation ? (
                <>
                    <div className="flex items-center justify-between text-sm text-slate-600 dark:text-slate-400">
                        <span>
                            {lastRecommendationHash !== currentConfigHash ? 'Recommendation may be out of date.' : 'Recommendation is up to date.'}
                        </span>
                        <button
                            type="button"
                            onClick={fetchInfrastructureRecommendation}
                            className="inline-flex items-center px-3 py-1.5 text-sm font-medium text-blue-700 dark:text-blue-300 bg-blue-50 dark:bg-blue-900/30 border border-blue-200 dark:border-blue-800 rounded hover:bg-blue-100 dark:hover:bg-blue-900/50 disabled:opacity-50"
                            disabled={recommendationLoading}
                        >
                            {recommendationLoading ? 'Refreshing…' : 'Refresh Recommendation'}
                        </button>
                    </div>
                    <InfrastructureSelector
                        recommendation={infrastructureRecommendation}
                        value={selectedInfrastructure || ''}
                        onChange={handleInfrastructureSelect}
                        selectedProviderId={selectedProviderId}
                        onProviderChange={handleProviderSelect}
                    />
                </>
            ) : (
                <div className="text-center py-8">
                    {recommendationLoading ? (
                        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600 mx-auto"></div>
                    ) : (
                        <div className="rounded-full h-8 w-8 border-2 border-slate-300 dark:border-slate-600 mx-auto"></div>
                    )}
                    <p className="mt-2 text-sm text-slate-600 dark:text-slate-400">
                        {recommendationLoading ? 'Analyzing your build requirements...' : 'No recommendation yet.'}
                    </p>
                    {recommendationError && (
                        <p className="mt-2 text-sm text-red-600 dark:text-red-400">{recommendationError}</p>
                    )}
                    <button
                        type="button"
                        onClick={fetchInfrastructureRecommendation}
                        className="mt-3 inline-flex items-center px-3 py-1.5 text-sm font-medium text-blue-700 dark:text-blue-300 bg-blue-50 dark:bg-blue-900/30 border border-blue-200 dark:border-blue-800 rounded hover:bg-blue-100 dark:hover:bg-blue-900/50 disabled:opacity-50"
                        disabled={recommendationLoading}
                    >
                        {recommendationLoading ? 'Refreshing…' : 'Get Recommendation'}
                    </button>
                </div>
            )}
        </div>
    )

    return (
        <div className="space-y-6">
            <div>
                <h3 className="text-lg font-medium text-slate-900 dark:text-white">
                    Build Configuration
                </h3>
                <p className="text-sm text-slate-600 dark:text-slate-400 mt-1">
                    Configure the specific settings for your {wizardState.buildMethod} build.
                </p>
            </div>

            {/* Tab Navigation */}
            <div className="border-b border-slate-200 dark:border-slate-700">
                <nav className="-mb-px flex space-x-8">
                    {[
                        { id: 'method', label: 'Build Configuration', icon: '🏗️' },
                        ...(showRegistryTab ? [{ id: 'registry', label: 'Registry Auth', icon: '🔐' }] : []),
                        { id: 'infrastructure', label: 'Infrastructure', icon: '🖥️' },
                        { id: 'common', label: 'Additional Settings', icon: '⚙️' }
                    ].map((tab) => (
                        <button
                            key={tab.id}
                            onClick={() => setActiveTab(tab.id as any)}
                            className={`py-2 px-1 border-b-2 font-medium text-sm ${activeTab === tab.id
                                ? 'border-blue-500 text-blue-600 dark:text-blue-400'
                                : 'border-transparent text-slate-500 hover:text-slate-700 hover:border-slate-300 dark:text-slate-400 dark:hover:text-slate-300'
                                }`}
                        >
                            <span className="mr-2">{tab.icon}</span>
                            {tab.label}
                        </button>
                    ))}
                </nav>
            </div>

            {/* Tab Content */}
            <div className="mt-6">
                {activeTab === 'method' && (
                    <>
                        {renderSourceBinding()}
                        {renderBuildMethodConfig()}
                    </>
                )}
                {activeTab === 'registry' && showRegistryTab && renderRegistryAuthTab()}
                {activeTab === 'infrastructure' && renderInfrastructureSettings()}
                {activeTab === 'common' && renderCommonSettings()}
            </div>

            {/* Contextual Help Messages */}
            {(wizardState.buildMethod === 'container') && !wizardState.buildConfig?.dockerfile && (
                <div className="bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-800 rounded-lg p-3">
                    <p className="text-sm text-blue-800 dark:text-blue-200">
                        💡 Configure your Dockerfile, build context, and any build arguments above.
                    </p>
                </div>
            )}

            {wizardState.buildMethod === 'buildx' && (!wizardState.buildConfig?.dockerfile || !wizardState.buildConfig?.platforms?.length) && (
                <div className="bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-800 rounded-lg p-3">
                    <p className="text-sm text-blue-800 dark:text-blue-200">
                        💡 Configure your Dockerfile, target registry/auth, image tags, and platform/caching options above.
                    </p>
                </div>
            )}

            {wizardState.buildMethod === 'kaniko' && !wizardState.buildConfig?.dockerfile && (
                <div className="bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-800 rounded-lg p-3">
                    <p className="text-sm text-blue-800 dark:text-blue-200">
                        💡 Configure your Dockerfile, build context, and registry settings above.
                    </p>
                </div>
            )}

            {wizardState.buildMethod === 'packer' && !wizardState.buildConfig?.packerTemplate && (
                <div className="bg-amber-50 dark:bg-amber-900/20 border border-amber-200 dark:border-amber-800 rounded-lg p-3">
                    <p className="text-sm text-amber-800 dark:text-amber-200">
                        📋 Provide your Packer template JSON configuration above to define your build.
                    </p>
                </div>
            )}

            {wizardState.buildMethod === 'nix' && !wizardState.buildConfig?.nixExpression && !wizardState.buildConfig?.flakeUri && (
                <div className="bg-purple-50 dark:bg-purple-900/20 border border-purple-200 dark:border-purple-800 rounded-lg p-3">
                    <p className="text-sm text-purple-800 dark:text-purple-200">
                        ❄️ Configure your Nix expression or Flake URI, and specify build attributes above.
                    </p>
                </div>
            )}

            {wizardState.buildMethod === 'paketo' && !wizardState.buildConfig?.paketoConfig?.builder && (
                <div className="bg-green-50 dark:bg-green-900/20 border border-green-200 dark:border-green-800 rounded-lg p-3">
                    <p className="text-sm text-green-800 dark:text-green-200">
                        🏗️ Select a buildpack builder and configure your application build settings above.
                    </p>
                </div>
            )}
        </div>
    )
}

export default ConfigurationStep
