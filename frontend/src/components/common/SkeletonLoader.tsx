/**
 * Skeleton loader components for loading states
 */

import React from 'react'

export const SkeletonCard: React.FC = () => {
    return (
        <div className="bg-white dark:bg-slate-800 rounded-lg border border-slate-200 dark:border-slate-700 p-6 animate-pulse">
            <div className="space-y-4">
                <div className="h-4 bg-slate-200 dark:bg-slate-700 rounded w-3/4"></div>
                <div className="h-4 bg-slate-200 dark:bg-slate-700 rounded w-1/2"></div>
                <div className="pt-4 space-y-2">
                    <div className="h-3 bg-slate-200 dark:bg-slate-700 rounded"></div>
                    <div className="h-3 bg-slate-200 dark:bg-slate-700 rounded w-5/6"></div>
                </div>
            </div>
        </div>
    )
}

export const SkeletonTable: React.FC<{ rows?: number }> = ({ rows = 5 }) => {
    return (
        <div className="space-y-2">
            {Array.from({ length: rows }).map((_, i) => (
                <div key={i} className="h-12 bg-slate-200 dark:bg-slate-700 rounded animate-pulse"></div>
            ))}
        </div>
    )
}

export const SkeletonGrid: React.FC<{ columns?: number; items?: number }> = ({ columns = 3, items = 6 }) => {
    return (
        <div className={`grid grid-cols-1 md:grid-cols-2 lg:grid-cols-${columns} gap-6`}>
            {Array.from({ length: items }).map((_, i) => (
                <SkeletonCard key={i} />
            ))}
        </div>
    )
}

export const SkeletonText: React.FC<{ lines?: number }> = ({ lines = 3 }) => {
    return (
        <div className="space-y-2">
            {Array.from({ length: lines }).map((_, i) => (
                <div
                    key={i}
                    className="h-4 bg-slate-200 dark:bg-slate-700 rounded animate-pulse"
                    style={{ width: `${80 + Math.random() * 20}%` }}
                ></div>
            ))}
        </div>
    )
}
