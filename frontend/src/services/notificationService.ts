import { api } from './api'

export interface UserNotification {
  id: string
  title?: string
  message?: string
  notification_type?: string
  related_resource_type?: string
  related_resource_id?: string
  is_read: boolean
  read_at?: string
  created_at: string
}

export interface NotificationListResponse {
  notifications: UserNotification[]
  total_count: number
  limit: number
  offset: number
}

export interface NotificationUnreadCountResponse {
  unread_count: number
}

export const notificationService = {
  async list(params?: { unread?: boolean; limit?: number; offset?: number }): Promise<NotificationListResponse> {
    const response = await api.get('/notifications', { params })
    return response.data
  },

  async getUnreadCount(): Promise<NotificationUnreadCountResponse> {
    const response = await api.get('/notifications/unread-count')
    return response.data
  },

  async markAsRead(notificationId: string): Promise<void> {
    await api.post(`/notifications/${notificationId}/read`)
  },

  async deleteOne(notificationId: string): Promise<{ success: boolean }> {
    const response = await api.delete(`/notifications/${notificationId}`)
    return response.data
  },

  async markAllAsRead(): Promise<{ success: boolean; updated: number }> {
    const response = await api.post('/notifications/read-all')
    return response.data
  },

  async deleteRead(): Promise<{ success: boolean; deleted: number }> {
    const response = await api.delete('/notifications/read')
    return response.data
  },

  async deleteBulk(notificationIds: string[]): Promise<{ success: boolean; deleted: number; requested: number }> {
    const response = await api.post('/notifications/delete-bulk', { ids: notificationIds })
    return response.data
  },
}
