import type { SREIncidentWorkspaceResponse, SREMCPToolInvocationResponse } from '@/types'

export interface AsyncBacklogInsight {
    source: 'workspace' | 'tool'
    incidentType: string
    displayName: string
    queueKind: string
    subsystem: string
    count: number
    threshold: number
    thresholdDelta: number
    thresholdRatioPercent: number
    trend: string
    operatorStatus: string
    latestSummary: string
    latestCapturedAt: string
    recentObservations: Record<string, number>
    correlationHints: Record<string, string>
}

export interface MessagingTransportInsight {
    source: 'workspace' | 'tool'
    status: string
    reconnects: number
    disconnects: number
    reconnectThreshold: number
    operatorStatus: string
    latestSummary: string
    latestCapturedAt: string
}

export interface MessagingConsumerInsight {
    source: 'workspace' | 'tool'
    incidentType: string
    displayName: string
    kind: string
    stream: string
    consumer: string
    targetRef: string
    count: number
    threshold: number
    thresholdDelta: number
    thresholdRatioPercent: number
    pendingCount: number
    ackPendingCount: number
    waitingCount: number
    redeliveredCount: number
    trend: string
    operatorStatus: string
    latestSummary: string
    latestCapturedAt: string
    lastActive: string
    correlationHints: Record<string, string>
}

const asRecord = (value: unknown): Record<string, any> | null => {
    if (!value || typeof value !== 'object' || Array.isArray(value)) return null
    return value as Record<string, any>
}

const isNumber = (value: unknown): value is number => typeof value === 'number' && Number.isFinite(value)

const isString = (value: unknown): value is string => typeof value === 'string'

const percentageFromThreshold = (count: number, threshold: number) => {
    if (threshold <= 0) return 0
    return Math.round((count / threshold) * 100)
}

const asyncBacklogToolConfig: Record<string, { countKey: string; thresholdKey: string; displayName: string; queueKind: string; subsystem: string }> = {
    backlog_pressure: {
        countKey: 'build_queue_depth',
        thresholdKey: 'build_queue_threshold',
        displayName: 'Dispatcher backlog',
        queueKind: 'build_queue',
        subsystem: 'dispatcher',
    },
    dispatcher_backlog_pressure: {
        countKey: 'build_queue_depth',
        thresholdKey: 'build_queue_threshold',
        displayName: 'Dispatcher backlog',
        queueKind: 'build_queue',
        subsystem: 'dispatcher',
    },
    email_queue_backlog_pressure: {
        countKey: 'email_queue_depth',
        thresholdKey: 'email_queue_threshold',
        displayName: 'Email queue backlog',
        queueKind: 'email_queue',
        subsystem: 'notifications',
    },
    messaging_outbox_backlog_pressure: {
        countKey: 'messaging_outbox_pending',
        thresholdKey: 'messaging_outbox_threshold',
        displayName: 'Messaging outbox backlog',
        queueKind: 'messaging_outbox',
        subsystem: 'messaging',
    },
}

export const isAsyncBacklogIncidentType = (incidentType?: string | null) => {
    switch ((incidentType || '').trim()) {
        case 'backlog_pressure':
        case 'dispatcher_backlog_pressure':
        case 'email_queue_backlog_pressure':
        case 'messaging_outbox_backlog_pressure':
            return true
        default:
            return false
    }
}

export const isMessagingConsumerIncidentType = (incidentType?: string | null) => {
    switch ((incidentType || '').trim()) {
        case 'nats_consumer_lag_pressure':
        case 'nats_consumer_stalled_progress':
        case 'nats_consumer_ack_pressure':
            return true
        default:
            return false
    }
}

export const deriveAsyncBacklogInsight = (
    incidentType: string | null | undefined,
    workspace: SREIncidentWorkspaceResponse | null,
    result?: SREMCPToolInvocationResponse | null,
): AsyncBacklogInsight | null => {
    const backlog = workspace?.async_pressure_summary?.backlog
    if (backlog) {
        return {
            source: 'workspace',
            incidentType: backlog.incident_type,
            displayName: backlog.display_name,
            queueKind: backlog.queue_kind,
            subsystem: backlog.subsystem,
            count: backlog.count,
            threshold: backlog.threshold,
            thresholdDelta: backlog.threshold_delta,
            thresholdRatioPercent: backlog.threshold_ratio_percent,
            trend: backlog.trend,
            operatorStatus: backlog.operator_status,
            latestSummary: backlog.latest_summary,
            latestCapturedAt: backlog.latest_captured_at || '',
            recentObservations: backlog.recent_observations || {},
            correlationHints: backlog.correlation_hints || {},
        }
    }

    if (!isAsyncBacklogIncidentType(incidentType)) return null

    const config = asyncBacklogToolConfig[(incidentType || '').trim()]
    if (!config) return null

    const payload = asRecord(result?.payload)
    if (!payload) return null

    const count = isNumber(payload[config.countKey]) ? payload[config.countKey] : 0
    const threshold = isNumber(payload[config.thresholdKey]) ? payload[config.thresholdKey] : 0
    const recentObservations = Object.entries({
        build_queue_depth: payload.build_queue_depth,
        email_queue_depth: payload.email_queue_depth,
        messaging_outbox_pending: payload.messaging_outbox_pending,
    }).reduce<Record<string, number>>((acc, [key, value]) => {
        if (isNumber(value)) {
            acc[key] = value
        }
        return acc
    }, {})

    return {
        source: 'tool',
        incidentType: (incidentType || '').trim(),
        displayName: config.displayName,
        queueKind: config.queueKind,
        subsystem: config.subsystem,
        count,
        threshold,
        thresholdDelta: count - threshold,
        thresholdRatioPercent: percentageFromThreshold(count, threshold),
        trend: 'unknown',
        operatorStatus: threshold > 0 && count > threshold ? 'Above configured backlog threshold' : 'Within configured backlog threshold',
        latestSummary: `Latest ${config.displayName.toLowerCase()} snapshot captured from MCP tool output.`,
        latestCapturedAt: isString(payload.last_activity) ? payload.last_activity : '',
        recentObservations,
        correlationHints: {},
    }
}

