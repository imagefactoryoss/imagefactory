import { api } from './api'

export type BuildTriggerType = 'webhook' | 'schedule' | 'git_event'

export interface BuildTrigger {
    id: string
    buildId: string
    projectId: string
    triggerType: BuildTriggerType
    name: string
    description?: string
    webhookUrl?: string
    webhookSecret?: string
    webhookEvents?: string[]
    isActive: boolean
    createdBy: string
    createdAt: string
    updatedAt: string
}

interface BuildTriggersListResponse {
    triggers: any[]
    total: number
}

export interface UpdateWebhookTriggerRequest {
    name?: string
    description?: string
    webhookUrl?: string
    webhookSecret?: string
    webhookEvents?: string[]
    isActive?: boolean
}

export interface CreateWebhookTriggerRequest {
    name: string
    description?: string
    webhookUrl: string
    webhookSecret?: string
    webhookEvents?: string[]
    buildId?: string
}

const normalizeTrigger = (payload: any): BuildTrigger => ({
    id: payload.id,
    buildId: payload.buildId || payload.build_id,
    projectId: payload.projectId || payload.project_id,
    triggerType: payload.triggerType || payload.trigger_type,
    name: payload.name || payload.trigger_name || '',
    description: payload.description || payload.trigger_description || '',
    webhookUrl: payload.webhookUrl || payload.webhook_url,
    webhookSecret: payload.webhookSecret || payload.webhook_secret,
    webhookEvents: payload.webhookEvents || payload.webhook_events || [],
    isActive: payload.isActive ?? payload.is_active ?? true,
    createdBy: payload.createdBy || payload.created_by,
    createdAt: payload.createdAt || payload.created_at,
    updatedAt: payload.updatedAt || payload.updated_at,
})

export const buildTriggerService = {
    async getBuildTriggers(projectId: string, buildId: string): Promise<BuildTrigger[]> {
        const response = await api.get<BuildTriggersListResponse>(`/projects/${projectId}/builds/${buildId}/triggers`)
        const triggers = response.data?.triggers || []
        return triggers.map(normalizeTrigger)
    },

    async createWebhookTrigger(projectId: string, buildId: string, payload: CreateWebhookTriggerRequest): Promise<BuildTrigger> {
        const response = await api.post(`/projects/${projectId}/builds/${buildId}/triggers/webhook`, {
            name: payload.name,
            description: payload.description,
            webhook_url: payload.webhookUrl,
            webhook_secret: payload.webhookSecret,
            webhook_events: payload.webhookEvents,
        })
        return normalizeTrigger(response.data)
    },

    async createProjectWebhookTrigger(projectId: string, payload: CreateWebhookTriggerRequest): Promise<BuildTrigger> {
        const response = await api.post(`/projects/${projectId}/triggers/webhook`, {
            build_id: payload.buildId,
            name: payload.name,
            description: payload.description,
            webhook_url: payload.webhookUrl,
            webhook_secret: payload.webhookSecret,
            webhook_events: payload.webhookEvents,
        })
        return normalizeTrigger(response.data)
    },

    async getProjectTriggers(projectId: string): Promise<BuildTrigger[]> {
        const response = await api.get<BuildTriggersListResponse>(`/projects/${projectId}/triggers`)
        const triggers = response.data?.triggers || []
        return triggers.map(normalizeTrigger)
    },

    async updateProjectWebhookTrigger(projectId: string, triggerId: string, payload: UpdateWebhookTriggerRequest): Promise<BuildTrigger> {
        const response = await api.patch(`/projects/${projectId}/triggers/${triggerId}`, {
            name: payload.name,
            description: payload.description,
            webhook_url: payload.webhookUrl,
            webhook_secret: payload.webhookSecret,
            webhook_events: payload.webhookEvents,
            is_active: payload.isActive,
        })
        return normalizeTrigger(response.data)
    },

    async deleteTrigger(projectId: string, triggerId: string): Promise<void> {
        await api.delete(`/projects/${projectId}/triggers/${triggerId}`)
    },
}
