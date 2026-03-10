/**
 * Custom hooks for project member management
 */

import { memberApi } from '@/services/memberApi';
import { ProjectMember } from '@/types/projectMember';
import { useCallback, useEffect, useState } from 'react';

interface UseMembersOptions {
    projectId: string;
    initialLimit?: number;
}

interface UseMembersResult {
    members: ProjectMember[];
    loading: boolean;
    error: string | null;
    totalCount: number;
    limit: number;
    offset: number;
    hasMore: boolean;
    refetch: () => Promise<void>;
    loadMore: () => Promise<void>;
}

/**
 * Hook for fetching and managing project members
 */
export const useProjectMembers = ({
    projectId,
    initialLimit = 20,
}: UseMembersOptions): UseMembersResult => {
    const [members, setMembers] = useState<ProjectMember[]>([]);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);
    const [totalCount, setTotalCount] = useState(0);
    const [offset, setOffset] = useState(0);
    const limit = initialLimit;

    const fetchMembers = useCallback(
        async (currentOffset: number = 0) => {
            try {
                setLoading(true);
                setError(null);
                const response = await memberApi.listMembers(projectId, limit, currentOffset);
                if (currentOffset === 0) {
                    setMembers(response.members);
                } else {
                    setMembers((prev) => [...prev, ...response.members]);
                }
                setTotalCount(response.totalCount);
                setOffset(currentOffset);
            } catch (err) {
                const errorMessage = err instanceof Error ? err.message : 'Failed to fetch members';
                setError(errorMessage);
            } finally {
                setLoading(false);
            }
        },
        [projectId, limit]
    );

    useEffect(() => {
        fetchMembers(0);
    }, [fetchMembers]);

    const refetch = useCallback(() => fetchMembers(0), [fetchMembers]);

    const loadMore = useCallback(() => {
        const nextOffset = offset + limit;
        if (nextOffset < totalCount) {
            return fetchMembers(nextOffset);
        }
        return Promise.resolve();
    }, [offset, limit, totalCount, fetchMembers]);

    return {
        members,
        loading,
        error,
        totalCount,
        limit,
        offset,
        hasMore: offset + limit < totalCount,
        refetch,
        loadMore,
    };
};

interface UseMemberManagementOptions {
    projectId: string;
    onSuccess?: () => void;
}

interface UseMemberManagementResult {
    addMember: (userId: string) => Promise<ProjectMember>;
    removeMember: (userId: string) => Promise<void>;
    updateMemberRole: (userId: string, roleId?: string) => Promise<ProjectMember>;
    loading: boolean;
    error: string | null;
    success: string | null;
}

/**
 * Hook for managing member operations (add, remove, update)
 */
export const useMemberManagement = ({
    projectId,
    onSuccess,
}: UseMemberManagementOptions): UseMemberManagementResult => {
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState<string | null>(null);
    const [success, setSuccess] = useState<string | null>(null);

    const addMember = useCallback(
        async (userId: string): Promise<ProjectMember> => {
            try {
                setLoading(true);
                setError(null);
                setSuccess(null);
                const member = await memberApi.addMember(projectId, { userId });
                setSuccess('Member added successfully');
                onSuccess?.();
                return member;
            } catch (err) {
                const errorMessage = err instanceof Error ? err.message : 'Failed to add member';
                setError(errorMessage);
                throw err;
            } finally {
                setLoading(false);
            }
        },
        [projectId, onSuccess]
    );

    const removeMember = useCallback(
        async (userId: string): Promise<void> => {
            try {
                setLoading(true);
                setError(null);
                setSuccess(null);
                await memberApi.removeMember(projectId, userId);
                setSuccess('Member removed successfully');
                onSuccess?.();
            } catch (err) {
                const errorMessage = err instanceof Error ? err.message : 'Failed to remove member';
                setError(errorMessage);
                throw err;
            } finally {
                setLoading(false);
            }
        },
        [projectId, onSuccess]
    );

    const updateMemberRole = useCallback(
        async (userId: string, roleId?: string): Promise<ProjectMember> => {
            try {
                setLoading(true);
                setError(null);
                setSuccess(null);
                const member = await memberApi.updateMemberRole(projectId, userId, {
                    roleId: roleId || null,
                });
                setSuccess(roleId ? 'Role updated successfully' : 'Role override removed successfully');
                onSuccess?.();
                return member;
            } catch (err) {
                const errorMessage = err instanceof Error ? err.message : 'Failed to update member role';
                setError(errorMessage);
                throw err;
            } finally {
                setLoading(false);
            }
        },
        [projectId, onSuccess]
    );

    return {
        addMember,
        removeMember,
        updateMemberRole,
        loading,
        error,
        success,
    };
};
