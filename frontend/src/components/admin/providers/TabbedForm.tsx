import React, { useState } from 'react'

interface Tab {
    id: string
    label: string
    content: React.ReactNode
    disabled?: boolean
}

interface TabbedFormProps {
    tabs: Tab[]
    defaultActiveTab?: string
    className?: string
}

export const TabbedForm: React.FC<TabbedFormProps> = ({
    tabs,
    defaultActiveTab,
    className = ''
}) => {
    const [activeTab, setActiveTab] = useState(defaultActiveTab || tabs[0]?.id || '')

    const activeTabContent = tabs.find(tab => tab.id === activeTab)?.content

    return (
        <div className={`space-y-4 ${className}`}>
            {/* Tab Navigation */}
            <div className="border-b border-gray-200 dark:border-gray-700">
                <nav className="-mb-px flex space-x-8" aria-label="Tabs">
                    {tabs.map((tab) => (
                        <button
                            key={tab.id}
                            type="button"
                            onClick={() => !tab.disabled && setActiveTab(tab.id)}
                            disabled={tab.disabled}
                            className={`whitespace-nowrap py-2 px-1 border-b-2 font-medium text-sm ${activeTab === tab.id
                                ? 'border-blue-500 text-blue-600 dark:text-blue-400'
                                : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300 dark:text-gray-400 dark:hover:text-gray-300'
                                } ${tab.disabled ? 'opacity-50 cursor-not-allowed' : 'cursor-pointer'}`}
                            aria-current={activeTab === tab.id ? 'page' : undefined}
                        >
                            {tab.label}
                        </button>
                    ))}
                </nav>
            </div>

            {/* Tab Content */}
            <div>
                {activeTabContent}
            </div>
        </div>
    )
}
