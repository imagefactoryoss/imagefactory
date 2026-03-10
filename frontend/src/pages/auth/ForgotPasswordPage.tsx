import { authService } from '@/services/authService'
import { useThemeStore } from '@/store/theme'
import { ArrowLeft, Building2, CheckCircle2, KeyRound } from 'lucide-react'
import { Moon, Sun } from 'lucide-react'
import React, { useState } from 'react'
import toast from 'react-hot-toast'
import { Link } from 'react-router-dom'

const ForgotPasswordPage: React.FC = () => {
    const { isDark, toggleTheme } = useThemeStore()
    const [email, setEmail] = useState('')
    const [isLoading, setIsLoading] = useState(false)
    const [isSubmitted, setIsSubmitted] = useState(false)
    const [errorMessage, setErrorMessage] = useState<string>('')
    const [isDirectoryManaged, setIsDirectoryManaged] = useState(false)

    const handleSubmit = async (e: React.FormEvent) => {
        e.preventDefault()
        if (isLoading) return

        setIsLoading(true)
        setErrorMessage('')
        setIsDirectoryManaged(false)

        try {
            await authService.requestPasswordReset(email)
            setIsSubmitted(true)
            toast.success("If we find your account in the system, you'll receive a password reset link")
        } catch (error: any) {
            const errorMsg = error.message || 'Failed to send password reset email'
            const normalizedError = errorMsg.toLowerCase()
            if (normalizedError.includes('not available for this account type') || normalizedError.includes('corporate directory') || normalizedError.includes('ldap')) {
                setIsDirectoryManaged(true)
                setErrorMessage('This account is managed by your corporate directory (LDAP). Change your password in your corporate identity system, then sign in again.')
                return
            }
            setErrorMessage(errorMsg)
            toast.error(errorMsg)
        } finally {
            setIsLoading(false)
        }
    }

    if (isSubmitted) {
        return (
            <div className="min-h-screen bg-gradient-to-br from-slate-100 via-white to-cyan-50 px-4 py-6 dark:from-slate-950 dark:via-slate-900 dark:to-slate-900 sm:px-6 lg:px-8">
                <div className="mx-auto flex min-h-[calc(100vh-3rem)] max-w-md items-center">
                    <div className="w-full rounded-2xl border border-slate-200 bg-white/95 p-8 shadow-xl dark:border-slate-700 dark:bg-slate-900">
                        <div className="mb-2 flex justify-end">
                            <button
                                type="button"
                                onClick={toggleTheme}
                                className="inline-flex items-center gap-1 rounded-md border border-slate-300 bg-white px-2.5 py-1 text-xs font-medium text-slate-700 transition-colors hover:bg-slate-50 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500/40 dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200 dark:hover:bg-slate-700 dark:focus-visible:ring-blue-300/40"
                                title="Toggle dark mode"
                            >
                                {isDark ? <Sun className="h-3.5 w-3.5" /> : <Moon className="h-3.5 w-3.5" />}
                                {isDark ? 'Light' : 'Dark'}
                            </button>
                        </div>
                        <div className="mx-auto flex h-12 w-12 items-center justify-center rounded-full bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-300">
                            <CheckCircle2 className="h-6 w-6" />
                        </div>
                        <h2 className="mt-5 text-center text-2xl font-bold text-slate-900 dark:text-slate-100">Check your email</h2>
                        <p className="mt-2 text-center text-sm text-slate-600 dark:text-slate-300">
                            If we find your account in the system, you&apos;ll receive a password reset link.
                        </p>

                        <div className="mt-6 rounded-lg border border-slate-200 bg-slate-50/80 p-3 text-center text-xs text-slate-600 dark:border-slate-700 dark:bg-slate-800/70 dark:text-slate-300">
                            Didn&apos;t receive the email?{' '}
                            <button
                                onClick={() => setIsSubmitted(false)}
                                className="font-medium text-blue-600 hover:text-blue-500 dark:text-blue-300 dark:hover:text-blue-200"
                            >
                                Try again
                            </button>
                        </div>

                        <div className="mt-6 text-center">
                            <Link to="/login" className="inline-flex items-center gap-1.5 text-sm font-medium text-blue-600 hover:text-blue-500 dark:text-blue-300 dark:hover:text-blue-200">
                                <ArrowLeft className="h-4 w-4" />
                                Back to sign in
                            </Link>
                        </div>
                    </div>
                </div>
            </div>
        )
    }

    return (
        <div className="min-h-screen bg-gradient-to-br from-slate-100 via-white to-cyan-50 px-4 py-6 dark:from-slate-950 dark:via-slate-900 dark:to-slate-900 sm:px-6 lg:px-8">
            <div className="mx-auto flex min-h-[calc(100vh-3rem)] max-w-md items-center">
                <div className="w-full rounded-2xl border border-slate-200 bg-white/95 p-8 shadow-xl dark:border-slate-700 dark:bg-slate-900">
                    <div className="mb-2 flex justify-end">
                        <button
                            type="button"
                            onClick={toggleTheme}
                            className="inline-flex items-center gap-1 rounded-md border border-slate-300 bg-white px-2.5 py-1 text-xs font-medium text-slate-700 transition-colors hover:bg-slate-50 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500/40 dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200 dark:hover:bg-slate-700 dark:focus-visible:ring-blue-300/40"
                            title="Toggle dark mode"
                        >
                            {isDark ? <Sun className="h-3.5 w-3.5" /> : <Moon className="h-3.5 w-3.5" />}
                            {isDark ? 'Light' : 'Dark'}
                        </button>
                    </div>
                    <div className="mx-auto flex h-12 w-12 items-center justify-center rounded-full bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-300">
                        <KeyRound className="h-6 w-6" />
                    </div>
                    <h2 className="mt-5 text-center text-2xl font-bold text-slate-900 dark:text-slate-100">Reset your password</h2>
                    <p className="mt-2 text-center text-sm text-slate-600 dark:text-slate-300">
                        Enter your email address and we&apos;ll send you a reset link for local accounts.
                    </p>

                    <div className="mt-5 rounded-lg border border-cyan-200 bg-cyan-50/70 p-3 text-xs text-cyan-900 dark:border-cyan-800 dark:bg-cyan-950/20 dark:text-cyan-200">
                        LDAP / directory-managed users must reset passwords in their corporate identity system.
                    </div>

                    <form className="mt-6 space-y-4" onSubmit={handleSubmit}>
                        <div>
                            <label htmlFor="email" className="mb-1 block text-sm font-medium text-slate-700 dark:text-slate-300">Email</label>
                            <input
                                id="email"
                                name="email"
                                type="email"
                                autoComplete="email"
                                required
                                value={email}
                                onChange={(e) => setEmail(e.target.value)}
                                placeholder="you@company.com"
                                className="auth-input w-full rounded-md border border-slate-300 bg-white px-3 py-2 text-sm text-slate-900 placeholder:text-slate-400 focus:border-blue-500 focus:outline-none focus:ring-2 focus:ring-blue-500/30 dark:border-slate-600 dark:bg-slate-800 dark:text-white dark:placeholder:text-slate-500"
                                disabled={isLoading}
                            />
                        </div>

                        {errorMessage && (
                            <div className={`rounded-md border p-3 text-sm ${isDirectoryManaged ? 'border-amber-300 bg-amber-50 text-amber-900 dark:border-amber-700 dark:bg-amber-900/30 dark:text-amber-200' : 'border-red-200 bg-red-50 text-red-700 dark:border-red-800 dark:bg-red-950/40 dark:text-red-200'}`}>
                                <div className="flex items-start gap-2">
                                    {isDirectoryManaged && <Building2 className="mt-0.5 h-4 w-4 shrink-0" />}
                                    <span>{errorMessage}</span>
                                </div>
                            </div>
                        )}

                        <button
                            type="submit"
                            disabled={isLoading}
                            className="inline-flex w-full items-center justify-center rounded-md bg-blue-600 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-blue-700 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500/40 disabled:cursor-not-allowed disabled:opacity-60 dark:bg-blue-500 dark:hover:bg-blue-400 dark:focus-visible:ring-blue-300/40"
                        >
                            {isLoading ? 'Sending...' : 'Send reset instructions'}
                        </button>

                        <div className="pt-1 text-center">
                            <Link to="/login" className="inline-flex items-center gap-1.5 text-sm font-medium text-blue-600 hover:text-blue-500 dark:text-blue-300 dark:hover:text-blue-200">
                                <ArrowLeft className="h-4 w-4" />
                                Back to sign in
                            </Link>
                        </div>
                    </form>
                </div>
            </div>
        </div>
    )
}

export default ForgotPasswordPage
