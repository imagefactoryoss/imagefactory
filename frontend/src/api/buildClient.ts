/**
 * Build Execution API Client
 * Type-safe client for all build endpoints with WebSocket support
 */

import { api } from '@/services/api';
// ============================================================================
// TYPE DEFINITIONS
// ============================================================================

export enum BuildStatus {
    QUEUED = 'queued',
    RUNNING = 'running',
    COMPLETED = 'completed',
    // Backward-compatible aliases for older payloads.
    IN_PROGRESS = 'running',
    SUCCESS = 'completed',
    FAILED = 'failed',
    CANCELLED = 'cancelled',
}

export enum LogLevel {
    DEBUG = 'DEBUG',
    INFO = 'INFO',
    WARN = 'WARN',
    ERROR = 'ERROR',
}

// Build resource
export interface Build {
    id: string;
    project_id: string;
    build_number: number;
    git_branch?: string;
    git_commit?: string;
    git_author_name?: string;
    git_author_email?: string;
    status: BuildStatus;
    error_message?: string;
    started_at?: string;
    completed_at?: string;
    created_at: string;
    updated_at: string;
}

// Build list response
export interface BuildList {
    builds: Build[];
    total_count: number;
    page: number;
    limit: number;
    has_more: boolean;
}

// Build status response
export interface BuildStatusResponse {
    build_id: string;
    status: BuildStatus | 'in_progress' | 'success';
    progress?: {
        current_step: number;
        total_steps: number;
        percentage: number;
    };
    started_at?: string;
    estimated_completion?: string;
    error?: string;
}

// Log entry
export interface LogEntry {
    timestamp: string;
    level: LogLevel;
    message: string;
    metadata?: Record<string, string>;
}

// Build logs response
export interface BuildLogs {
    build_id: string;
    logs: LogEntry[];
    total_lines: number;
    offset: number;
    limit: number;
}

// WebSocket log message
export interface LogMessage {
    build_id: string;
    timestamp: string;
    level: LogLevel;
    message: string;
    metadata?: Record<string, string>;
}

// Create build request
export interface CreateBuildRequest {
    project_id: string;
    git_branch: string;
    git_commit?: string;
    build_args?: Record<string, string>;
}

// List builds filter
export interface ListBuildsFilter {
    page?: number;
    limit?: number;
    status?: BuildStatus;
    project_id?: string;
    sort_by?: 'created_at' | 'updated_at' | 'build_number';
    sort_order?: 'asc' | 'desc';
}

// Error response
export interface APIError {
    error: string;
    code: string;
    timestamp: string;
    trace_id?: string;
    details?: Record<string, string>;
}

// ============================================================================
// BUILD API CLIENT
// ============================================================================

export class BuildClient {
    private baseURL: string;

    constructor(_token: string, baseURL: string = 'http://localhost:8080/api/v1') {
        this.baseURL = baseURL;
    }

    // Refresh token for expired tokens
    setToken(_token: string): void {
    }

    // ========================================================================
    // BUILD OPERATIONS
    // ========================================================================

    /**
     * Create a new build
     */
    async createBuild(request: CreateBuildRequest): Promise<Build> {
        const response = await api.post('/builds', request);
        return response.data;
    }

    /**
     * List all builds with filters
     */
    async listBuilds(filter?: ListBuildsFilter): Promise<BuildList> {
        const response = await api.get('/builds', {
            params: filter,
        });
        return response.data;
    }

    /**
     * Get build details
     */
    async getBuild(buildId: string): Promise<Build> {
        const response = await api.get(`/builds/${buildId}`);
        return response.data;
    }

    /**
     * Delete a build
     */
    async deleteBuild(buildId: string): Promise<void> {
        await api.delete(`/builds/${buildId}`);
    }

    // ========================================================================
    // BUILD ACTIONS
    // ========================================================================

    /**
     * Start a build
     */
    async startBuild(buildId: string): Promise<Build> {
        const response = await api.post(`/builds/${buildId}/start`);
        return response.data;
    }

    /**
     * Cancel a build
     */
    async cancelBuild(buildId: string, reason?: string): Promise<Build> {
        const response = await api.post(`/builds/${buildId}/cancel`, { reason });
        return response.data;
    }

    /**
     * Retry a failed build
     */
    async retryBuild(buildId: string): Promise<Build> {
        const response = await api.post(`/builds/${buildId}/retry`);
        return response.data;
    }

    // ========================================================================
    // BUILD STATUS & LOGS
    // ========================================================================

    /**
     * Get build status
     */
    async getBuildStatus(buildId: string): Promise<BuildStatusResponse> {
        const response = await api.get(`/builds/${buildId}/status`);
        return response.data;
    }

    /**
     * Get build logs (HTTP)
     */
    async getBuildLogs(buildId: string, offset: number = 0, limit: number = 100): Promise<BuildLogs> {
        const response = await api.get(`/builds/${buildId}/logs`, {
            params: { offset, limit },
        });
        return response.data;
    }

    /**
     * Stream build logs via WebSocket
     */
    streamBuildLogs(buildId: string): Promise<WebSocket> {
        return new Promise((resolve, reject) => {
            const protocol = this.baseURL.startsWith('https') ? 'wss' : 'ws';
            const wsURL = this.baseURL
                .replace('http://', `${protocol}://`)
                .replace('https://', `${protocol}://`) +
                `/builds/${buildId}/logs/stream`;

            try {
                const ws = new WebSocket(wsURL);

                // Set auth header for WebSocket
                ws.addEventListener('open', () => {
                    // Some WebSocket implementations don't support headers directly
                    // The server should validate token from the original HTTP upgrade request
                });

                ws.addEventListener('error', (error) => {
                    reject(new Error(`WebSocket connection failed: ${error}`));
                });

                ws.addEventListener('open', () => {
                    resolve(ws);
                });

                // Custom method to send auth if needed
                (ws as any).setToken = (token: string) => {
                    this.setToken(token);
                };
            } catch (error) {
                reject(error);
            }
        });
    }

    // ========================================================================
    // ERROR HANDLING
    // ========================================================================

    // Error handling is centralized in the shared axios client.
}

// ============================================================================
// SINGLETON INSTANCE (for global use)
// ============================================================================

let instance: BuildClient | null = null;

export function initializeBuildClient(token: string, baseURL?: string): BuildClient {
    instance = new BuildClient(token, baseURL);
    return instance;
}

export function getBuildClient(): BuildClient {
    if (!instance) {
        throw new Error('BuildClient not initialized. Call initializeBuildClient first.');
    }
    return instance;
}

export function updateBuildClientToken(token: string): void {
    if (instance) {
        instance.setToken(token);
    }
}
