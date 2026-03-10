/**
 * Repository Authentication Types
 * TypeScript interfaces and enums for repository authentication
 */

export enum RepositoryAuthType {
    SSH_KEY = 'ssh_key',
    TOKEN = 'token',
    BASIC_AUTH = 'basic_auth',
    OAUTH = 'oauth',
}

export interface RepositoryAuth {
    id: string;
    tenant_id: string;
    project_id?: string;
    scope?: 'tenant' | 'project';
    name: string;
    description?: string;
    auth_type: RepositoryAuthType;
    is_active: boolean;
    version: number;
    created_at: string;
    updated_at: string;
    created_by: string;
    updated_by: string;
}

export interface RepositoryAuthSummary {
    id: string;
    project_id: string;
    project_name: string;
    git_provider_key?: string;
    name: string;
    description?: string;
    auth_type: RepositoryAuthType;
    is_active: boolean;
    created_by: string;
    created_by_email?: string;
    created_at: string;
    updated_at: string;
}

export interface RepositoryAuthCreate {
    project_id?: string;
    name: string;
    description?: string;
    auth_type: RepositoryAuthType;
    username?: string;
    ssh_key?: string;
    token?: string;
    password?: string;
}

export interface RepositoryAuthUpdate {
    name?: string;
    description?: string;
    auth_type?: RepositoryAuthType;
    username?: string;
    ssh_key?: string;
    token?: string;
    password?: string;
}

export interface RepositoryAuthList {
    repository_auths: RepositoryAuth[];
    total_count: number;
}

export interface TestConnectionResponse {
    success: boolean;
    message: string;
    details?: Record<string, any>;
}
