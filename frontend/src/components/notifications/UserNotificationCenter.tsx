import { notificationService, type UserNotification } from '@/services/notificationService'
import useNotificationWebSocket from '@/hooks/useNotificationWebSocket'
import { Bell, CheckCheck, Trash2 } from 'lucide-react'
import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { useNavigate } from 'react-router-dom'

function formatNotificationTime(value: string): string {
  const parsed = new Date(value)
  if (Number.isNaN(parsed.getTime())) {
    return ''
  }
  return parsed.toLocaleString()
}

const UserNotificationCenter: React.FC = () => {
  const navigate = useNavigate()
  const [open, setOpen] = useState(false)
  const [loading, setLoading] = useState(false)
  const [markingAll, setMarkingAll] = useState(false)
  const [deletingRead, setDeletingRead] = useState(false)
  const [deletingIds, setDeletingIds] = useState<Record<string, boolean>>({})
  const [unreadCount, setUnreadCount] = useState(0)
  const [notifications, setNotifications] = useState<UserNotification[]>([])
  const rootRef = useRef<HTMLDivElement | null>(null)
  const refreshTimerRef = useRef<number | null>(null)

  const refreshUnreadCount = useCallback(async () => {
    try {
      const data = await notificationService.getUnreadCount()
      setUnreadCount(data.unread_count ?? 0)
    } catch {
      // Keep badge state stable on transient errors.
    }
  }, [])

  const refreshList = useCallback(async () => {
    setLoading(true)
    try {
      const data = await notificationService.list({ limit: 10, offset: 0 })
      setNotifications(data.notifications ?? [])
    } finally {
      setLoading(false)
    }
  }, [])

  const scheduleRefreshFromEvent = useCallback(() => {
    if (refreshTimerRef.current) {
      window.clearTimeout(refreshTimerRef.current)
    }
    refreshTimerRef.current = window.setTimeout(() => {
      void refreshUnreadCount()
      if (open) {
        void refreshList()
      }
    }, 300)
  }, [open, refreshList, refreshUnreadCount])

  useEffect(() => {
    void refreshUnreadCount()
  }, [refreshUnreadCount])

  useNotificationWebSocket({
    enabled: true,
    onEvent: () => {
      scheduleRefreshFromEvent()
    },
  })

  useEffect(() => {
    if (!open) {
      return
    }
    refreshList()
  }, [open, refreshList])

  useEffect(() => {
    const onDocMouseDown = (event: MouseEvent) => {
      if (!rootRef.current) {
        return
      }
      if (!rootRef.current.contains(event.target as Node)) {
        setOpen(false)
      }
    }
    if (open) {
      document.addEventListener('mousedown', onDocMouseDown)
    }
    return () => {
      document.removeEventListener('mousedown', onDocMouseDown)
    }
  }, [open])

  useEffect(() => {
    return () => {
      if (refreshTimerRef.current) {
        window.clearTimeout(refreshTimerRef.current)
      }
    }
  }, [])

  const unreadBadge = useMemo(() => {
    if (unreadCount <= 0) {
      return null
    }
    const label = unreadCount > 99 ? '99+' : String(unreadCount)
    return (
      <span className="absolute -top-1 -right-1 min-w-[18px] h-[18px] px-1 rounded-full bg-red-600 dark:bg-red-500 text-white text-[10px] leading-[18px] text-center font-semibold border border-white dark:border-slate-800">
        {label}
      </span>
    )
  }, [unreadCount])

  const handleMarkAsRead = useCallback(
    async (item: UserNotification) => {
      if (item.is_read) {
        return
      }
      await notificationService.markAsRead(item.id)
      setNotifications((prev) =>
        prev.map((existing) =>
          existing.id === item.id
            ? { ...existing, is_read: true, read_at: existing.read_at ?? new Date().toISOString() }
            : existing,
        ),
      )
      setUnreadCount((prev) => Math.max(0, prev - 1))
    },
    [],
  )

  const handleOpenItem = useCallback(
    async (item: UserNotification) => {
      await handleMarkAsRead(item)
      setOpen(false)

      if (item.related_resource_type === 'build' && item.related_resource_id) {
        navigate(`/builds/${item.related_resource_id}`)
      }
    },
    [handleMarkAsRead, navigate],
  )

  const handleMarkAll = useCallback(async () => {
    setMarkingAll(true)
    try {
      await notificationService.markAllAsRead()
      setNotifications((prev) => prev.map((item) => ({ ...item, is_read: true })))
      setUnreadCount(0)
    } finally {
      setMarkingAll(false)
    }
  }, [])

  const handleDeleteRead = useCallback(async () => {
    setDeletingRead(true)
    try {
      const result = await notificationService.deleteRead()
      if ((result.deleted ?? 0) <= 0) {
        return
      }
      setNotifications((prev) => prev.filter((item) => !item.is_read))
    } finally {
      setDeletingRead(false)
    }
  }, [])

  const handleDeleteOne = useCallback(async (notificationId: string) => {
    if (!notificationId) {
      return
    }
    setDeletingIds((prev) => ({ ...prev, [notificationId]: true }))
    try {
      await notificationService.deleteOne(notificationId)
      setNotifications((prev) => {
        const target = prev.find((item) => item.id === notificationId)
        if (target && !target.is_read) {
          setUnreadCount((existing) => Math.max(0, existing - 1))
        }
        return prev.filter((item) => item.id !== notificationId)
      })
    } finally {
      setDeletingIds((prev) => {
        const next = { ...prev }
        delete next[notificationId]
        return next
      })
    }
  }, [])

  const readCount = useMemo(
    () => notifications.reduce((count, item) => (item.is_read ? count + 1 : count), 0),
    [notifications],
  )

  return (
    <div className="relative" ref={rootRef}>
      <button
        type="button"
        onClick={() => setOpen((prev) => !prev)}
        className="relative p-1 rounded-full hover:bg-slate-100 dark:hover:bg-slate-700 transition-colors text-slate-700 dark:text-slate-300 hover:text-slate-900 dark:hover:text-slate-100"
        title="Notifications"
        aria-label="Open notifications"
      >
        <Bell className="w-5 h-5" />
        {unreadBadge}
      </button>

      {open && (
        <div className="absolute right-0 mt-2 w-[360px] max-w-[90vw] rounded-xl border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-900 shadow-xl z-50">
          <div className="flex items-center justify-between px-3 py-2 border-b border-slate-200 dark:border-slate-700">
            <div>
              <h3 className="text-sm font-semibold text-slate-900 dark:text-slate-100">Notifications</h3>
              <p className="text-xs text-slate-500 dark:text-slate-400">{unreadCount} unread</p>
            </div>
            <div className="flex items-center gap-2">
              <button
                type="button"
                onClick={handleDeleteRead}
                disabled={deletingRead || readCount === 0}
                className="inline-flex items-center gap-1 text-xs px-2 py-1 rounded-md border border-slate-200 dark:border-slate-600 text-slate-700 dark:text-slate-200 hover:bg-slate-100 dark:hover:bg-slate-800 disabled:opacity-50 disabled:cursor-not-allowed"
              >
                <Trash2 className="w-3.5 h-3.5" />
                Delete read
              </button>
              <button
                type="button"
                onClick={handleMarkAll}
                disabled={markingAll || unreadCount === 0}
                className="inline-flex items-center gap-1 text-xs px-2 py-1 rounded-md border border-slate-200 dark:border-slate-600 text-slate-700 dark:text-slate-200 hover:bg-slate-100 dark:hover:bg-slate-800 disabled:opacity-50 disabled:cursor-not-allowed"
              >
                <CheckCheck className="w-3.5 h-3.5" />
                Mark all read
              </button>
            </div>
          </div>

          <div className="max-h-[420px] overflow-y-auto">
            {loading ? (
              <div className="px-4 py-6 text-sm text-slate-500 dark:text-slate-400">Loading notifications...</div>
            ) : notifications.length === 0 ? (
              <div className="px-4 py-6 text-sm text-slate-500 dark:text-slate-400">No notifications yet.</div>
            ) : (
              <ul className="divide-y divide-slate-100 dark:divide-slate-800">
                {notifications.map((item) => (
                  <li key={item.id}>
                    <div
                      className={`w-full px-3 py-2.5 transition-colors ${
                        item.is_read
                          ? 'bg-white dark:bg-slate-900 hover:bg-slate-50 dark:hover:bg-slate-800/80'
                          : 'bg-blue-50/70 dark:bg-blue-950/30 hover:bg-blue-100/70 dark:hover:bg-blue-900/30'
                      }`}
                    >
                      <div className="flex items-start justify-between gap-3">
                        <button
                          type="button"
                          onClick={() => void handleOpenItem(item)}
                          className="min-w-0 text-left flex-1"
                        >
                          <p className="text-sm font-medium text-slate-900 dark:text-slate-100 truncate">
                            {item.title || 'Notification'}
                          </p>
                          <p className="text-xs text-slate-600 dark:text-slate-300 mt-0.5 break-words">
                            {item.message || 'No message'}
                          </p>
                          <p className="text-[11px] text-slate-500 dark:text-slate-400 mt-1">
                            {formatNotificationTime(item.created_at)}
                          </p>
                        </button>
                        <div className="flex items-center gap-2 shrink-0">
                          {!item.is_read && (
                            <span className="inline-block w-2 h-2 rounded-full bg-blue-600 dark:bg-blue-400" />
                          )}
                          <button
                            type="button"
                            onClick={() => void handleDeleteOne(item.id)}
                            disabled={Boolean(deletingIds[item.id])}
                            className="inline-flex items-center justify-center p-1 rounded-md text-slate-500 dark:text-slate-300 hover:bg-slate-200 dark:hover:bg-slate-700 disabled:opacity-50 disabled:cursor-not-allowed"
                            title="Delete notification"
                            aria-label="Delete notification"
                          >
                            <Trash2 className="w-3.5 h-3.5" />
                          </button>
                        </div>
                      </div>
                    </div>
                  </li>
                ))}
              </ul>
            )}
          </div>
          <div className="px-3 py-2 border-t border-slate-200 dark:border-slate-700 bg-slate-50 dark:bg-slate-800/60">
            <button
              type="button"
              onClick={() => {
                setOpen(false)
                navigate('/notifications')
              }}
              className="w-full text-center text-xs font-medium text-blue-700 dark:text-blue-300 hover:text-blue-800 dark:hover:text-blue-200"
            >
              View all notifications
            </button>
          </div>
        </div>
      )}
    </div>
  )
}

export default UserNotificationCenter
