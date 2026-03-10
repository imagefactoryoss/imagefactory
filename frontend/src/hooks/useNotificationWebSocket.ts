import { useAuthStore } from '@/store/auth'
import { useTenantStore } from '@/store/tenant'
import { useEffect, useRef, useState } from 'react'

export interface NotificationSocketEvent {
  type?: string
  notification_id?: string
  timestamp?: string
  metadata?: Record<string, any>
}

interface UseNotificationWebSocketOptions {
  enabled?: boolean
  reconnectDelayMs?: number
  onEvent?: (event: NotificationSocketEvent) => void
}

export const useNotificationWebSocket = (options: UseNotificationWebSocketOptions = {}) => {
  const { enabled = true, reconnectDelayMs = 4000, onEvent } = options
  const token = useAuthStore((state) => state.token)
  const selectedTenantId = useTenantStore((state) => state.selectedTenantId)
  const [isConnected, setIsConnected] = useState(false)

  const wsRef = useRef<WebSocket | null>(null)
  const reconnectTimerRef = useRef<number | undefined>(undefined)
  const onEventRef = useRef<typeof onEvent>(onEvent)

  useEffect(() => {
    onEventRef.current = onEvent
  }, [onEvent])

  useEffect(() => {
    if (!enabled || !token || !selectedTenantId) return

    let active = true

    const connect = () => {
      if (!active) return
      if (wsRef.current && (wsRef.current.readyState === WebSocket.OPEN || wsRef.current.readyState === WebSocket.CONNECTING)) {
        return
      }

      const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
      const wsUrl = `${protocol}//${window.location.host}/api/notifications/events?token=${encodeURIComponent(token)}&tenant_id=${encodeURIComponent(selectedTenantId)}`
      const ws = new WebSocket(wsUrl)
      wsRef.current = ws

      ws.onopen = () => {
        if (!active) return
        setIsConnected(true)
      }

      ws.onclose = (event) => {
        if (!active) return
        if (wsRef.current === ws) {
          wsRef.current = null
        }
        setIsConnected(false)
        if (event.code === 1000 || event.code === 1001 || event.code === 1008) {
          return
        }
        reconnectTimerRef.current = window.setTimeout(connect, reconnectDelayMs)
      }

      ws.onerror = () => {
        if (!active) return
        setIsConnected(false)
      }

      ws.onmessage = (messageEvent) => {
        if (!active) return
        try {
          const payload: NotificationSocketEvent = JSON.parse(messageEvent.data)
          onEventRef.current?.(payload)
        } catch {
          // Ignore malformed payload
        }
      }
    }

    connect()

    return () => {
      active = false
      setIsConnected(false)
      if (reconnectTimerRef.current) {
        window.clearTimeout(reconnectTimerRef.current)
        reconnectTimerRef.current = undefined
      }
      if (wsRef.current) {
        wsRef.current.close()
        wsRef.current = null
      }
    }
  }, [enabled, reconnectDelayMs, selectedTenantId, token])

  return { isConnected }
}

export default useNotificationWebSocket
