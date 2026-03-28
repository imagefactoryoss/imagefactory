import { describe, expect, it } from 'vitest'

import { deriveAsyncBacklogInsight, deriveMessagingConsumerInsight, deriveMessagingTransportInsight, isAsyncBacklogIncidentType, isMessagingConsumerIncidentType } from '../sreSmartBotAsyncSummary'

describe('sreSmartBotAsyncSummary', () => {
    it('recognizes normalized async backlog incident types', () => {
        expect(isAsyncBacklogIncidentType('dispatcher_backlog_pressure')).toBe(true)
        expect(isAsyncBacklogIncidentType('email_queue_backlog_pressure')).toBe(true)
        expect(isAsyncBacklogIncidentType('messaging_outbox_backlog_pressure')).toBe(true)
        expect(isAsyncBacklogIncidentType('messaging_transport_degraded')).toBe(false)
    })

    it('recognizes normalized messaging consumer incident types', () => {
        expect(isMessagingConsumerIncidentType('nats_consumer_lag_pressure')).toBe(true)
        expect(isMessagingConsumerIncidentType('nats_consumer_stalled_progress')).toBe(true)
        expect(isMessagingConsumerIncidentType('nats_consumer_ack_pressure')).toBe(true)
        expect(isMessagingConsumerIncidentType('dispatcher_backlog_pressure')).toBe(false)
    })

    it('prefers workspace backlog summary over MCP payload fallback', () => {
        const insight = deriveAsyncBacklogInsight(
            'dispatcher_backlog_pressure',
            {
                async_pressure_summary: {
                    backlog: {
                        incident_type: 'dispatcher_backlog_pressure',
                        display_name: 'Dispatcher backlog',
                        queue_kind: 'build_queue',
                        subsystem: 'dispatcher',
                        count: 18,
                        threshold: 10,
                        threshold_delta: 8,
                        threshold_ratio_percent: 180,
                        trend: 'rising',
                        operator_status: 'Backlog remains above threshold',
                        latest_summary: 'Dispatcher backlog remained elevated for two samples.',
                        latest_captured_at: '2026-03-15T10:00:00Z',
                        recent_observations: {
                            previous: 14,
                            current: 18,
                        },
                        correlation_hints: {
                            messaging_transport: 'stable',
                        },
                    },
                },
            } as any,
            {
                payload: {
                    build_queue_depth: 99,
                    build_queue_threshold: 3,
                },
            } as any,
        )

        expect(insight).toMatchObject({
            source: 'workspace',
            count: 18,
            threshold: 10,
            trend: 'rising',
            latestSummary: 'Dispatcher backlog remained elevated for two samples.',
        })
    })

    it('falls back to the correct queue metric from MCP backlog output', () => {
        const insight = deriveAsyncBacklogInsight(
            'email_queue_backlog_pressure',
            null,
            {
                payload: {
                    build_queue_depth: 2,
                    build_queue_threshold: 5,
                    email_queue_depth: 8,
                    email_queue_threshold: 4,
                    messaging_outbox_pending: 1,
                    messaging_outbox_threshold: 3,
                    last_activity: '2026-03-15T10:05:00Z',
                },
            } as any,
        )

        expect(insight).toMatchObject({
            source: 'tool',
            displayName: 'Email queue backlog',
            count: 8,
            threshold: 4,
            thresholdDelta: 4,
            thresholdRatioPercent: 200,
            latestCapturedAt: '2026-03-15T10:05:00Z',
        })
        expect(insight?.recentObservations).toEqual({
            build_queue_depth: 2,
            email_queue_depth: 8,
            messaging_outbox_pending: 1,
        })
    })

    it('prefers workspace transport summary over MCP payload fallback', () => {
        const insight = deriveMessagingTransportInsight(
            {
                async_pressure_summary: {
                    messaging_transport: {
                        status: 'degraded',
                        reconnects: 6,
                        disconnects: 2,
                        reconnect_threshold: 3,
                        operator_status: 'Transport instability is likely contributing to backlog pressure',
                        latest_summary: 'Reconnect spikes preceded the outbox increase.',
                        latest_captured_at: '2026-03-15T10:10:00Z',
                    },
                },
            } as any,
            {
                payload: {
                    reconnects: 1,
                    disconnects: 0,
                    reconnect_threshold: 9,
                },
            } as any,
        )

        expect(insight).toMatchObject({
            source: 'workspace',
            reconnects: 6,
            disconnects: 2,
            reconnectThreshold: 3,
            latestSummary: 'Reconnect spikes preceded the outbox increase.',
        })
    })

    it('falls back to MCP transport output when no workspace summary exists', () => {
        const insight = deriveMessagingTransportInsight(
            null,
            {
                payload: {
                    reconnects: 4,
                    disconnects: 1,
                    reconnect_threshold: 2,
                    last_activity: '2026-03-15T10:12:00Z',
                },
            } as any,
        )

        expect(insight).toMatchObject({
            source: 'tool',
            status: 'degraded',
            reconnects: 4,
            disconnects: 1,
            reconnectThreshold: 2,
            latestCapturedAt: '2026-03-15T10:12:00Z',
        })
    })

    it('prefers workspace consumer summary over MCP payload fallback', () => {
        const insight = deriveMessagingConsumerInsight(
            'nats_consumer_lag_pressure',
            {
                async_pressure_summary: {
                    messaging_consumer: {
                        incident_type: 'nats_consumer_lag_pressure',
                        display_name: 'NATS consumer lag pressure for build-events/dispatcher',
                        kind: 'consumer_lag_pressure',
                        stream: 'build-events',
                        consumer: 'dispatcher',
                        target_ref: 'build-events/dispatcher',
                        count: 42,
                        threshold: 25,
                        threshold_delta: 17,
                        threshold_ratio_percent: 168,
                        pending_count: 42,
                        ack_pending_count: 4,
                        waiting_count: 1,
                        redelivered_count: 0,
                        trend: 'growing',
                        operator_status: 'consumer lag growing',
                        latest_summary: 'Dispatcher lag is rising without current transport instability.',
                        latest_captured_at: '2026-03-15T10:15:00Z',
                        last_active: '2026-03-15T10:10:00Z',
                        correlation_hints: {
                            messaging_consumers_tool: 'messaging_consumers.recent',
                        },
                    },
                },
            } as any,
            {
                payload: {
                    count: 99,
                    lagging_count: 5,
                },
            } as any,
        )

        expect(insight).toMatchObject({
            source: 'workspace',
            count: 42,
            threshold: 25,
            trend: 'growing',
            operatorStatus: 'consumer lag growing',
            latestSummary: 'Dispatcher lag is rising without current transport instability.',
        })
    })

    it('falls back to MCP messaging consumer output when no workspace summary exists', () => {
        const insight = deriveMessagingConsumerInsight(
            'nats_consumer_lag_pressure',
            null,
            {
                executed_at: '2026-03-15T10:17:00Z',
                payload: {
                    max_pending_count: 42,
                    consumers: [
                        {
                            stream: 'build-events',
                            consumer: 'dispatcher',
                            pending_count: 42,
                            ack_pending_count: 4,
                            waiting_count: 1,
                            redelivered_count: 0,
                            last_active: '2026-03-15T10:11:00Z',
                        },
                    ],
                },
            } as any,
        )

        expect(insight).toMatchObject({
            source: 'tool',
            stream: 'build-events',
            consumer: 'dispatcher',
            count: 42,
            threshold: 42,
            latestCapturedAt: '2026-03-15T10:17:00Z',
        })
    })
})