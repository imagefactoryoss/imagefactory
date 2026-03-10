/**
 * Repository Authentication API Client
 * Type-safe client for repository authentication operations
 */

import {
    RepositoryAuth,
    RepositoryAuthCreate,
    RepositoryAuthList,
    RepositoryAuthSummary,
    RepositoryAuthUpdate,
    TestConnectionResponse
} from '@/types/repositoryAuth';
import { api } from '@/services/api';

// ============================================================================
// API CLIENT
// ============================================================================

class RepositoryAuthClient {
    async createScopedRepositoryAuth(data: RepositoryAuthCreate): Promise<RepositoryAuth> {
        const response = await api.post('/repository-auth', data);
        return response.data;
    }

    async listScopedRepositoryAuth(projectId?: string, includeTenant: boolean = false): Promise<RepositoryAuthList> {
        const params = new URLSearchParams();
        if (projectId) {
            params.set('project_id', projectId);
            params.set('include_tenant', includeTenant ? 'true' : 'false');
        }
        const suffix = params.toString() ? `?${params.toString()}` : '';
        const response = await api.get(`/repository-auth${suffix}`);
        const data = response.data as any;
        const auths: RepositoryAuth[] = Array.isArray(data)
            ? data
            : Array.isArray(data?.repository_auths)
                ? data.repository_auths
                : Array.isArray(data?.data)
                    ? data.data
                    : [];
        return {
            repository_auths: auths,
            total_count: typeof data?.total_count === 'number' ? data.total_count : auths.length,
        };
    }

    async deleteScopedRepositoryAuth(authId: string): Promise<void> {
        await api.delete(`/repository-auth/${authId}`);
    }

    async updateScopedRepositoryAuth(authId: string, data: RepositoryAuthUpdate): Promise<RepositoryAuth> {
        const response = await api.put(`/repository-auth/${authId}`, data);
        return response.data;
    }

    // Create a new repository authentication
    async createRepositoryAuth(
        projectId: string,
        data: RepositoryAuthCreate
    ): Promise<RepositoryAuth> {
        const response = await api.post(`/projects/${projectId}/repository-auth`, data);
        return response.data;
    }

    // Get all repository authentications for a project
    async getRepositoryAuths(projectId: string, includeTenant: boolean = true): Promise<RepositoryAuthList> {
        const response = await api.get(`/projects/${projectId}/repository-auth?include_tenant=${includeTenant ? 'true' : 'false'}`);
        const data = response.data as any;
        const auths: RepositoryAuth[] = Array.isArray(data)
            ? data
            : Array.isArray(data?.repository_auths)
                ? data.repository_auths
                : Array.isArray(data?.data)
                    ? data.data
                    : [];

        return {
            repository_auths: auths,
            total_count: typeof data?.total_count === 'number' ? data.total_count : auths.length,
        };
    }

    // Get a specific repository authentication
    async getRepositoryAuth(
        projectId: string,
        authId: string
    ): Promise<RepositoryAuth> {
        const response = await api.get(`/projects/${projectId}/repository-auth/${authId}`);
        return response.data;
    }

    // Update a repository authentication
    async updateRepositoryAuth(
        projectId: string,
        authId: string,
        data: RepositoryAuthUpdate
    ): Promise<RepositoryAuth> {
        const response = await api.put(`/projects/${projectId}/repository-auth/${authId}`, data);
        return response.data;
    }

    // Delete a repository authentication
    async deleteRepositoryAuth(
        projectId: string,
        authId: string
    ): Promise<void> {
        await api.delete(`/projects/${projectId}/repository-auth/${authId}`);
    }

    // Test connection for a repository authentication
    async testRepositoryAuth(
        projectId: string,
        authId: string,
        options?: { full_test?: boolean; repo_url?: string }
    ): Promise<TestConnectionResponse> {
        const response = await api.post(`/projects/${projectId}/repository-auth/${authId}/test-connection`, options || {});
        return response.data;
    }

    // List repository authentications available to clone for a project
    async getAvailableRepositoryAuths(projectId: string): Promise<{ repository_auths: RepositoryAuthSummary[]; total_count: number }> {
        const response = await api.get(`/projects/${projectId}/repository-auth/available`);
        return response.data;
    }

    // Clone a repository authentication into a project
    async cloneRepositoryAuth(
        projectId: string,
        sourceAuthId: string,
        name?: string,
        description?: string
    ): Promise<RepositoryAuth> {
        const response = await api.post(`/projects/${projectId}/repository-auth/clone`, {
            source_auth_id: sourceAuthId,
            name,
            description,
        });
        return response.data;
    }
}

// Export singleton instance
export const repositoryAuthClient = new RepositoryAuthClient();
