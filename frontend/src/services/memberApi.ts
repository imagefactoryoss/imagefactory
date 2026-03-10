/**
 * Project member management API service
 */

import { AddMemberPayload, MembersListResponse, ProjectMember, UpdateMemberRolePayload } from '@/types/projectMember';
import { api } from './api';

export const memberApi = {
    /**
     * Add a member to a project
     */
    addMember: async (projectId: string, payload: AddMemberPayload): Promise<ProjectMember> => {
        const response = await api.post(
            `/projects/${projectId}/members`,
            { user_id: payload.userId }
        );
        return response.data;
    },

    /**
     * Remove a member from a project
     */
    removeMember: async (projectId: string, userId: string): Promise<void> => {
        await api.delete(`/projects/${projectId}/members/${userId}`);
    },

    /**
     * List all members of a project
     */
    listMembers: async (
        projectId: string,
        limit: number = 20,
        offset: number = 0
    ): Promise<MembersListResponse> => {
        const response = await api.get(
            `/projects/${projectId}/members`,
            { params: { limit, offset } }
        );
        return response.data;
    },

    /**
     * Update a member's role
     */
    updateMemberRole: async (
        projectId: string,
        userId: string,
        payload: UpdateMemberRolePayload
    ): Promise<ProjectMember> => {
        const response = await api.patch(
            `/projects/${projectId}/members/${userId}`,
            { role_id: payload.roleId || null }
        );
        return response.data;
    },

    /**
     * Fetch a specific member
     */
    getMember: async (projectId: string, userId: string): Promise<ProjectMember> => {
        const response = await api.get(`/projects/${projectId}/members/${userId}`);
        return response.data;
    },
};
