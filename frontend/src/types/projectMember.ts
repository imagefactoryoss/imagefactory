/**
 * Project member management types and interfaces
 */

export interface ProjectMember {
    id: string;
    projectId: string;
    userId: string;
    roleId?: string;
    assignedByUserId?: string;
    createdAt: string;
    updatedAt: string;
}

export interface MembersListResponse {
    members: ProjectMember[];
    totalCount: number;
    limit: number;
    offset: number;
}

export interface AddMemberRequest {
    user_id: string;
}

export interface UpdateMemberRoleRequest {
    role_id: string | null;
}

export interface AddMemberPayload {
    userId: string;
}

export interface UpdateMemberRolePayload {
    roleId?: string | null;
}
