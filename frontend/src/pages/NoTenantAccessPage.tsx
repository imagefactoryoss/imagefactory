import { authService } from '@/services/authService'
import { useAuthStore } from '@/store/auth'
import { AlertCircle, Clock, LogOut } from 'lucide-react'
import { useNavigate } from 'react-router-dom'

export default function NoTenantAccessPage() {
    const { user, logout } = useAuthStore()
    const navigate = useNavigate()

    const handleLogout = async () => {
        try {
            await authService.logout()
        } catch (error) {
        }
        logout()
        navigate('/login')
    }

    return (
        <div className="min-h-screen bg-gradient-to-br from-slate-50 to-slate-100 dark:from-slate-950 dark:to-slate-900 flex items-center justify-center p-4">
            <div className="max-w-md w-full bg-white dark:bg-slate-900 rounded-lg shadow-lg border border-slate-200 dark:border-slate-700">
                {/* Header with icon */}
                <div className="bg-gradient-to-r from-amber-50 to-orange-50 dark:from-amber-950 dark:to-orange-950 p-6 border-b border-slate-200 dark:border-slate-700">
                    <div className="flex items-center justify-center mb-4">
                        <div className="p-3 bg-amber-100 dark:bg-amber-900 rounded-full">
                            <AlertCircle className="w-6 h-6 text-amber-600 dark:text-amber-400" />
                        </div>
                    </div>
                    <h1 className="text-xl font-bold text-center text-slate-900 dark:text-white">
                        Account Pending Access
                    </h1>
                </div>

                {/* Content */}
                <div className="p-6">
                    {/* User info section */}
                    <div className="mb-6 p-4 bg-slate-50 dark:bg-slate-800 rounded-lg">
                        <p className="text-sm text-slate-600 dark:text-slate-400 mb-2">
                            Logged in as:
                        </p>
                        <p className="font-semibold text-slate-900 dark:text-white">
                            {user?.first_name || ''} {user?.last_name || ''}
                        </p>
                        <p className="text-sm text-slate-600 dark:text-slate-400 mt-1">
                            {user?.email || ''}
                        </p>
                    </div>

                    {/* Message section */}
                    <div className="mb-6">
                        <div className="flex items-start gap-3 p-4 bg-blue-50 dark:bg-blue-950 rounded-lg border border-blue-200 dark:border-blue-800">
                            <Clock className="w-5 h-5 text-blue-600 dark:text-blue-400 flex-shrink-0 mt-0.5" />
                            <div>
                                <h2 className="font-semibold text-blue-900 dark:text-blue-100 mb-2">
                                    Waiting for Access
                                </h2>
                                <p className="text-sm text-blue-800 dark:text-blue-200 leading-relaxed">
                                    Your account has been created, but you need to be assigned to at least one
                                    tenant group to access Image Factory content.
                                </p>
                            </div>
                        </div>
                    </div>

                    {/* What to do section */}
                    <div className="mb-6 p-4 bg-slate-50 dark:bg-slate-800 rounded-lg">
                        <h3 className="font-semibold text-slate-900 dark:text-white mb-3">
                            What happens next?
                        </h3>
                        <ul className="space-y-2 text-sm text-slate-700 dark:text-slate-300">
                            <li className="flex gap-2">
                                <span className="font-semibold text-slate-900 dark:text-white">1.</span>
                                <span>A tenant administrator will add you to a tenant group</span>
                            </li>
                            <li className="flex gap-2">
                                <span className="font-semibold text-slate-900 dark:text-white">2.</span>
                                <span>Your access will be activated automatically</span>
                            </li>
                            <li className="flex gap-2">
                                <span className="font-semibold text-slate-900 dark:text-white">3.</span>
                                <span>You'll be able to see content and work with builds</span>
                            </li>
                        </ul>
                    </div>

                    {/* Contact information */}
                    <div className="mb-6 p-4 bg-slate-50 dark:bg-slate-800 rounded-lg border border-slate-200 dark:border-slate-700">
                        <p className="text-sm text-slate-600 dark:text-slate-400 mb-2">
                            <span className="font-semibold text-slate-900 dark:text-white">Need help?</span>
                        </p>
                        <p className="text-sm text-slate-600 dark:text-slate-400">
                            Contact your system administrator if you believe this is in error.
                        </p>
                    </div>

                    {/* Profile details */}
                    <div className="mb-6 p-4 bg-slate-50 dark:bg-slate-800 rounded-lg border border-slate-200 dark:border-slate-700">
                        <h3 className="font-semibold text-slate-900 dark:text-white mb-3 text-sm">
                            Your Profile Information
                        </h3>
                        <div className="space-y-2 text-sm">
                            <div className="flex justify-between">
                                <span className="text-slate-600 dark:text-slate-400">Name:</span>
                                <span className="text-slate-900 dark:text-white font-medium">
                                    {user?.first_name} {user?.last_name}
                                </span>
                            </div>
                            <div className="flex justify-between">
                                <span className="text-slate-600 dark:text-slate-400">Email:</span>
                                <span className="text-slate-900 dark:text-white font-medium truncate">
                                    {user?.email}
                                </span>
                            </div>
                            {/* Phone field not available in UserResponse type */}
                            <div className="flex justify-between">
                                <span className="text-slate-600 dark:text-slate-400">Status:</span>
                                <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-green-100 dark:bg-green-900 text-green-800 dark:text-green-200">
                                    Active
                                </span>
                            </div>
                        </div>
                    </div>

                    {/* Action button */}
                    <button
                        onClick={handleLogout}
                        className="w-full px-4 py-2 bg-slate-900 dark:bg-slate-100 text-white dark:text-slate-900 rounded-lg font-medium hover:bg-slate-800 dark:hover:bg-slate-200 transition-colors flex items-center justify-center gap-2"
                    >
                        <LogOut className="w-4 h-4" />
                        Logout
                    </button>
                </div>

                {/* Footer help text */}
                <div className="px-6 py-4 bg-slate-50 dark:bg-slate-800 border-t border-slate-200 dark:border-slate-700 rounded-b-lg text-center text-xs text-slate-600 dark:text-slate-400">
                    You can login again once you've been assigned to a tenant group.
                </div>
            </div>
        </div>
    )
}
