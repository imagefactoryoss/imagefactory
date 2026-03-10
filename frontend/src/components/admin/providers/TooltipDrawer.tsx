import { Check, Copy, X } from 'lucide-react'
import React, { useState } from 'react'

interface TooltipDrawerProps {
    isOpen: boolean
    onClose: () => void
    title: string
    children: React.ReactNode
}

export const TooltipDrawer: React.FC<TooltipDrawerProps> = ({
    isOpen,
    onClose,
    title,
    children
}) => {
    if (!isOpen) return null

    return (
        <div className="fixed inset-0 z-50 flex items-center justify-center">
            {/* Backdrop */}
            <div
                className="absolute inset-0 bg-black bg-opacity-50"
                onClick={onClose}
            />

            {/* Drawer */}
            <div className="relative bg-white dark:bg-gray-800 rounded-lg shadow-xl max-w-4xl w-full mx-4 max-h-[90vh] overflow-hidden">
                {/* Header */}
                <div className="flex items-center justify-between p-6 border-b border-gray-200 dark:border-gray-700">
                    <h2 className="text-xl font-semibold text-gray-900 dark:text-white">
                        {title}
                    </h2>
                    <button
                        type="button"
                        onClick={onClose}
                        className="text-gray-400 hover:text-gray-600 dark:hover:text-gray-300"
                    >
                        <X className="h-6 w-6" />
                    </button>
                </div>

                {/* Content */}
                <div className="p-6 overflow-y-auto max-h-[calc(90vh-120px)]">
                    {children}
                </div>
            </div>
        </div>
    )
}

interface CopyableCodeBlockProps {
    title: string
    code: string
    language?: string
}

export const CopyableCodeBlock: React.FC<CopyableCodeBlockProps> = ({
    title,
    code,
    language = 'json'
}) => {
    const [copied, setCopied] = useState(false)

    const copyToClipboard = async () => {
        try {
            await navigator.clipboard.writeText(code)
            setCopied(true)
            setTimeout(() => setCopied(false), 2000)
        } catch (err) {
            console.error('Failed to copy text: ', err)
        }
    }

    return (
        <div className="mb-6">
            <div className="flex items-center justify-between mb-2">
                <h3 className="text-sm font-medium text-gray-900 dark:text-white">
                    {title}
                </h3>
                <button
                    type="button"
                    onClick={copyToClipboard}
                    className="flex items-center gap-2 px-3 py-1 text-xs bg-gray-100 dark:bg-gray-700 hover:bg-gray-200 dark:hover:bg-gray-600 rounded-md transition-colors"
                    title="Copy to clipboard"
                >
                    {copied ? (
                        <>
                            <Check className="h-3 w-3 text-green-600" />
                            <span className="text-green-600">Copied!</span>
                        </>
                    ) : (
                        <>
                            <Copy className="h-3 w-3" />
                            <span>Copy</span>
                        </>
                    )}
                </button>
            </div>
            <pre className="bg-gray-900 text-gray-100 p-4 rounded-md overflow-x-auto text-sm">
                <code className={`language-${language}`}>{code}</code>
            </pre>
        </div>
    )
}
