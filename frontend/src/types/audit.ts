export type AuditEventType =
    | 'login_success'
    | 'login_failure'
    | 'logout'
    | 'password_change'
    | 'user_create'
    | 'user_update'
    | 'user_delete'
    | 'tenant_create'
    | 'tenant_update'
    | 'tenant_activate'
    | 'tenant_delete'
    | 'role_assign'
    | 'role_remove'
    | 'permission_check'
    | 'config_change'
    | 'build_create'
    | 'build_start'
    | 'build_cancel'
    | 'server_start'
    | 'project_purge'

export type AuditEventSeverity = 'info' | 'warning' | 'error' | 'critical'

export interface AuditEvent {
    id: string
    tenant_id?: string
    user_id?: string
    user_name?: string
    event_type: AuditEventType
    severity: AuditEventSeverity
    resource: string
    action: string
    ip_address?: string
    user_agent?: string
    details?: Record<string, any>
    message: string
    timestamp: string
    created_at?: string
}

export interface AuditEventFilter {
    user_id?: string
    event_type?: AuditEventType
    severity?: AuditEventSeverity
    resource?: string
    action?: string
    start_time?: string
    end_time?: string
}
