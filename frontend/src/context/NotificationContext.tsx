import React, { createContext, useCallback, useContext, useState } from 'react'

export interface BuildNotification {
    id: string
    buildId: string
    buildNumber: string
    message: string
    type: 'started' | 'completed' | 'failed' | 'cancelled'
    status: 'queued' | 'running' | 'completed' | 'failed' | 'cancelled' | 'in_progress' | 'success'
    timestamp: Date
}

interface NotificationContextType {
    notifications: BuildNotification[]
    addNotification: (notification: Omit<BuildNotification, 'id' | 'timestamp'>) => void
    removeNotification: (id: string) => void
    clearNotifications: () => void
}

const NotificationContext = createContext<NotificationContextType | undefined>(undefined)

export const NotificationProvider: React.FC<{ children: React.ReactNode }> = ({ children }) => {
    const [notifications, setNotifications] = useState<BuildNotification[]>([])

    const addNotification = useCallback(
        (notification: Omit<BuildNotification, 'id' | 'timestamp'>) => {
            const id = `${Date.now()}-${Math.random()}`
            const newNotification: BuildNotification = {
                ...notification,
                id,
                timestamp: new Date(),
            }
            setNotifications((prev) => [newNotification, ...prev])

            // Auto-remove notification after 5 seconds for non-error notifications
            if (notification.type !== 'failed') {
                setTimeout(() => {
                    removeNotification(id)
                }, 5000)
            } else {
                // Keep error notifications for 10 seconds
                setTimeout(() => {
                    removeNotification(id)
                }, 10000)
            }
        },
        []
    )

    const removeNotification = useCallback((id: string) => {
        setNotifications((prev) => prev.filter((n) => n.id !== id))
    }, [])

    const clearNotifications = useCallback(() => {
        setNotifications([])
    }, [])

    return (
        <NotificationContext.Provider
            value={{
                notifications,
                addNotification,
                removeNotification,
                clearNotifications,
            }}
        >
            {children}
        </NotificationContext.Provider>
    )
}

export const useNotifications = (): NotificationContextType => {
    const context = useContext(NotificationContext)
    if (!context) {
        throw new Error('useNotifications must be used within NotificationProvider')
    }
    return context
}
