/**
 * Empty state component for list pages
 */

import React from 'react'
import { Link } from 'react-router-dom'

interface EmptyStateProps {
    icon?: string
    title: string
    description: string
    actionLabel?: string
    actionHref?: string
    actionOnClick?: () => void
    actionText?: string
    actionLink?: string
    action?: React.ReactNode
}

export const EmptyState: React.FC<EmptyStateProps> = ({
    icon,
    title,
    description,
    actionLabel,
    actionHref,
    actionOnClick,
    actionText,
    actionLink,
    action,
}) => {
    // Support both actionLabel/actionHref and actionText/actionLink
    const finalLabel = actionText || actionLabel
    const finalHref = actionLink || actionHref

    return (
        <div className="flex flex-col items-center justify-center py-12 px-4 text-center">
            {icon && <div className="text-6xl mb-4">{icon}</div>}
            <h3 className="text-lg font-semibold text-slate-900 dark:text-white mb-2">
                {title}
            </h3>
            <p className="text-slate-600 dark:text-slate-400 max-w-sm mb-6">
                {description}
            </p>
            {action}
            {finalLabel && (
                <>
                    {finalHref ? (
                        <Link
                            to={finalHref}
                            className="inline-flex items-center px-4 py-2 border border-transparent text-sm font-medium rounded-md shadow-sm text-white bg-blue-600 hover:bg-blue-700"
                        >
                            {finalLabel}
                        </Link>
                    ) : (
                        <button
                            onClick={actionOnClick}
                            className="inline-flex items-center px-4 py-2 border border-transparent text-sm font-medium rounded-md shadow-sm text-white bg-blue-600 hover:bg-blue-700"
                        >
                            {finalLabel}
                        </button>
                    )}
                </>
            )}
        </div>
    )
}

export default EmptyState