export const deriveMessagingTransportInsight = (
    workspace: SREIncidentWorkspaceResponse | null,
    result?: SREMCPToolInvocationResponse | null,
): MessagingTransportInsight | null => {
    const transport = workspace?.async_pressure_summary?.messaging_transport
    if (transport) {
        return {
            source: 'workspace',
            status: transport.status,
            reconnects: transport.reconnects,
            disconnects: transport.disconnects,
            reconnectThreshold: transport.reconnect_threshold,
            operatorStatus: transport.operator_status,
            latestSummary: transport.latest_summary,
            latestCapturedAt: transport.latest_captured_at || '',
        }
    }

    const payload = asRecord(result?.payload)
    if (!payload) return null

    const reconnects = isNumber(payload.reconnects) ? payload.reconnects : 0
    const disconnects = isNumber(payload.disconnects) ? payload.disconnects : 0
    const reconnectThreshold = isNumber(payload.reconnect_threshold) ? payload.reconnect_threshold : 0
    const status = reconnects > 0 || disconnects > 0 ? 'degraded' : 'stable'

    return {
        source: 'tool',
        status,
        reconnects,
        disconnects,
        reconnectThreshold,
        operatorStatus: status === 'degraded' ? 'Transport reconnect pressure observed' : 'Transport stable in the latest snapshot',
        latestSummary: status === 'degraded'
            ? 'Latest transport snapshot captured reconnect or disconnect pressure from MCP tool output.'
            : 'Latest transport snapshot did not capture reconnect or disconnect pressure.',
        latestCapturedAt: isString(payload.last_activity) ? payload.last_activity : '',
    }
}

export const deriveMessagingConsumerInsight = (
    incidentType: string | null | undefined,
    workspace: SREIncidentWorkspaceResponse | null,
    result?: SREMCPToolInvocationResponse | null,
): MessagingConsumerInsight | null => {
    const consumer = workspace?.async_pressure_summary?.messaging_consumer
    if (consumer) {
        return {
            source: 'workspace',
            incidentType: consumer.incident_type,
            displayName: consumer.display_name,
            kind: consumer.kind,
            stream: consumer.stream,
            consumer: consumer.consumer,
            targetRef: consumer.target_ref,
            count: consumer.count,
            threshold: consumer.threshold,
            thresholdDelta: consumer.threshold_delta,
            thresholdRatioPercent: consumer.threshold_ratio_percent,
            pendingCount: consumer.pending_count,
            ackPendingCount: consumer.ack_pending_count,
            waitingCount: consumer.waiting_count,
            redeliveredCount: consumer.redelivered_count,
            trend: consumer.trend,
            operatorStatus: consumer.operator_status,
            latestSummary: consumer.latest_summary,
            latestCapturedAt: consumer.latest_captured_at || '',
            lastActive: consumer.last_active || '',
            correlationHints: consumer.correlation_hints || {},
        }
    }

    if (!isMessagingConsumerIncidentType(incidentType)) return null

    const payload = asRecord(result?.payload)
    if (!payload) return null

    const consumers = Array.isArray(payload.consumers) ? payload.consumers.map((item) => asRecord(item)).filter((item): item is Record<string, any> => item !== null) : []
    const prioritized = consumers.sort((left, right) => {
        const leftPending = isNumber(left.pending_count) ? left.pending_count : 0
        const rightPending = isNumber(right.pending_count) ? right.pending_count : 0
        return rightPending - leftPending
    })[0]
    if (!prioritized) return null

    const count = isNumber(prioritized.pending_count) ? prioritized.pending_count : 0
    const threshold = isNumber(payload.max_pending_count) && payload.max_pending_count > 0 ? payload.max_pending_count : count
    return {
        source: 'tool',
        incidentType: (incidentType || '').trim(),
        displayName: `Messaging consumer pressure for ${isString(prioritized.stream) ? prioritized.stream : 'unknown-stream'}/${isString(prioritized.consumer) ? prioritized.consumer : 'unknown-consumer'}`,
        kind: (incidentType || '').trim().replace(/^nats_/, '').replace(/_/g, ' '),
        stream: isString(prioritized.stream) ? prioritized.stream : '',
        consumer: isString(prioritized.consumer) ? prioritized.consumer : '',
        targetRef: `${isString(prioritized.stream) ? prioritized.stream : 'unknown-stream'}/${isString(prioritized.consumer) ? prioritized.consumer : 'unknown-consumer'}`,
        count,
        threshold,
        thresholdDelta: count - threshold,
        thresholdRatioPercent: percentageFromThreshold(count, threshold),
        pendingCount: count,
        ackPendingCount: isNumber(prioritized.ack_pending_count) ? prioritized.ack_pending_count : 0,
        waitingCount: isNumber(prioritized.waiting_count) ? prioritized.waiting_count : 0,
        redeliveredCount: isNumber(prioritized.redelivered_count) ? prioritized.redelivered_count : 0,
        trend: 'unknown',
        operatorStatus: 'Consumer pressure observed in the latest MCP snapshot',
        latestSummary: 'Latest consumer pressure snapshot captured from MCP tool output.',
        latestCapturedAt: isString(result?.executed_at) ? result.executed_at : '',
        lastActive: isString(prioritized.last_active) ? prioritized.last_active : '',
        correlationHints: {},
    }
}