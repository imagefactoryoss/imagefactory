import React, { useEffect } from 'react'

interface DrawerProps {
    isOpen: boolean
    title: string
    description?: string
    onClose: () => void
    children: React.ReactNode
    width?: 'sm' | 'md' | 'lg' | 'xl' | '2xl' | '3xl' | '60vw'
    size?: 'sm' | 'md' | 'lg' | 'xl' | '2xl' | '3xl' | '60vw'
}

const widthClasses: Record<NonNullable<DrawerProps['width']>, string> = {
    sm: 'sm:max-w-sm',
    md: 'sm:max-w-md',
    lg: 'sm:max-w-lg',
    xl: 'sm:max-w-xl',
    '2xl': 'sm:max-w-2xl',
    '3xl': 'sm:max-w-3xl',
    '60vw': 'sm:max-w-[60vw]',
}

const Drawer: React.FC<DrawerProps> = ({
    isOpen,
    title,
    description,
    onClose,
    children,
    width = 'md',
    size,
}) => {
    const resolvedWidth = size || width

    // Close drawer on Escape key
    useEffect(() => {
        const handleEscape = (e: KeyboardEvent) => {
            if (e.key === 'Escape' && isOpen) {
                onClose()
            }
        }

        if (isOpen) {
            document.addEventListener('keydown', handleEscape)
            return () => document.removeEventListener('keydown', handleEscape)
        }
    }, [isOpen, onClose])

    if (!isOpen) return null

    return (
        <div className="fixed inset-0 z-50 overflow-hidden">
            {/* Overlay */}
            <div
                className="absolute inset-0 bg-black/50 dark:bg-black/70 transition-opacity"
                onClick={onClose}
            />

            {/* Drawer */}
            <div className="pointer-events-none fixed inset-y-0 right-0 flex max-w-full pl-10">
                <div className={`pointer-events-auto w-screen ${widthClasses[resolvedWidth]}`} onClick={(e) => e.stopPropagation()}>
                    <div className="flex h-full flex-col overflow-y-scroll bg-white dark:bg-slate-900 shadow-xl">
                        {/* Header */}
                        <div className="border-b border-slate-200 dark:border-slate-700 px-4 py-6 sm:px-6">
                            <div className="flex items-center justify-between">
                                <div>
                                    <h2 className="text-lg font-semibold text-slate-900 dark:text-white">
                                        {title}
                                    </h2>
                                    {description && (
                                        <p className="mt-1 text-sm text-slate-600 dark:text-slate-400">
                                            {description}
                                        </p>
                                    )}
                                </div>
                                <button
                                    onClick={onClose}
                                    className="rounded-md bg-white dark:bg-slate-900 text-slate-500 dark:text-slate-400 hover:text-slate-700 dark:hover:text-slate-300 focus:outline-none"
                                >
                                    <svg
                                        className="h-6 w-6"
                                        fill="none"
                                        stroke="currentColor"
                                        viewBox="0 0 24 24"
                                    >
                                        <path
                                            strokeLinecap="round"
                                            strokeLinejoin="round"
                                            strokeWidth={2}
                                            d="M6 18L18 6M6 6l12 12"
                                        />
                                    </svg>
                                </button>
                            </div>
                        </div>

                        {/* Content */}
                        <div className="flex-1 overflow-y-auto px-4 py-6 sm:px-6">
                            {children}
                        </div>
                    </div>
                </div>
            </div>
        </div>
    )
}

export default Drawer
