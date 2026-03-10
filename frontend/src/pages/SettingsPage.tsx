import React from 'react'

const SettingsPage: React.FC = () => {
    return (
        <div className="px-4 py-6 sm:px-6 lg:px-8">
            <div className="sm:flex sm:items-center">
                <div className="sm:flex-auto">
                    <h1 className="text-2xl font-semibold text-slate-900 dark:text-white">Settings</h1>
                    <p className="mt-2 text-sm text-slate-700 dark:text-slate-400">
                        Manage your account settings and preferences.
                    </p>
                </div>
            </div>

            <div className="mt-8 max-w-2xl">
                <div className="card">
                    <div className="card-header">
                        <h3 className="text-lg font-medium">Account Settings</h3>
                    </div>
                    <div className="card-body">
                        <p className="text-muted-foreground">Settings page coming soon...</p>
                    </div>
                </div>
            </div>
        </div>
    )
}

export default SettingsPage