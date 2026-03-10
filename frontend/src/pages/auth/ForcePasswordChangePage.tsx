import { authService } from '@/services/authService'
import { useAuthStore } from '@/store/auth'
import React, { useState } from 'react'
import toast from 'react-hot-toast'
import { useNavigate } from 'react-router-dom'

const ForcePasswordChangePage: React.FC = () => {
    const navigate = useNavigate()
    const { setupRequired, setRequiresPasswordChange } = useAuthStore()

    const [currentPassword, setCurrentPassword] = useState('')
    const [newPassword, setNewPassword] = useState('')
    const [confirmPassword, setConfirmPassword] = useState('')
    const [loading, setLoading] = useState(false)
    const [showCurrentPassword, setShowCurrentPassword] = useState(false)
    const [showNewPassword, setShowNewPassword] = useState(false)
    const [showConfirmPassword, setShowConfirmPassword] = useState(false)

    const onSubmit = async (e: React.FormEvent) => {
        e.preventDefault()
        if (loading) return

        if (!currentPassword || !newPassword || !confirmPassword) {
            toast.error('All fields are required')
            return
        }
        if (newPassword !== confirmPassword) {
            toast.error('New password and confirmation do not match')
            return
        }

        try {
            setLoading(true)
            await authService.changePassword(currentPassword, newPassword)
            setRequiresPasswordChange(false)
            toast.success('Password changed successfully')
            navigate(setupRequired ? '/admin/setup' : '/')
        } catch (error: any) {
            toast.error(error?.message || 'Failed to change password')
        } finally {
            setLoading(false)
        }
    }

    return (
        <div className="min-h-screen flex items-center justify-center bg-slate-50 dark:bg-slate-900 py-12 px-4">
            <div className="max-w-md w-full bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 rounded-lg p-6 space-y-4 shadow-sm">
                <h1 className="text-xl font-semibold text-slate-900 dark:text-white">Password Change Required</h1>
                <p className="text-sm text-slate-600 dark:text-slate-300">
                    You must change your password before continuing.
                </p>
                <form className="space-y-3" onSubmit={onSubmit}>
                    <div className="relative">
                        <input
                            type={showCurrentPassword ? 'text' : 'password'}
                            className="w-full border border-slate-300 dark:border-slate-600 rounded px-3 py-2 pr-10 bg-white dark:bg-slate-700 text-slate-900 dark:text-white placeholder-slate-500 dark:placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
                            placeholder="Current password"
                            value={currentPassword}
                            onChange={e => setCurrentPassword(e.target.value)}
                        />
                        <button
                            type="button"
                            onClick={() => setShowCurrentPassword(prev => !prev)}
                            className="absolute inset-y-0 right-0 px-3 text-xs font-medium text-slate-500 dark:text-slate-300"
                            aria-label={showCurrentPassword ? 'Hide current password' : 'Show current password'}
                        >
                            {showCurrentPassword ? 'Hide' : 'Show'}
                        </button>
                    </div>
                    <div className="relative">
                        <input
                            type={showNewPassword ? 'text' : 'password'}
                            className="w-full border border-slate-300 dark:border-slate-600 rounded px-3 py-2 pr-10 bg-white dark:bg-slate-700 text-slate-900 dark:text-white placeholder-slate-500 dark:placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
                            placeholder="New password"
                            value={newPassword}
                            onChange={e => setNewPassword(e.target.value)}
                        />
                        <button
                            type="button"
                            onClick={() => setShowNewPassword(prev => !prev)}
                            className="absolute inset-y-0 right-0 px-3 text-xs font-medium text-slate-500 dark:text-slate-300"
                            aria-label={showNewPassword ? 'Hide new password' : 'Show new password'}
                        >
                            {showNewPassword ? 'Hide' : 'Show'}
                        </button>
                    </div>
                    <div className="relative">
                        <input
                            type={showConfirmPassword ? 'text' : 'password'}
                            className="w-full border border-slate-300 dark:border-slate-600 rounded px-3 py-2 pr-10 bg-white dark:bg-slate-700 text-slate-900 dark:text-white placeholder-slate-500 dark:placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
                            placeholder="Confirm new password"
                            value={confirmPassword}
                            onChange={e => setConfirmPassword(e.target.value)}
                        />
                        <button
                            type="button"
                            onClick={() => setShowConfirmPassword(prev => !prev)}
                            className="absolute inset-y-0 right-0 px-3 text-xs font-medium text-slate-500 dark:text-slate-300"
                            aria-label={showConfirmPassword ? 'Hide confirm password' : 'Show confirm password'}
                        >
                            {showConfirmPassword ? 'Hide' : 'Show'}
                        </button>
                    </div>
                    <button
                        type="submit"
                        className="w-full rounded bg-blue-600 hover:bg-blue-700 text-white px-4 py-2 disabled:opacity-60 disabled:cursor-not-allowed transition-colors"
                        disabled={loading}
                    >
                        {loading ? 'Updating...' : 'Update Password'}
                    </button>
                </form>
            </div>
        </div>
    )
}

export default ForcePasswordChangePage
