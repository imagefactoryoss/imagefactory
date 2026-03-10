import { api } from './api'

export type BuildNotificationTriggerID =
  | 'BN-001'
  | 'BN-002'
  | 'BN-003'
  | 'BN-004'
  | 'BN-005'
  | 'BN-006'
  | 'BN-007'
  | 'BN-008'
  | 'BN-009'
  | 'BN-010'

export type BuildNotificationChannel = 'in_app' | 'email'
export type BuildNotificationRecipientPolicy =
  | 'initiator'
  | 'project_members'
  | 'tenant_admins'
  | 'custom_users'
export type BuildNotificationSeverity = 'low' | 'normal' | 'high'
export type BuildNotificationPreferenceSource = 'system' | 'tenant' | 'project'

export interface ProjectNotificationTriggerPreference {
  trigger_id: BuildNotificationTriggerID
  source?: BuildNotificationPreferenceSource
  enabled: boolean
  channels: BuildNotificationChannel[]
  recipient_policy: BuildNotificationRecipientPolicy
  custom_recipient_user_ids?: string[]
  severity_override?: BuildNotificationSeverity
}

export interface ProjectNotificationTriggerResponse {
  project_id: string
  preferences: ProjectNotificationTriggerPreference[]
}

export const projectNotificationTriggerService = {
  async getProjectNotificationTriggers(
    projectId: string,
  ): Promise<ProjectNotificationTriggerResponse> {
    const response = await api.get(`/projects/${projectId}/notification-triggers`)
    return response.data
  },

  async updateProjectNotificationTriggers(
    projectId: string,
    preferences: ProjectNotificationTriggerPreference[],
  ): Promise<ProjectNotificationTriggerResponse> {
    const response = await api.put(`/projects/${projectId}/notification-triggers`, {
      preferences,
    })
    return response.data
  },

  async deleteProjectNotificationTrigger(
    projectId: string,
    triggerId: BuildNotificationTriggerID,
  ): Promise<ProjectNotificationTriggerResponse> {
    const response = await api.delete(
      `/projects/${projectId}/notification-triggers/${triggerId}`,
    )
    return response.data
  },
}
