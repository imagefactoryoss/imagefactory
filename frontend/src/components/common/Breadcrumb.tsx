/**
 * Breadcrumb navigation component for showing page hierarchy
 */

import React from 'react'
import { Link } from 'react-router-dom'

interface BreadcrumbItem {
    label?: string
    name?: string
    href?: string
    current?: boolean
}

interface BreadcrumbProps {
    items: BreadcrumbItem[]
}

export const Breadcrumb: React.FC<BreadcrumbProps> = ({ items }) => {
    return (
        <nav className="flex items-center space-x-2 text-sm mb-6" aria-label="Breadcrumb">
            <ol className="flex items-center space-x-2">
                {items.map((item, index) => {
                    const label = item.label || item.name || ''
                    const isCurrent = item.current || index === items.length - 1

                    return (
                        <li key={index} className="flex items-center space-x-2">
                            {index > 0 && (
                                <svg
                                    className="w-4 h-4 text-slate-400 dark:text-slate-600"
                                    fill="none"
                                    stroke="currentColor"
                                    viewBox="0 0 24 24"
                                >
                                    <path
                                        strokeLinecap="round"
                                        strokeLinejoin="round"
                                        strokeWidth={2}
                                        d="M9 5l7 7-7 7"
                                    />
                                </svg>
                            )}
                            {item.href && !isCurrent ? (
                                <Link
                                    to={item.href}
                                    className="text-blue-600 dark:text-blue-400 hover:text-blue-800 dark:hover:text-blue-300 hover:underline"
                                >
                                    {label}
                                </Link>
                            ) : (
                                <span
                                    className={
                                        isCurrent
                                            ? 'text-slate-900 dark:text-white font-medium'
                                            : 'text-slate-600 dark:text-slate-400'
                                    }
                                >
                                    {label}
                                </span>
                            )}
                        </li>
                    )
                })}
            </ol>
        </nav>
    )
}

export default Breadcrumb

