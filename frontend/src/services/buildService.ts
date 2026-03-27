import {
    Build,
    BuildExecutionAttempt,
    BuildContextSuggestionsResponse,
    BuildCapabilitiesConfig,
    BuildTraceResponse,
    BuildWorkflowResponse,
    CreateBuildRequest,
    InfrastructureRecommendation,
    InfrastructureRecommendationRequest,
    InfrastructureUsageResponse,
    PaginatedResponse,
    ToolAvailabilityConfig
} from '../types'
import { api } from './api'

export const buildService = {
    // Normalize API build (snake_case) to frontend Build (camelCase)
    normalizeBuild(apiBuild: any): Build {
        if (!apiBuild) {
            throw new Error('Invalid build response')
        }

        const pick = (obj: any, keys: string[]) => {
            if (!obj) return undefined
            for (const key of keys) {
                if (Object.prototype.hasOwnProperty.call(obj, key)) {
                    return obj[key]
                }
            }
            return undefined
        }

        const manifest = apiBuild.manifest || {}
        const buildConfig = manifest.build_config || manifest.buildConfig
        const infrastructureType = pick(manifest, ['infrastructure_type', 'infrastructureType'])
        const infrastructureProviderId = pick(manifest, ['infrastructure_provider_id', 'infrastructureProviderId'])

        const sourceId = pick(manifest.metadata, ['source_id', 'sourceId'])
        const refPolicy = pick(manifest.metadata, ['ref_policy', 'refPolicy'])
        const fixedRef = pick(manifest.metadata, ['fixed_ref', 'fixedRef'])

        const normalizedBuildConfig = (buildConfig || sourceId || refPolicy || fixedRef)
            ? {
                buildType: pick(buildConfig, ['build_type', 'buildType', 'BuildType']),
                sbomTool: pick(buildConfig, ['sbom_tool', 'sbomTool', 'SBOMTool']),
                scanTool: pick(buildConfig, ['scan_tool', 'scanTool', 'ScanTool']),
                registryType: pick(buildConfig, ['registry_type', 'registryType', 'RegistryType']),
                secretManagerType: pick(buildConfig, ['secret_manager_type', 'secretManagerType', 'SecretManagerType']),
                packerTemplate: pick(buildConfig, ['packer_template', 'packerTemplate', 'PackerTemplate']),
                buildVars: pick(buildConfig, ['build_vars', 'buildVars', 'BuildVars']),
                onError: pick(buildConfig, ['on_error', 'onError', 'OnError']),
                parallel: pick(buildConfig, ['parallel', 'Parallel']),
                paketoConfig: pick(buildConfig, ['paketo_config', 'paketoConfig', 'PaketoConfig']),
                nixExpression: pick(buildConfig, ['nix_expression', 'nixExpression', 'NixExpression']),
                flakeUri: pick(buildConfig, ['flake_uri', 'flakeUri', 'FlakeURI']),
                nixAttributes: pick(buildConfig, ['attributes', 'nix_attributes', 'Attributes']),
                nixOutputs: pick(buildConfig, ['outputs', 'nix_outputs', 'Outputs']),
                nixCacheDir: pick(buildConfig, ['cache_dir', 'nix_cache_dir', 'CacheDir']),
                nixPure: pick(buildConfig, ['pure', 'nix_pure', 'Pure']),
                nixShowTrace: pick(buildConfig, ['show_trace', 'nix_show_trace', 'ShowTrace']),
                dockerfile: pick(buildConfig, ['dockerfile', 'Dockerfile']),
                buildContext: pick(buildConfig, ['build_context', 'buildContext', 'BuildContext']),
                buildArgs: pick(buildConfig, ['build_args', 'buildArgs', 'BuildArgs']),
                target: pick(buildConfig, ['target', 'Target']),
                cache: pick(buildConfig, ['cache', 'Cache']),
                cacheRepo: pick(buildConfig, ['cache_repo', 'cacheRepo', 'CacheRepo']),
                registryRepo: pick(buildConfig, ['registry_repo', 'registryRepo', 'RegistryRepo']),
                registryAuthId: pick(buildConfig, ['registry_auth_id', 'registryAuthId', 'RegistryAuthID']),
                skipUnusedStages: pick(buildConfig, ['skip_unused_stages', 'skipUnusedStages', 'SkipUnusedStages']),
                platforms: pick(buildConfig, ['platforms', 'Platforms']),
                cacheTo: pick(buildConfig, ['cache_to', 'cacheTo', 'CacheTo']),
                cacheFrom: pick(buildConfig, ['cache_from', 'cacheFrom', 'CacheFrom']),
                secrets: pick(buildConfig, ['secrets', 'Secrets']),
                variables: pick(buildConfig, ['variables', 'Variables']),
                builders: pick(buildConfig, ['builders', 'Builders']),
                provisioners: pick(buildConfig, ['provisioners', 'Provisioners']),
                postProcessors: pick(buildConfig, ['post_processors', 'postProcessors', 'PostProcessors']),
                sourceId,
                refPolicy,
                fixedRef,
            }
            : undefined

        const normalizeBuildStatus = (status: string): Build['status'] => {
            switch ((status || '').toLowerCase()) {
                case 'success':
                    return 'completed'
                case 'in_progress':
                    return 'running'
                case 'pending':
                case 'queued':
                case 'running':
                case 'completed':
                case 'failed':
                case 'cancelled':
                    return status as Build['status']
                default:
                    return 'pending'
            }
        }

        return {
            id: apiBuild.id,
            tenantId: apiBuild.tenant_id ?? apiBuild.tenantId,
            projectId: apiBuild.project_id ?? apiBuild.projectId,
            manifest: {
                name: manifest.name,
                type: manifest.type,
                baseImage: manifest.base_image ?? manifest.baseImage ?? '',
                instructions: manifest.instructions ?? [],
                environment: manifest.environment ?? {},
                tags: manifest.tags ?? [],
                metadata: manifest.metadata ?? {},
                infrastructureType,
                infrastructureProviderId,
                vmConfig: manifest.vm_config ?? manifest.vmConfig,
                buildConfig: normalizedBuildConfig
            },
            status: normalizeBuildStatus(apiBuild.status),
            result: apiBuild.result,
            errorMessage: apiBuild.error_message ?? apiBuild.errorMessage,
            createdAt: apiBuild.created_at ?? apiBuild.createdAt,
            startedAt: apiBuild.started_at ?? apiBuild.startedAt,
            completedAt: apiBuild.completed_at ?? apiBuild.completedAt,
            updatedAt: apiBuild.updated_at ?? apiBuild.updatedAt,
            version: apiBuild.version ?? 1
        }
    },

    // Build CRUD operations
    async getBuilds(params?: {
        page?: number
        limit?: number
        status?: string[]
        tenantId?: string
        projectId?: string
        search?: string
    }): Promise<PaginatedResponse<Build>> {
        const response = await api.get('/builds', { params })
        // Transform API response to match PaginatedResponse type
        const builds = (response.data?.builds || []).map((b: any) => buildService.normalizeBuild(b))
        return {
            data: builds,
            pagination: {
                page: Math.floor((response.data?.offset || 0) / (response.data?.limit || 20)) + 1,
                limit: response.data?.limit || 20,
                total: response.data?.total_count || 0,
                totalPages: Math.ceil((response.data?.total_count || 0) / (response.data?.limit || 20))
            }
        }
    },

    async getBuild(id: string): Promise<Build> {
        const response = await api.get(`/builds/${id}`)
        return buildService.normalizeBuild(response.data)
    },

    async getBuildWorkflow(id: string, executionId?: string): Promise<BuildWorkflowResponse> {
        const response = await api.get(`/builds/${id}/workflow`, {
            params: executionId ? { execution_id: executionId } : undefined
        })
        return response.data
    },

    async getBuildTrace(id: string, executionId?: string): Promise<BuildTraceResponse> {
        const response = await api.get(`/builds/${id}/trace`, {
            params: executionId ? { execution_id: executionId } : undefined
        })
        const payload = response.data || {}
        const executions: BuildExecutionAttempt[] = (payload.executions || []).map((e: any) => ({
            id: e.id,
            status: e.status,
            createdAt: e.created_at,
            startedAt: e.started_at,
            completedAt: e.completed_at,
            durationSeconds: e.duration_seconds,
            errorMessage: e.error_message
        }))
        const workflowRaw = payload.workflow || { steps: [] }
        const workflow: BuildWorkflowResponse = {
            instanceId: workflowRaw.instance_id,
            executionId: workflowRaw.execution_id,
            status: workflowRaw.status,
            steps: (workflowRaw.steps || []).map((step: any) => ({
                stepKey: step.step_key,
                status: step.status,
                attempts: step.attempts,
                lastError: step.last_error,
                startedAt: step.started_at,
                completedAt: step.completed_at,
                createdAt: step.created_at,
                updatedAt: step.updated_at,
            })),
        }

        return {
            build: buildService.normalizeBuild(payload.build),
            executions,
            selectedExecutionId: payload.selected_execution_id,
            workflow,
            diagnostics: payload.diagnostics
                ? {
                    repoConfig: payload.diagnostics.repo_config
                        ? {
                            applied: !!payload.diagnostics.repo_config.applied,
                            path: payload.diagnostics.repo_config.path,
                            ref: payload.diagnostics.repo_config.ref,
                            stage: payload.diagnostics.repo_config.stage,
                            error: payload.diagnostics.repo_config.error,
                            errorCode: payload.diagnostics.repo_config.error_code,
                            updatedAt: payload.diagnostics.repo_config.updated_at,
                        }
                        : undefined,
                }
                : undefined,
            runtime: payload.runtime,
            correlation: payload.correlation
                ? {
                    workflowInstanceId: payload.correlation.workflow_instance_id,
                    executionId: payload.correlation.execution_id,
                    activeStepKey: payload.correlation.active_step_key,
                }
                : undefined,
        }
    },

    async exportBuildTrace(id: string, executionId?: string): Promise<void> {
        const response = await api.get(`/builds/${id}/trace/export`, {
            params: executionId ? { execution_id: executionId } : undefined,
            responseType: 'blob',
        })
        const blob = new Blob([response.data], { type: 'application/json' })
        const url = window.URL.createObjectURL(blob)
        const link = document.createElement('a')
        const disposition = response.headers?.['content-disposition'] as string | undefined
        const filenameMatch = disposition?.match(/filename=\"?([^\";]+)\"?/)
        link.href = url
        link.download = filenameMatch?.[1] || `build-trace-${id}.json`
        document.body.appendChild(link)
        link.click()
        document.body.removeChild(link)
        window.URL.revokeObjectURL(url)
    },

    async createBuild(buildData: CreateBuildRequest): Promise<Build> {
        // Transform camelCase to snake_case for API compatibility
        if (!buildData.tenantId) {
            throw new Error('tenant_id is required for build creation')
        }
        if (!buildData.projectId) {
            throw new Error('project_id is required for build creation')
        }

        const manifest = buildData.manifest
        const normalizedBuildConfig = (() => {
            const config = manifest.buildConfig
            if (!config) return undefined
            let dockerfileValue: any = config.dockerfile
            if (dockerfileValue && typeof dockerfileValue === 'object') {
                if (dockerfileValue.source === 'content' || dockerfileValue.source === 'upload') {
                    dockerfileValue = dockerfileValue.content || ''
                } else if (dockerfileValue.source === 'path') {
                    dockerfileValue = dockerfileValue.path || ''
                }
            }

            const preferredTag = (manifest.tags || []).find((tag) => typeof tag === 'string' && tag.trim().length > 0)?.trim()
            const hasExplicitTag = (imageRef?: string): boolean => {
                const ref = (imageRef || '').trim()
                if (!ref) return false
                if (ref.includes('@')) return true
                const lastSlash = ref.lastIndexOf('/')
                const lastColon = ref.lastIndexOf(':')
                return lastColon > lastSlash
            }
            const resolvedRegistryRepo =
                config.buildType === 'kaniko' &&
                    config.registryRepo &&
                    preferredTag &&
                    !hasExplicitTag(config.registryRepo)
                    ? `${config.registryRepo}:${preferredTag}`
                    : config.registryRepo

            return {
                build_type: config.buildType,
                sbom_tool: config.sbomTool,
                scan_tool: config.scanTool,
                registry_type: config.registryType,
                secret_manager_type: config.secretManagerType,
                packer_template: config.packerTemplate,
                build_vars: config.buildVars,
                on_error: config.onError,
                parallel: config.parallel,
                paketo_config: config.paketoConfig,
                nix_expression: config.nixExpression,
                flake_uri: config.flakeUri,
                attributes: config.nixAttributes,
                outputs: config.nixOutputs,
                cache_dir: config.nixCacheDir,
                pure: config.nixPure,
                show_trace: config.nixShowTrace,
                dockerfile: dockerfileValue,
                build_context: config.buildContext,
                build_args: config.buildArgs,
                target: config.target,
                cache: config.cache,
                cache_repo: config.cacheRepo,
                registry_repo: resolvedRegistryRepo,
                registry_auth_id: config.registryAuthId,
                skip_unused_stages: config.skipUnusedStages,
                platforms: config.platforms,
                cache_to: config.cacheTo,
                cache_from: config.cacheFrom,
                secrets: config.secrets,
                variables: config.variables,
                builders: config.builders,
                provisioners: config.provisioners,
                post_processors: config.postProcessors
            }
        })()

        const normalizedMetadata = (() => {
            const metadata: Record<string, any> = { ...(manifest.metadata || {}) }
            const dockerfile = manifest.buildConfig?.dockerfile as any
            if (dockerfile && typeof dockerfile === 'object') {
                if ((dockerfile.source === 'content' || dockerfile.source === 'upload') && dockerfile.content) {
                    metadata.dockerfile_inline = dockerfile.content
                }
                if (dockerfile.path) {
                    metadata.dockerfile_path = dockerfile.path
                }
            }
            if (manifest.buildConfig?.sourceId) {
                metadata.source_id = manifest.buildConfig.sourceId
            }
            if (manifest.buildConfig?.refPolicy) {
                metadata.ref_policy = manifest.buildConfig.refPolicy
            }
            if (manifest.buildConfig?.fixedRef) {
                metadata.fixed_ref = manifest.buildConfig.fixedRef
            }
            return metadata
        })()

        const transformedManifest = {
            name: manifest.name,
            type: manifest.type,
            base_image: manifest.baseImage,
            instructions: manifest.instructions,
            environment: manifest.environment,
            tags: manifest.tags,
            metadata: normalizedMetadata,
            infrastructure_type: (manifest as any).infrastructure_type ?? (manifest as any).infrastructureType,
            infrastructure_provider_id: (manifest as any).infrastructure_provider_id ?? (manifest as any).infrastructureProviderId,
            vm_config: manifest.vmConfig,
            build_config: normalizedBuildConfig
        }

        const transformedData = {
            tenant_id: buildData.tenantId,
            project_id: buildData.projectId,
            manifest: transformedManifest
        }
        const response = await api.post('/builds', transformedData)
        return response.data
    },

    async startBuild(id: string): Promise<void> {
        await api.post(`/builds/${id}/start`)
    },

    async cancelBuild(id: string): Promise<void> {
        await api.post(`/builds/${id}/cancel`)
    },

    async retryBuild(id: string): Promise<{ status: string; build_id?: string; source_build_id?: string; new_build_id?: string }> {
        const response = await api.post(`/builds/${id}/retry`)
        return response.data
    },

    async cloneBuild(id: string): Promise<{ status: string; source_build_id: string; new_build_id: string }> {
        const response = await api.post(`/builds/${id}/clone`)
        return response.data
    },

    async getBuildContextSuggestions(projectId: string, ref?: string): Promise<BuildContextSuggestionsResponse> {
        const response = await api.get(`/projects/${projectId}/build-context-suggestions`, {
            params: ref ? { ref } : undefined
        })
        return response.data
    },

    async deleteBuild(id: string): Promise<void> {
        await api.delete(`/builds/${id}`)
    },

    // Administrative tool availability management
    async getToolAvailability(options?: { tenantId?: string; globalDefault?: boolean }): Promise<ToolAvailabilityConfig> {
        const params: Record<string, string | boolean> = {}
        if (options?.globalDefault) {
            params.all_tenants = true
        } else if (options?.tenantId) {
            params.tenant_id = options.tenantId
        }
        const response = await api.get('/admin/settings/tools', { params })
        return response.data
    },

    // Tool availability for build wizard (tenant-scoped, non-admin)
    async getBuildToolAvailability(): Promise<ToolAvailabilityConfig> {
        const response = await api.get('/settings/tools')
        return response.data
    },

    async updateToolAvailability(
        config: ToolAvailabilityConfig,
        options?: { tenantId?: string; globalDefault?: boolean }
    ): Promise<ToolAvailabilityConfig> {
        const params: Record<string, string | boolean> = {}
        if (options?.globalDefault) {
            params.all_tenants = true
        } else if (options?.tenantId) {
            params.tenant_id = options.tenantId
        }
        const response = await api.put('/admin/settings/tools', config, { params })
        return response.data
    },

    async getBuildCapabilities(options?: { tenantId?: string; globalDefault?: boolean }): Promise<BuildCapabilitiesConfig> {
        const params: Record<string, string | boolean> = {}
        if (options?.globalDefault) {
            params.all_tenants = true
        } else if (options?.tenantId) {
            params.tenant_id = options.tenantId
        }
        const response = await api.get('/admin/settings/build-capabilities', { params })
        return response.data
    },

    async updateBuildCapabilities(
        config: BuildCapabilitiesConfig,
        options?: { tenantId?: string; globalDefault?: boolean }
    ): Promise<BuildCapabilitiesConfig> {
        const params: Record<string, string | boolean> = {}
        if (options?.globalDefault) {
            params.all_tenants = true
        } else if (options?.tenantId) {
            params.tenant_id = options.tenantId
        }
        const response = await api.put('/admin/settings/build-capabilities', config, { params })
        return response.data
    },

    // Build artifacts and results
    async getBuildExecutions(id: string, params?: { limit?: number; offset?: number }): Promise<{ executions: BuildExecutionAttempt[] }> {
        const response = await api.get(`/builds/${id}/executions`, { params })
        const executions: BuildExecutionAttempt[] = (response.data?.executions || []).map((e: any) => ({
            id: e.id,
            status: e.status,
            createdAt: e.created_at,
            startedAt: e.started_at,
            completedAt: e.completed_at,
            durationSeconds: e.duration_seconds,
            errorMessage: e.error_message
        }))
        return { executions }
    },

    async getBuildLogs(id: string, executionId?: string): Promise<{ executionId?: string; lines: string[] }> {
        const response = await api.get(`/builds/${id}/logs`, {
            params: executionId ? { execution_id: executionId, limit: 500 } : { limit: 500 }
        })
        const logs = (response.data?.logs || []).map((entry: any) => {
            const timestamp = entry.timestamp ? new Date(entry.timestamp).toISOString() : ''
            const level = entry.level ? String(entry.level).toUpperCase() : 'INFO'
            const message = entry.message || ''
            return timestamp ? `[${timestamp}] [${level}] ${message}` : `[${level}] ${message}`
        })
        return {
            executionId: response.data?.execution_id,
            lines: logs
        }
    },

    // New: return structured log entries including metadata (used by BuildDetailPage)
    async getBuildLogEntries(
        id: string,
        executionId?: string,
        options?: { source?: 'all' | 'tekton' | 'lifecycle'; minLevel?: 'debug' | 'info' | 'warn' | 'error' }
    ): Promise<{ executionId?: string; entries: Array<{ timestamp?: string; level?: string; message?: string; metadata?: Record<string, any> }>; }> {
        const params: Record<string, any> = { limit: 500 }
        if (executionId) params.execution_id = executionId
        if (options?.source) params.source = options.source
        if (options?.minLevel) params.min_level = options.minLevel
        const response = await api.get(`/builds/${id}/logs`, { params })
        const entries = (response.data?.logs || []).map((entry: any) => ({
            timestamp: entry.timestamp || undefined,
            level: entry.level ? String(entry.level).toUpperCase() : undefined,
            message: entry.message || '',
            metadata: entry.metadata || undefined,
        }))
        return {
            executionId: response.data?.execution_id,
            entries,
        }
    },

    async getBuildScanResults(id: string): Promise<any> {
        const response = await api.get(`/builds/${id}/scan-results`)
        return response.data.data
    },

    async getBuildSBOM(id: string): Promise<any> {
        const response = await api.get(`/builds/${id}/sbom`)
        return response.data.data
    },

    async downloadBuildSBOM(id: string): Promise<Blob> {
        const response = await api.get(`/builds/${id}/sbom/download`, {
            responseType: 'blob'
        })
        return response.data
    },

    // Admin build analytics endpoints
    async getAnalytics(): Promise<any> {
        const response = await api.get('/admin/builds/analytics', {
            params: { all_tenants: true }
        })
        return response.data
    },

    async getPerformance(): Promise<any> {
        const response = await api.get('/admin/builds/performance', {
            params: { all_tenants: true }
        })
        return response.data
    },

    async getFailures(): Promise<any> {
        const response = await api.get('/admin/builds/failures', {
            params: { all_tenants: true }
        })
        return response.data
    },

    // Infrastructure recommendation and monitoring (Phase 3)
    async getInfrastructureRecommendation(request: InfrastructureRecommendationRequest): Promise<InfrastructureRecommendation> {
        const response = await api.post('/builds/infrastructure-recommendation', request)
        return response.data
    },

    async getInfrastructureUsage(params?: { range?: string }): Promise<InfrastructureUsageResponse> {
        const response = await api.get('/admin/infrastructure/usage', { params })
        return response.data
    }
}
