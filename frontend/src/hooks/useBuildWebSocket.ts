import { useNotifications } from '@/context/NotificationContext'
import { useEffect, useRef } from 'react'

interface BuildEvent {
    type: 'build.started' | 'build.completed' | 'build.failed' | 'build.cancelled'
    buildId: string
    buildNumber: string
    projectId: string
    status: 'queued' | 'running' | 'completed' | 'failed' | 'cancelled' | 'in_progress' | 'success'
    message?: string
    duration?: number
    failureReason?: string
    gitBranch?: string
    gitCommit?: string
}

export const useBuildWebSocket = () => {
    const { addNotification } = useNotifications()
    const wsRef = useRef<WebSocket | null>(null)
    const reconnectTimeoutRef = useRef<NodeJS.Timeout>()
    const isConnectingRef = useRef(false)
    const failedAttemptsRef = useRef(0)
    const maxRetries = 3 // Stop retrying after 3 failed attempts

    useEffect(() => {
        const connectWebSocket = () => {
            // Stop retrying if we've exceeded max retries
            if (failedAttemptsRef.current >= maxRetries) {
                console.log('WebSocket max retries reached, stopping reconnection attempts')
                return
            }

            if (isConnectingRef.current || wsRef.current?.readyState === WebSocket.OPEN) {
                return
            }

            isConnectingRef.current = true

            try {
                const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
                const wsUrl = `${protocol}//${window.location.host}/api/builds/events`

                console.log(`Attempting to connect to WebSocket: ${wsUrl}`)
                const ws = new WebSocket(wsUrl)

                ws.onopen = () => {
                    console.log('WebSocket connected for build events')
                    isConnectingRef.current = false
                    failedAttemptsRef.current = 0 // Reset on successful connection
                }

                ws.onmessage = (event) => {
                    try {
                        const buildEvent: BuildEvent = JSON.parse(event.data)
                        handleBuildEvent(buildEvent)
                    } catch (error) {
                        console.error('Failed to parse WebSocket message:', error)
                    }
                }

                ws.onerror = (error) => {
                    console.warn('WebSocket error - endpoint may not be available yet:', error)
                    isConnectingRef.current = false
                }

                ws.onclose = () => {
                    console.log('WebSocket disconnected')
                    isConnectingRef.current = false
                    wsRef.current = null

                    // Increment failed attempts
                    failedAttemptsRef.current++

                    // Only retry if we haven't exceeded max retries
                    if (failedAttemptsRef.current < maxRetries) {
                        console.log(
                            `Will retry WebSocket connection in 5 seconds (attempt ${failedAttemptsRef.current}/${maxRetries})`
                        )
                        reconnectTimeoutRef.current = setTimeout(() => {
                            connectWebSocket()
                        }, 5000)
                    }
                }

                wsRef.current = ws
            } catch (error) {
                console.warn('Failed to create WebSocket connection:', error)
                isConnectingRef.current = false
                failedAttemptsRef.current++

                // Only retry if we haven't exceeded max retries
                if (failedAttemptsRef.current < maxRetries) {
                    console.log(
                        `Will retry WebSocket connection in 5 seconds (attempt ${failedAttemptsRef.current}/${maxRetries})`
                    )
                    reconnectTimeoutRef.current = setTimeout(() => {
                        connectWebSocket()
                    }, 5000)
                }
            }
        }

        const handleBuildEvent = (event: BuildEvent) => {
            switch (event.type) {
                case 'build.started':
                    addNotification({
                        buildId: event.buildId,
                        buildNumber: event.buildNumber,
                        message: `Build started on branch ${event.gitBranch || 'unknown'}`,
                        type: 'started',
                        status: 'running',
                    })
                    break

                case 'build.completed':
                    addNotification({
                        buildId: event.buildId,
                        buildNumber: event.buildNumber,
                        message: `Build completed in ${event.duration}s`,
                        type: 'completed',
                        status: 'completed',
                    })
                    break

                case 'build.failed':
                    addNotification({
                        buildId: event.buildId,
                        buildNumber: event.buildNumber,
                        message: event.failureReason || 'Build failed',
                        type: 'failed',
                        status: 'failed',
                    })
                    break

                case 'build.cancelled':
                    addNotification({
                        buildId: event.buildId,
                        buildNumber: event.buildNumber,
                        message: 'Build was cancelled',
                        type: 'cancelled',
                        status: 'cancelled',
                    })
                    break
            }
        }

        connectWebSocket()

        return () => {
            if (reconnectTimeoutRef.current) {
                clearTimeout(reconnectTimeoutRef.current)
            }
            if (wsRef.current) {
                wsRef.current.close()
                wsRef.current = null
            }
        }
    }, [addNotification])

    return wsRef.current
}

export default useBuildWebSocket
