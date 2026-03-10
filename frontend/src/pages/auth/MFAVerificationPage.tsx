import { useDashboardPath } from '@/hooks/useAccess'
import { mfaService } from '@/services/authService'
import React, { useState } from 'react'
import toast from 'react-hot-toast'
import { useLocation, useNavigate } from 'react-router-dom'

const MFAVerificationPage: React.FC = () => {
    const navigate = useNavigate()
    const location = useLocation()
    const dashboardPath = useDashboardPath()
    // const { login } = useAuthStore()

    const [verificationCode, setVerificationCode] = useState('')
    const [loading, setLoading] = useState(false)

    const state = location.state as {
        user: any
        token: string
        refreshToken: string
        challenge: any
    }

    // Redirect if no MFA challenge data
    if (!state?.user || !state?.token || !state?.challenge) {
        navigate('/login')
        return null
    }

    const handleSubmit = async (e: React.FormEvent) => {
        e.preventDefault()
        if (loading || !verificationCode.trim()) return

        setLoading(true)

        try {
            const response = await mfaService.verifyMFA({
                challengeId: state.challenge.id,
                verificationCode: verificationCode.trim(),
                method: state.challenge.method,
            })

            if (response.success) {
                // MFA verification successful - complete login
                navigate(dashboardPath, { replace: true })
                toast.success('Login successful')
            } else {
                toast.error('Invalid verification code. Please try again.')
            }
        } catch (error: any) {
            toast.error(error.message || 'MFA verification failed')
        } finally {
            setLoading(false)
        }
    }

    const handleResendCode = async () => {
        try {
            // In a real implementation, call MFA service to resend code
            toast.success('New verification code sent')
        } catch (error: any) {
            toast.error(error.message || 'Failed to resend verification code')
        }
    }

    // const handleCodeChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    //     const value = e.target.value.replace(/\D/g, '') // Only digits
    //     if (value.length <= 6) {
    //         setVerificationCode(value)
    //     }
    // }

    const handleInputChange = (e: React.ChangeEvent<HTMLInputElement>) => {
        const value = e.target.value
        // Allow digits and common MFA code separators
        const cleanValue = value.replace(/[^\d\s-]/g, '').slice(0, 6)
        setVerificationCode(cleanValue)
    }

    const formatCode = (code: string) => {
        const digits = code.replace(/\D/g, '')
        return digits.replace(/(\d{2})(?=\d)/g, '$1 ').trim()
    }

    const getMFAMethodIcon = (method: string) => {
        switch (method) {
            case 'totp':
                return (
                    <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z" />
                    </svg>
                )
            case 'sms':
                return (
                    <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M7 4V2a1 1 0 011-1h8a1 1 0 011 1v2h4a1 1 0 011 1v1a1 1 0 01-1 1h-1v12a2 2 0 01-2 2H6a2 2 0 01-2-2V7H3a1 1 0 01-1-1V5a1 1 0 011-1h4zM9 9h6M9 13h6M9 17h3" />
                    </svg>
                )
            case 'email':
                return (
                    <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M3 8l7.89 4.26a2 2 0 002.22 0L21 8M5 19h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z" />
                    </svg>
                )
            default:
                return (
                    <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
                    </svg>
                )
        }
    }

    const getMFAMethodDescription = (method: string) => {
        switch (method) {
            case 'totp':
                return 'Enter the 6-digit code from your authenticator app'
            case 'sms':
                return 'Enter the 6-digit code sent to your phone'
            case 'email':
                return 'Enter the 6-digit code sent to your email'
            default:
                return 'Enter your verification code'
        }
    }

    return (
        <div className="min-h-screen flex items-center justify-center bg-slate-50 dark:bg-slate-900 py-12 px-4 sm:px-6 lg:px-8">
            <div className="max-w-md w-full space-y-8">
                <div>
                    <div className="mx-auto h-16 w-16 bg-blue-600 dark:bg-blue-600 rounded-full flex items-center justify-center text-white dark:text-white">
                        {getMFAMethodIcon(state.challenge.method)}
                    </div>
                    <h2 className="mt-6 text-center text-3xl font-extrabold text-slate-900 dark:text-white">
                        Two-Factor Authentication
                    </h2>
                    <p className="mt-2 text-center text-sm text-slate-600 dark:text-slate-400">
                        {getMFAMethodDescription(state.challenge.method)}
                    </p>
                </div>

                <div className="bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-800 rounded-md p-4">
                    <div className="flex">
                        <div className="flex-shrink-0">
                            <svg className="h-5 w-5 text-blue-400 dark:text-blue-400" fill="currentColor" viewBox="0 0 20 20">
                                <path fillRule="evenodd" d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7-4a1 1 0 11-2 0 1 1 0 012 0zM9 9a1 1 0 000 2v3a1 1 0 001 1h1a1 1 0 100-2v-3a1 1 0 00-1-1H9z" clipRule="evenodd" />
                            </svg>
                        </div>
                        <div className="ml-3">
                            <h3 className="text-sm font-medium text-blue-800 dark:text-blue-200">
                                Verification Required
                            </h3>
                            <p className="mt-1 text-sm text-blue-700 dark:text-blue-100">
                                We need to verify your identity before proceeding to your account.
                            </p>
                        </div>
                    </div>
                </div>

                <form className="mt-8 space-y-6" onSubmit={handleSubmit}>
                    <div>
                        <label htmlFor="verification-code" className="block text-sm font-medium text-slate-900 dark:text-slate-300">
                            Verification Code
                        </label>
                        <div className="mt-1">
                            <input
                                id="verification-code"
                                name="verification-code"
                                type="text"
                                inputMode="numeric"
                                pattern="[0-9]*"
                                autoComplete="off"
                                required
                                maxLength={8} // Allow for spaces/hyphens
                                className="appearance-none relative block w-full px-3 py-2 border border-slate-300 dark:border-slate-600 placeholder-slate-500 dark:placeholder-slate-400 text-slate-900 dark:text-white dark:bg-slate-800 rounded-md focus:outline-none focus:ring-blue-500 focus:border-blue-500 focus:z-10 sm:text-sm text-center text-2xl tracking-widest font-mono"
                                placeholder="00 00 00"
                                value={formatCode(verificationCode)}
                                onChange={handleInputChange}
                                autoFocus
                            />
                        </div>
                        <p className="mt-2 text-sm text-slate-600 dark:text-slate-400">
                            Enter the 6-digit code from your {state.challenge.method === 'totp' ? 'authenticator app' : state.challenge.method === 'sms' ? 'SMS' : 'email'}
                        </p>
                    </div>

                    <div className="flex items-center justify-between">
                        <div className="text-sm">
                            <button
                                type="button"
                                onClick={() => navigate('/login')}
                                className="font-medium text-blue-600 dark:text-blue-400 hover:text-blue-500 dark:hover:text-blue-300"
                            >
                                Back to login
                            </button>
                        </div>

                        <div className="text-sm">
                            <button
                                type="button"
                                onClick={handleResendCode}
                                className="font-medium text-blue-600 dark:text-blue-400 hover:text-blue-500 dark:hover:text-blue-300"
                            >
                                Resend code
                            </button>
                        </div>
                    </div>

                    <div>
                        <button
                            type="submit"
                            disabled={loading || verificationCode.replace(/\D/g, '').length !== 6}
                            className="group relative w-full flex justify-center py-2 px-4 border border-transparent text-sm font-medium rounded-md text-white bg-blue-600 hover:bg-blue-700 dark:bg-blue-600 dark:hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 dark:focus:ring-offset-slate-900 focus:ring-blue-500 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
                        >
                            {loading ? (
                                <svg className="animate-spin -ml-1 mr-3 h-5 w-5 text-white" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
                                    <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4"></circle>
                                    <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
                                </svg>
                            ) : null}
                            Verify and Continue
                        </button>
                    </div>
                </form>

                <div className="text-center">
                    <p className="text-xs text-slate-600 dark:text-slate-400">
                        Having trouble? Contact your administrator for assistance.
                    </p>
                </div>
            </div>
        </div>
    )
}

export default MFAVerificationPage