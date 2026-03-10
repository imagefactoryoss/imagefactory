import { notificationService, type UserNotification } from '@/services/notificationService'
import { CheckCheck, Trash2 } from 'lucide-react'
import React, { useCallback, useEffect, useMemo, useState } from 'react'
import toast from 'react-hot-toast'
import { Link, useNavigate } from 'react-router-dom'

const PAGE_SIZE = 25

function formatNotificationTime(value: string): string {
  const parsed = new Date(value)
  if (Number.isNaN(parsed.getTime())) {
    return ''
  }
  return parsed.toLocaleString()
}

const NotificationsPage: React.FC = () => {
  const navigate = useNavigate()
  const [loading, setLoading] = useState(true)
  const [markingAll, setMarkingAll] = useState(false)
  const [deletingRead, setDeletingRead] = useState(false)
  const [deletingIds, setDeletingIds] = useState<Record<string, boolean>>({})
  const [markingIds, setMarkingIds] = useState<Record<string, boolean>>({})
  const [notifications, setNotifications] = useState<UserNotification[]>([])
  const [selectedIds, setSelectedIds] = useState<Record<string, boolean>>({})
  const [totalCount, setTotalCount] = useState(0)
  const [offset, setOffset] = useState(0)
  const [showUnreadOnly, setShowUnreadOnly] = useState(false)
  const [bulkDeleting, setBulkDeleting] = useState(false)

  const loadNotifications = useCallback(async () => {
    setLoading(true)
    try {
      const data = await notificationService.list({
        limit: PAGE_SIZE,
        offset,
        unread: showUnreadOnly || undefined,
      })
      setNotifications(data.notifications ?? [])
      setTotalCount(data.total_count ?? 0)
      setSelectedIds({})
    } catch {
      toast.error('Failed to load notifications')
    } finally {
      setLoading(false)
    }
  }, [offset, showUnreadOnly])

  useEffect(() => {
    void loadNotifications()
  }, [loadNotifications])

  const currentPage = Math.floor(offset / PAGE_SIZE) + 1
  const totalPages = Math.max(1, Math.ceil(totalCount / PAGE_SIZE))
  const hasPrev = offset > 0
  const hasNext = offset + PAGE_SIZE < totalCount

  const unreadCountOnPage = useMemo(
    () => notifications.reduce((count, item) => (item.is_read ? count : count + 1), 0),
    [notifications],
  )
  const readCountOnPage = useMemo(
    () => notifications.reduce((count, item) => (item.is_read ? count + 1 : count), 0),
    [notifications],
  )
  const selectedCount = useMemo(
    () => notifications.reduce((count, item) => (selectedIds[item.id] ? count + 1 : count), 0),
    [notifications, selectedIds],
  )
  const allOnPageSelected = notifications.length > 0 && selectedCount === notifications.length

  const handleMarkAsRead = useCallback(async (item: UserNotification) => {
    if (item.is_read) {
      return
    }
    setMarkingIds((prev) => ({ ...prev, [item.id]: true }))
    try {
      await notificationService.markAsRead(item.id)
      setNotifications((prev) =>
        prev.map((existing) =>
          existing.id === item.id
            ? { ...existing, is_read: true, read_at: existing.read_at ?? new Date().toISOString() }
            : existing,
        ),
      )
    } catch {
      toast.error('Failed to mark notification as read')
    } finally {
      setMarkingIds((prev) => {
        const next = { ...prev }
        delete next[item.id]
        return next
      })
    }
  }, [])

  const handleDeleteOne = useCallback(async (notificationId: string) => {
    setDeletingIds((prev) => ({ ...prev, [notificationId]: true }))
    try {
      await notificationService.deleteOne(notificationId)
      setNotifications((prev) => prev.filter((item) => item.id !== notificationId))
      setTotalCount((prev) => Math.max(0, prev - 1))
      if (offset > 0 && notifications.length === 1) {
        setOffset((prev) => Math.max(0, prev - PAGE_SIZE))
      }
    } catch {
      toast.error('Failed to delete notification')
    } finally {
      setDeletingIds((prev) => {
        const next = { ...prev }
        delete next[notificationId]
        return next
      })
    }
  }, [notifications.length, offset])

  const handleMarkAll = useCallback(async () => {
    setMarkingAll(true)
    try {
      const result = await notificationService.markAllAsRead()
      if ((result.updated ?? 0) > 0) {
        setNotifications((prev) => prev.map((item) => ({ ...item, is_read: true })))
      }
    } catch {
      toast.error('Failed to mark notifications as read')
    } finally {
      setMarkingAll(false)
    }
  }, [])

  const handleDeleteRead = useCallback(async () => {
    setDeletingRead(true)
    try {
      const result = await notificationService.deleteRead()
      const deleted = result.deleted ?? 0
      if (deleted <= 0) {
        return
      }
      setNotifications((prev) => prev.filter((item) => !item.is_read))
      setTotalCount((prev) => Math.max(0, prev - deleted))
      if (offset > 0 && notifications.length <= deleted) {
        setOffset((prev) => Math.max(0, prev - PAGE_SIZE))
      }
    } catch {
      toast.error('Failed to delete read notifications')
    } finally {
      setDeletingRead(false)
    }
  }, [notifications.length, offset])

  const handleToggleSelect = (notificationID: string) => {
    setSelectedIds((prev) => ({ ...prev, [notificationID]: !prev[notificationID] }))
  }

  const handleToggleSelectAllOnPage = () => {
    if (notifications.length === 0) {
      return
    }
    setSelectedIds((prev) => {
      const next = { ...prev }
      const selectAll = notifications.some((item) => !next[item.id])
      for (const item of notifications) {
        next[item.id] = selectAll
      }
      return next
    })
  }

  const handleDeleteSelected = useCallback(async () => {
    const idsToDelete = notifications.filter((item) => selectedIds[item.id]).map((item) => item.id)
    if (idsToDelete.length === 0) {
      return
    }
    setBulkDeleting(true)
    try {
      const result = await notificationService.deleteBulk(idsToDelete)
      const deleted = result.deleted ?? 0
      setNotifications((prev) => prev.filter((item) => !selectedIds[item.id]))
      setSelectedIds({})
      setTotalCount((prev) => Math.max(0, prev - deleted))
      if (offset > 0 && notifications.length <= deleted) {
        setOffset((prev) => Math.max(0, prev - PAGE_SIZE))
      }
      toast.success(`Deleted ${deleted} notification${deleted === 1 ? '' : 's'}`)
    } catch {
      toast.error('Failed to delete selected notifications')
    } finally {
      setBulkDeleting(false)
    }
  }, [notifications, offset, selectedIds])

  const handleFilterToggle = (next: boolean) => {
    setShowUnreadOnly(next)
    setOffset(0)
  }

  return (
    <div className="min-h-screen bg-slate-50 dark:bg-slate-900 px-4 py-6 sm:px-6 lg:px-8">
      <div className="sm:flex sm:items-center sm:justify-between">
        <div>
          <h1 className="text-2xl font-semibold text-slate-900 dark:text-white">Notifications</h1>
          <p className="mt-2 text-sm text-slate-700 dark:text-slate-400">
            Review, mark, and delete your in-app notifications.
          </p>
        </div>
        <Link
          to="/dashboard"
          className="mt-4 sm:mt-0 inline-flex items-center rounded-md border border-slate-300 dark:border-slate-600 px-3 py-2 text-sm text-slate-700 dark:text-slate-200 hover:bg-slate-100 dark:hover:bg-slate-800"
        >
          Back to dashboard
        </Link>
      </div>

      <div className="mt-6 rounded-xl border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800 shadow-sm">
        <div className="flex flex-wrap items-center justify-between gap-2 px-4 py-3 border-b border-slate-200 dark:border-slate-700">
          <div className="flex items-center gap-2">
            <button
              type="button"
              onClick={() => handleFilterToggle(false)}
              className={`px-3 py-1.5 text-xs rounded-md border ${
                !showUnreadOnly
                  ? 'bg-blue-600 border-blue-600 text-white'
                  : 'border-slate-300 dark:border-slate-600 text-slate-700 dark:text-slate-200 hover:bg-slate-100 dark:hover:bg-slate-700'
              }`}
            >
              All
            </button>
            <button
              type="button"
              onClick={() => handleFilterToggle(true)}
              className={`px-3 py-1.5 text-xs rounded-md border ${
                showUnreadOnly
                  ? 'bg-blue-600 border-blue-600 text-white'
                  : 'border-slate-300 dark:border-slate-600 text-slate-700 dark:text-slate-200 hover:bg-slate-100 dark:hover:bg-slate-700'
              }`}
            >
              Unread
            </button>
          </div>
          <div className="flex flex-wrap items-center gap-2">
            <span className="text-xs text-slate-500 dark:text-slate-400">
              {totalCount} total, {unreadCountOnPage} unread on this page
            </span>
            <button
              type="button"
              onClick={handleDeleteSelected}
              disabled={bulkDeleting || selectedCount === 0}
              className="inline-flex items-center gap-1 text-xs px-2 py-1 rounded-md border border-slate-300 dark:border-slate-600 text-slate-700 dark:text-slate-200 hover:bg-slate-100 dark:hover:bg-slate-700 disabled:opacity-50 disabled:cursor-not-allowed"
            >
              <Trash2 className="w-3.5 h-3.5" />
              Delete selected ({selectedCount})
            </button>
            <button
              type="button"
              onClick={handleDeleteRead}
              disabled={deletingRead || readCountOnPage === 0}
              className="inline-flex items-center gap-1 text-xs px-2 py-1 rounded-md border border-slate-300 dark:border-slate-600 text-slate-700 dark:text-slate-200 hover:bg-slate-100 dark:hover:bg-slate-700 disabled:opacity-50 disabled:cursor-not-allowed"
            >
              <Trash2 className="w-3.5 h-3.5" />
              Delete read
            </button>
            <button
              type="button"
              onClick={handleMarkAll}
              disabled={markingAll || unreadCountOnPage === 0}
              className="inline-flex items-center gap-1 text-xs px-2 py-1 rounded-md border border-slate-300 dark:border-slate-600 text-slate-700 dark:text-slate-200 hover:bg-slate-100 dark:hover:bg-slate-700 disabled:opacity-50 disabled:cursor-not-allowed"
            >
              <CheckCheck className="w-3.5 h-3.5" />
              Mark all read
            </button>
          </div>
        </div>

        {loading ? (
          <div className="px-4 py-6 text-sm text-slate-500 dark:text-slate-400">Loading notifications...</div>
        ) : notifications.length === 0 ? (
          <div className="px-4 py-6 text-sm text-slate-500 dark:text-slate-400">No notifications found.</div>
        ) : (
          <>
            <div className="px-4 py-2 border-b border-slate-200 dark:border-slate-700 bg-slate-50 dark:bg-slate-800/60">
              <label className="inline-flex items-center gap-2 text-xs text-slate-700 dark:text-slate-200">
                <input
                  type="checkbox"
                  checked={allOnPageSelected}
                  onChange={handleToggleSelectAllOnPage}
                  className="h-4 w-4 rounded border-slate-300 dark:border-slate-600 bg-white dark:bg-slate-900 text-blue-600 focus:ring-blue-500"
                />
                Select all on page
              </label>
            </div>
            <ul className="divide-y divide-slate-100 dark:divide-slate-700">
              {notifications.map((item) => (
                <li key={item.id} className="px-4 py-3">
                  <div className="flex items-start justify-between gap-3">
                    <div className="min-w-0 flex items-start gap-3">
                      <input
                        type="checkbox"
                        checked={Boolean(selectedIds[item.id])}
                        onChange={() => handleToggleSelect(item.id)}
                        className="mt-0.5 h-4 w-4 rounded border-slate-300 dark:border-slate-600 bg-white dark:bg-slate-900 text-blue-600 focus:ring-blue-500"
                        aria-label="Select notification"
                      />
                      <button
                        type="button"
                        onClick={() => {
                          if (item.related_resource_type === 'build' && item.related_resource_id) {
                            navigate(`/builds/${item.related_resource_id}`)
                          }
                        }}
                        className="text-left min-w-0"
                      >
                        <p className="text-sm font-medium text-slate-900 dark:text-slate-100 break-words">
                          {item.title || 'Notification'}
                        </p>
                        <p className="text-xs text-slate-600 dark:text-slate-300 mt-1 break-words">
                          {item.message || 'No message'}
                        </p>
                        <p className="text-[11px] text-slate-500 dark:text-slate-400 mt-1">
                          {formatNotificationTime(item.created_at)}
                        </p>
                      </button>
                    </div>
                    <div className="flex items-center gap-2 shrink-0">
                      {!item.is_read && <span className="inline-block w-2 h-2 rounded-full bg-blue-600 dark:bg-blue-400" />}
                      {!item.is_read && (
                        <button
                          type="button"
                          onClick={() => void handleMarkAsRead(item)}
                          disabled={Boolean(markingIds[item.id])}
                          className="inline-flex items-center rounded-md border border-slate-300 dark:border-slate-600 px-2 py-1 text-[11px] text-slate-700 dark:text-slate-200 hover:bg-slate-100 dark:hover:bg-slate-700 disabled:opacity-50 disabled:cursor-not-allowed"
                        >
                          Mark read
                        </button>
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
                </li>
              ))}
            </ul>
          </>
        )}

        <div className="flex items-center justify-between px-4 py-3 border-t border-slate-200 dark:border-slate-700">
          <p className="text-xs text-slate-500 dark:text-slate-400">
            Page {currentPage} of {totalPages}
          </p>
          <div className="flex items-center gap-2">
            <button
              type="button"
              onClick={() => setOffset((prev) => Math.max(0, prev - PAGE_SIZE))}
              disabled={!hasPrev}
              className="px-2 py-1 text-xs rounded-md border border-slate-300 dark:border-slate-600 text-slate-700 dark:text-slate-200 hover:bg-slate-100 dark:hover:bg-slate-700 disabled:opacity-50 disabled:cursor-not-allowed"
            >
              Previous
            </button>
            <button
              type="button"
              onClick={() => setOffset((prev) => prev + PAGE_SIZE)}
              disabled={!hasNext}
              className="px-2 py-1 text-xs rounded-md border border-slate-300 dark:border-slate-600 text-slate-700 dark:text-slate-200 hover:bg-slate-100 dark:hover:bg-slate-700 disabled:opacity-50 disabled:cursor-not-allowed"
            >
              Next
            </button>
          </div>
        </div>
      </div>
    </div>
  )
}

export default NotificationsPage
