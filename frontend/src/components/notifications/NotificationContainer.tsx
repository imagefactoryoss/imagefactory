import { BuildNotification, useNotifications } from '@/context/NotificationContext'
import { AlertCircle, CheckCircle, Clock, X } from 'lucide-react'
import React from 'react'

export const NotificationContainer: React.FC = () => {
    const { notifications, removeNotification } = useNotifications()

    const getIcon = (notification: BuildNotification) => {
        switch (notification.type) {
            case 'started':
                return <Clock className="w-5 h-5 text-blue-600 animate-spin" />
            case 'completed':
                return <CheckCircle className="w-5 h-5 text-green-600" />
            case 'failed':
                return <AlertCircle className="w-5 h-5 text-red-600" />
            case 'cancelled':
                return <X className="w-5 h-5 text-gray-600" />
            default:
                return <Clock className="w-5 h-5 text-gray-600" />
        }
    }

    const getBgColor = (notification: BuildNotification) => {
        switch (notification.type) {
            case 'started':
                return 'bg-blue-50 border-blue-200'
            case 'completed':
                return 'bg-green-50 border-green-200'
            case 'failed':
                return 'bg-red-50 border-red-200'
            case 'cancelled':
                return 'bg-gray-50 border-gray-200'
            default:
                return 'bg-gray-50 border-gray-200'
        }
    }

    const getTextColor = (notification: BuildNotification) => {
        switch (notification.type) {
            case 'started':
                return 'text-blue-900'
            case 'completed':
                return 'text-green-900'
            case 'failed':
                return 'text-red-900'
            case 'cancelled':
                return 'text-gray-900'
            default:
                return 'text-gray-900'
        }
    }

    return (
        <div className="fixed bottom-0 right-0 z-50 p-4 space-y-3 max-w-md">
            {notifications.map((notification) => (
                <div
                    key={notification.id}
                    className={`flex items-start gap-3 border rounded-lg p-4 shadow-lg ${getBgColor(
                        notification
                    )} animate-in fade-in slide-in-from-right-5 duration-300`}
                >
                    <div className="flex-shrink-0">{getIcon(notification)}</div>
                    <div className="flex-1 min-w-0">
                        <p className={`font-semibold ${getTextColor(notification)}`}>
                            Build #{notification.buildNumber}
                        </p>
                        <p className={`text-sm ${getTextColor(notification)}`}>{notification.message}</p>
                    </div>
                    <button
                        onClick={() => removeNotification(notification.id)}
                        className="flex-shrink-0 text-gray-400 hover:text-gray-600 transition-colors"
                    >
                        <X className="w-5 h-5" />
                    </button>
                </div>
            ))}
        </div>
    )
}

export default NotificationContainer
