import {
    BuildConfiguration,
    CreateProjectSourceRequest,
    CreateBuildConfigurationRequest,
    CreateProjectRequest,
    PaginatedResponse,
    Project,
    ProjectBuildSettings,
    ProjectSource,
    UpdateProjectSourceRequest,
    UpdateBuildConfigurationRequest,
    UpdateProjectRequest
} from '../types'
import { api } from './api'

export interface ProjectWebhookReceipt {
    id: string
    provider: string
    deliveryId?: string
    eventType: string
    eventRef?: string
    eventBranch?: string
    eventCommitSha?: string
    repoUrl?: string
    eventSha?: string
    signatureValid: boolean
    status: string
    reason?: string
    matchedTriggerCount: number
    triggeredBuildIds: string[]
    receivedAt: string
}

export const projectService = {
    normalizeProject(project: any): Project {
        return {
            ...project,
            tenantId: project.tenantId || project.tenant_id || '',
            slug: project.slug || '',
            repositoryUrl: project.repositoryUrl || project.git_repo || '',
            branch: project.branch || project.git_branch || '',
            gitProvider: project.gitProvider || project.git_provider || '',
            repositoryAuthId: project.repositoryAuthId || project.repository_auth_id || '',
            isDraft: project.isDraft ?? project.is_draft ?? false,
            buildCount: project.buildCount ?? project.build_count ?? 0,
            lastBuildAt: project.lastBuildAt ?? project.last_build_at,
            createdAt: project.createdAt || project.created_at || '',
            updatedAt: project.updatedAt || project.updated_at || '',
        }
    },

    normalizeProjectSource(source: any): ProjectSource {
        return {
            id: source.id,
            projectId: source.projectId || source.project_id || '',
            tenantId: source.tenantId || source.tenant_id || '',
            name: source.name || '',
            provider: source.provider || 'generic',
            repositoryUrl: source.repositoryUrl || source.repository_url || '',
            defaultBranch: source.defaultBranch || source.default_branch || 'main',
            repositoryAuthId: source.repositoryAuthId || source.repository_auth_id || '',
            isDefault: source.isDefault ?? source.is_default ?? false,
            isActive: source.isActive ?? source.is_active ?? true,
            createdAt: source.createdAt || source.created_at || '',
            updatedAt: source.updatedAt || source.updated_at || '',
        }
    },

    normalizeWebhookReceipt(receipt: any): ProjectWebhookReceipt {
        return {
            id: receipt.id,
            provider: receipt.provider || 'unknown',
            deliveryId: receipt.deliveryId || receipt.delivery_id || undefined,
            eventType: receipt.eventType || receipt.event_type || '',
            eventRef: receipt.eventRef || receipt.event_ref || undefined,
            eventBranch: receipt.eventBranch || receipt.event_branch || undefined,
            eventCommitSha: receipt.eventCommitSha || receipt.event_commit_sha || undefined,
            repoUrl: receipt.repoUrl || receipt.repo_url || undefined,
            eventSha: receipt.eventSha || receipt.event_sha || undefined,
            signatureValid: receipt.signatureValid ?? receipt.signature_valid ?? false,
            status: receipt.status || '',
            reason: receipt.reason || undefined,
            matchedTriggerCount: receipt.matchedTriggerCount ?? receipt.matched_trigger_count ?? 0,
            triggeredBuildIds: receipt.triggeredBuildIds || receipt.triggered_build_ids || [],
            receivedAt: receipt.receivedAt || receipt.received_at || '',
        }
    },

    normalizeProjectBuildSettings(settings: any): ProjectBuildSettings {
        return {
            projectId: settings.projectId || settings.project_id || '',
            buildConfigMode: settings.buildConfigMode || settings.build_config_mode || 'repo_managed',
            buildConfigFile: settings.buildConfigFile || settings.build_config_file || 'image-factory.yaml',
            buildConfigOnError: settings.buildConfigOnError || settings.build_config_on_error || 'strict',
            updatedAt: settings.updatedAt || settings.updated_at || undefined,
        }
    },

    // Project CRUD operations
    async getProjects(params?: {
        page?: number
        limit?: number
        status?: string[]
        tenantId?: string
        search?: string
    }): Promise<PaginatedResponse<Project>> {
        // Map tenantId to tenant_id for API call
        const apiParams: any = {}
        if (params) {
            Object.keys(params).forEach(key => {
                if (key === 'tenantId') {
                    apiParams.tenant_id = params.tenantId
                } else {
                    apiParams[key] = (params as any)[key]
                }
            })
        }
        const response = await api.get('/projects', { params: apiParams })
        // Transform API response to match PaginatedResponse type
        const projects = (response.data?.projects || []).map((project: any) => projectService.normalizeProject(project))
        return {
            data: projects,
            pagination: {
                page: Math.floor((response.data?.offset || 0) / (response.data?.limit || 20)) + 1,
                limit: response.data?.limit || 20,
                total: response.data?.total_count || 0,
                totalPages: Math.ceil((response.data?.total_count || 0) / (response.data?.limit || 20))
            }
        }
    },

    async getProject(id: string): Promise<Project> {
        const response = await api.get(`/projects/${id}`)
        const projectData = response.data.data
        // Map backend field names to frontend field names
        return projectService.normalizeProject(projectData)
    },

    async createProject(tenantId: string, projectData: Omit<CreateProjectRequest, 'tenantId'>): Promise<Project> {
        // Map frontend field names to backend API field names
        const apiData = {
            tenant_id: tenantId,
            name: projectData.name,
            slug: projectData.slug,
            description: projectData.description,
            git_repo: projectData.repositoryUrl,
            git_branch: projectData.branch,
            git_provider: projectData.gitProvider,
            repository_auth_id: projectData.repositoryAuthId,
            visibility: projectData.visibility || 'private',
            is_draft: projectData.isDraft,
        }
        const response = await api.post('/projects', apiData)
        const responseData = response.data.data
        // Map backend field names to frontend field names
        return projectService.normalizeProject(responseData)
    },

    async updateProject(id: string, projectData: UpdateProjectRequest): Promise<Project> {
        // Map frontend field names to backend API field names
        const apiData = {
            name: projectData.name,
            slug: projectData.slug,
            description: projectData.description,
            git_repo: projectData.repositoryUrl,
            git_branch: projectData.branch,
            git_provider: projectData.gitProvider,
            repository_auth_id: projectData.repositoryAuthId,
            visibility: projectData.visibility,
            is_draft: projectData.isDraft,
        }
        const response = await api.put(`/projects/${id}`, apiData)
        const responseData = response.data.data
        // Map backend field names to frontend field names
        return projectService.normalizeProject(responseData)
    },

    async deleteProject(id: string, params?: { source?: string }): Promise<void> {
        await api.delete(`/projects/${id}`, { params })
    },

    async listProjectSources(projectId: string): Promise<ProjectSource[]> {
        const response = await api.get(`/projects/${projectId}/sources`)
        const items = response.data?.data || []
        return items.map((item: any) => projectService.normalizeProjectSource(item))
    },

    async createProjectSource(projectId: string, payload: CreateProjectSourceRequest): Promise<ProjectSource> {
        const apiData = {
            name: payload.name,
            provider: payload.provider,
            repository_url: payload.repositoryUrl,
            default_branch: payload.defaultBranch,
            repository_auth_id: payload.repositoryAuthId,
            is_default: payload.isDefault ?? false,
            is_active: payload.isActive ?? true,
        }
        const response = await api.post(`/projects/${projectId}/sources`, apiData)
        return projectService.normalizeProjectSource(response.data?.data)
    },

    async updateProjectSource(projectId: string, sourceId: string, payload: UpdateProjectSourceRequest): Promise<ProjectSource> {
        const apiData = {
            name: payload.name,
            provider: payload.provider,
            repository_url: payload.repositoryUrl,
            default_branch: payload.defaultBranch,
            repository_auth_id: payload.repositoryAuthId,
            is_default: payload.isDefault ?? false,
            is_active: payload.isActive ?? true,
        }
        const response = await api.patch(`/projects/${projectId}/sources/${sourceId}`, apiData)
        return projectService.normalizeProjectSource(response.data?.data)
    },

    async deleteProjectSource(projectId: string, sourceId: string): Promise<void> {
        await api.delete(`/projects/${projectId}/sources/${sourceId}`)
    },

    async listProjectWebhookReceipts(projectId: string, params?: { limit?: number; offset?: number }): Promise<ProjectWebhookReceipt[]> {
        const response = await api.get(`/projects/${projectId}/webhook-receipts`, { params })
        const items = response.data?.data || []
        return items.map((item: any) => projectService.normalizeWebhookReceipt(item))
    },

    async getProjectBuildSettings(projectId: string): Promise<ProjectBuildSettings> {
        const response = await api.get(`/projects/${projectId}/build-settings`)
        return projectService.normalizeProjectBuildSettings(response.data?.data)
    },

    async updateProjectBuildSettings(projectId: string, payload: { buildConfigMode: 'ui_managed' | 'repo_managed'; buildConfigFile?: string; buildConfigOnError?: 'strict' | 'fallback_to_ui' }): Promise<ProjectBuildSettings> {
        const response = await api.put(`/projects/${projectId}/build-settings`, {
            build_config_mode: payload.buildConfigMode,
            build_config_file: payload.buildConfigFile,
            build_config_on_error: payload.buildConfigOnError,
        })
        return projectService.normalizeProjectBuildSettings(response.data?.data)
    },

    // Build Configuration operations within projects
    async getBuildConfigurations(projectId: string, params?: {
        page?: number
        limit?: number
        isActive?: boolean
        search?: string
    }): Promise<PaginatedResponse<BuildConfiguration>> {
        const response = await api.get(`/projects/${projectId}/build-configs`, { params })
        return response.data
    },

    async getBuildConfiguration(projectId: string, configId: string): Promise<BuildConfiguration> {
        const response = await api.get(`/projects/${projectId}/build-configs/${configId}`)
        return response.data.data
    },

    async createBuildConfiguration(projectId: string, configData: CreateBuildConfigurationRequest): Promise<BuildConfiguration> {
        const response = await api.post(`/projects/${projectId}/build-configs`, configData)
        return response.data.data
    },

    async updateBuildConfiguration(projectId: string, configId: string, configData: UpdateBuildConfigurationRequest): Promise<BuildConfiguration> {
        const response = await api.put(`/projects/${projectId}/build-configs/${configId}`, configData)
        return response.data.data
    },

    async deleteBuildConfiguration(projectId: string, configId: string): Promise<void> {
        await api.delete(`/projects/${projectId}/build-configs/${configId}`)
    },

    // Project statistics
    async getProjectStats(projectId: string): Promise<{
        totalBuilds: number
        successfulBuilds: number
        failedBuilds: number
        averageDuration: number
        lastBuildAt?: string
        activeBuilds: number
    }> {
        const response = await api.get(`/projects/${projectId}/stats`)
        return response.data
    }
}
