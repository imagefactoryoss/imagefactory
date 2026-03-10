import { ConfigTemplate, ConfigTemplateListResponse, SaveTemplateRequest } from '../types/buildConfig';
import { api } from './api';

const API_BASE = '/api/templates';

class ConfigTemplateService {
    async saveTemplate(req: SaveTemplateRequest): Promise<{ data: ConfigTemplate }> {
        const response = await api.request({
            url: API_BASE,
            method: 'POST',
            data: req,
            baseURL: '',
        });

        return response.data;
    }

    async getTemplate(templateId: string): Promise<{ data: ConfigTemplate }> {
        const response = await api.request({
            url: `${API_BASE}/${templateId}`,
            method: 'GET',
            baseURL: '',
        });

        return response.data;
    }

    async listTemplates(
        projectId: string,
        limit: number = 20,
        offset: number = 0
    ): Promise<ConfigTemplateListResponse> {
        const params = new URLSearchParams({
            project_id: projectId,
            limit: limit.toString(),
            offset: offset.toString(),
        });

        const response = await api.request({
            url: `${API_BASE}?${params}`,
            method: 'GET',
            baseURL: '',
        });

        return response.data;
    }

    async updateTemplate(
        templateId: string,
        updates: Partial<SaveTemplateRequest>
    ): Promise<{ data: ConfigTemplate }> {
        const response = await api.request({
            url: `${API_BASE}/${templateId}`,
            method: 'PUT',
            data: updates,
            baseURL: '',
        });

        return response.data;
    }

    async deleteTemplate(templateId: string): Promise<void> {
        await api.request({
            url: `${API_BASE}/${templateId}`,
            method: 'DELETE',
            baseURL: '',
        });
    }

    async shareTemplate(
        templateId: string,
        userId: string,
        canUse: boolean = true,
        canEdit: boolean = false,
        canDelete: boolean = false
    ): Promise<any> {
        const response = await api.request({
            url: `${API_BASE}/${templateId}/share`,
            method: 'POST',
            data: {
                template_id: templateId,
                shared_with_user_id: userId,
                can_use: canUse,
                can_edit: canEdit,
                can_delete: canDelete,
            },
            baseURL: '',
        });

        return response.data;
    }

    async getSharedWithUser(): Promise<{ data: any[] }> {
        const response = await api.request({
            url: `${API_BASE}/user/shared`,
            method: 'GET',
            baseURL: '',
        });

        return response.data;
    }
}

export const configTemplateService = new ConfigTemplateService();
