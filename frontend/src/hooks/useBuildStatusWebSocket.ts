import { useAuthStore } from '@/store/auth'
import { useTenantStore } from '@/store/tenant'
import { useEffect, useRef, useState } from 'react'

const connectionCooldownUntilByKey = new Map<string, number>()

export interface BuildStatusEvent {
    type?: string
    build_id?: string
    buildId?: string
    project_id?: string
    projectId?: string
    status?: string
    [key: string]: any
}

interface UseBuildStatusWebSocketOptions {
    enabled?: boolean
    filterBuildId?: string
    filterProjectId?: string
    reconnectDelayMs?: number
    onBuildEvent?: (event: BuildStatusEvent) => void
}

export const useBuildStatusWebSocket = (options: UseBuildStatusWebSocketOptions = {}) => {
    const {
        enabled = true,
        filterBuildId,
        filterProjectId,
        reconnectDelayMs = 4000,
        onBuildEvent,
    } = options

    const token = useAuthStore((state) => state.token)
    const selectedTenantId = useTenantStore((state) => state.selectedTenantId)
    const [isConnected, setIsConnected] = useState(false)

    const wsRef = useRef<WebSocket | null>(null)
    const reconnectTimerRef = useRef<number | undefined>(undefined)
    const onBuildEventRef = useRef<typeof onBuildEvent>(onBuildEvent)
    const reconnectingRef = useRef(false)
    const lastConnectAttemptAtRef = useRef(0)
    const reconnectAttemptRef = useRef(0)
    const endpointAttemptRef = useRef(0)

    useEffect(() => {
        onBuildEventRef.current = onBuildEvent
    }, [onBuildEvent])

    useEffect(() => {
        if (!enabled || !token || !selectedTenantId) return

        let active = true
        reconnectingRef.current = false
        reconnectAttemptRef.current = 0
        endpointAttemptRef.current = 0
        const connectionKey = `${selectedTenantId}:${filterBuildId || ''}:${filterProjectId || ''}`

        const connect = () => {
            if (!active) return
            const existing = wsRef.current
            if (existing && (existing.readyState === WebSocket.OPEN || existing.readyState === WebSocket.CONNECTING)) {
                return
            }

            const now = Date.now()
            const cooldownUntil = connectionCooldownUntilByKey.get(connectionKey) || 0
            if (cooldownUntil > now) {
                if (!reconnectingRef.current) {
                    reconnectingRef.current = true
                    reconnectTimerRef.current = window.setTimeout(() => {
                        reconnectingRef.current = false
                        connect()
                    }, cooldownUntil - now)
                }
                return
            }

            const minReconnectIntervalMs = Math.max(1000, reconnectDelayMs)
            if (lastConnectAttemptAtRef.current > 0 && now - lastConnectAttemptAtRef.current < minReconnectIntervalMs) {
                if (!reconnectingRef.current) {
                    reconnectingRef.current = true
                    const wait = minReconnectIntervalMs - (now - lastConnectAttemptAtRef.current)
                    reconnectTimerRef.current = window.setTimeout(() => {
                        reconnectingRef.current = false
                        connect()
                    }, wait)
                }
                return
            }
            lastConnectAttemptAtRef.current = now

            const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
            const endpointCandidates = ['/api/builds/events', '/api/v1/builds/events']
            const endpoint = endpointCandidates[endpointAttemptRef.current % endpointCandidates.length]
            const wsUrl = `${protocol}//${window.location.host}${endpoint}?token=${encodeURIComponent(token)}&tenant_id=${encodeURIComponent(selectedTenantId)}`
            const ws = new WebSocket(wsUrl)
            wsRef.current = ws

            ws.onopen = () => {
                if (!active) return
                reconnectAttemptRef.current = 0
                endpointAttemptRef.current = 0
                connectionCooldownUntilByKey.delete(connectionKey)
                setIsConnected(true)
            }

            ws.onclose = (event) => {
                if (!active) return
                if (wsRef.current === ws) {
                    wsRef.current = null
                }
                setIsConnected(false)

                // Do not reconnect for policy/auth closes.
                // Normal closes (1000/1001) can happen during rolling updates and should reconnect.
                if (event.code === 1008 || event.code === 4001 || event.code === 4401 || event.code === 4403) {
                    reconnectingRef.current = false
                    connectionCooldownUntilByKey.set(connectionKey, Date.now() + 30000)
                    return
                }

                if (reconnectingRef.current) return
                reconnectingRef.current = true
                reconnectAttemptRef.current += 1
                endpointAttemptRef.current += 1
                const nextDelayMs = Math.min(30000, reconnectDelayMs * Math.max(1, 2 ** (reconnectAttemptRef.current - 1)))
                connectionCooldownUntilByKey.set(connectionKey, Date.now() + nextDelayMs)
                reconnectTimerRef.current = window.setTimeout(() => {
                    reconnectingRef.current = false
                    connect()
                }, nextDelayMs)
            }

            ws.onerror = () => {
                if (!active) return
                setIsConnected(false)
            }

            ws.onmessage = (messageEvent) => {
                if (!active) return
                try {
                    const event: BuildStatusEvent = JSON.parse(messageEvent.data)
                    const eventBuildID = event.build_id || event.buildId
                    const eventProjectID = event.project_id || event.projectId

                    if (filterBuildId && eventBuildID !== filterBuildId) return
                    if (filterProjectId && eventProjectID !== filterProjectId) return

                    onBuildEventRef.current?.(event)
                } catch {
                    // Ignore malformed payload
                }
            }
        }

        connect()

        return () => {
            active = false
            setIsConnected(false)
            reconnectingRef.current = false
            if (reconnectTimerRef.current) {
                window.clearTimeout(reconnectTimerRef.current)
                reconnectTimerRef.current = undefined
            }
            if (wsRef.current) {
                wsRef.current.close()
                wsRef.current = null
            }
        }
    }, [enabled, filterBuildId, filterProjectId, reconnectDelayMs, selectedTenantId, token])

    return { isConnected }
}

export default useBuildStatusWebSocket
