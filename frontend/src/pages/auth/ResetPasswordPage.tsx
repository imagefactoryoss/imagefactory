import { authService } from '@/services/authService'
import React, { useEffect, useState } from 'react'
import toast from 'react-hot-toast'
import { useNavigate, useSearchParams } from 'react-router-dom'

const ResetPasswordPage: React.FC = () => {
    const [searchParams] = useSearchParams()
    const navigate = useNavigate()
    const [formData, setFormData] = useState({
        password: '',
        confirmPassword: '',
    })
    const [isLoading, setIsLoading] = useState(false)
    const [isValidating, setIsValidating] = useState(true)
    const [isTokenValid, setIsTokenValid] = useState(false)

    useEffect(() => {
        const tokenParam = searchParams.get('token')
        if (!tokenParam) {
            toast.error('Invalid reset link')
            navigate('/login')
            return
        }

        validateToken()
    }, [searchParams, navigate])

    const validateToken = async () => {
        const tokenParam = searchParams.get('token')
        if (!tokenParam) return

        try {
            const response = await authService.validateResetToken(tokenParam)
            if (response.valid) {
                setIsTokenValid(true)
            } else {
                toast.error('This reset link has expired or is invalid')
                navigate('/login')
            }
        } catch (error: any) {
            toast.error('This reset link has expired or is invalid')
            navigate('/login')
        } finally {
            setIsValidating(false)
        }
    }

    const validatePassword = (password: string): string | null => {
        if (password.length < 8) {
            return 'Password must be at least 8 characters long'
        }
        if (!/(?=.*[a-z])/.test(password)) {
            return 'Password must contain at least one lowercase letter'
        }
        if (!/(?=.*[A-Z])/.test(password)) {
            return 'Password must contain at least one uppercase letter'
        }
        if (!/(?=.*\d)/.test(password)) {
            return 'Password must contain at least one number'
        }
        return null
    }

    const handleSubmit = async (e: React.FormEvent) => {
        e.preventDefault()
        // setErrors({})

        // Validate passwords
        const passwordError = validatePassword(formData.password)
        if (passwordError) {
            // setErrors({ password: passwordError })
            return
        }

        if (formData.password !== formData.confirmPassword) {
            // setErrors({ confirmPassword: 'Passwords do not match' })
            return
        }

        setIsLoading(true)

        try {
            const token = searchParams.get('token')
            if (!token) {
                toast.error('Invalid reset token')
                navigate('/login')
                return
            }

            await authService.resetPassword(token, formData.password)
            toast.success('Password reset successfully! You can now sign in with your new password.')
            navigate('/login')
        } catch (error: any) {
            if (error.message.includes('expired') || error.message.includes('invalid')) {
                toast.error('This reset link has expired. Please request a new one.')
                navigate('/forgot-password')
            } else {
                toast.error(error.message || 'Failed to reset password')
            }
        } finally {
            setIsLoading(false)
        }
    }

    const getPasswordStrength = (password: string): { score: number; label: string; color: string } => {
        let score = 0
        if (password.length >= 8) score++
        if (/(?=.*[a-z])/.test(password)) score++
        if (/(?=.*[A-Z])/.test(password)) score++
        if (/(?=.*\d)/.test(password)) score++
        if (/(?=.*[!@#$%^&*])/.test(password)) score++

        if (score <= 2) return { score, label: 'Weak', color: 'bg-red-500' }
        if (score <= 3) return { score, label: 'Fair', color: 'bg-yellow-500' }
        if (score <= 4) return { score, label: 'Good', color: 'bg-blue-500' }
        return { score, label: 'Strong', color: 'bg-green-500' }
    }

    if (isValidating) {
        return (
            <div className="min-h-screen flex items-center justify-center bg-gray-50">
                <div className="text-center">
                    <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600 mx-auto"></div>
                    <p className="mt-4 text-gray-600">Validating reset link...</p>
                </div>
            </div>
        )
    }

    if (!isTokenValid) {
        return null // Will redirect in useEffect
    }

    const passwordStrength = getPasswordStrength(formData.password)

    return (
        <div className="min-h-screen flex items-center justify-center bg-gray-50 py-12 px-4 sm:px-6 lg:px-8">
            <div className="max-w-md w-full space-y-8">
                <div>
                    <h2 className="mt-6 text-center text-3xl font-extrabold text-gray-900">
                        Set new password
                    </h2>
                    <p className="mt-2 text-center text-sm text-gray-600">
                        Enter your new password below.
                    </p>
                </div>

                <form className="mt-8 space-y-6" onSubmit={handleSubmit}>
                    <div>
                        <label htmlFor="password" className="block text-sm font-medium text-gray-700 mb-1">
                            New Password
                        </label>
                        <input
                            id="password"
                            name="password"
                            type="password"
                            autoComplete="new-password"
                            required
                            value={formData.password}
                            onChange={(e) => setFormData({ ...formData, password: e.target.value })}
                            placeholder="Enter your new password"
                            className="appearance-none rounded-md relative block w-full px-3 py-2 border border-gray-300 placeholder-gray-500 text-gray-900 focus:outline-none focus:ring-blue-500 focus:border-blue-500 focus:z-10 sm:text-sm"
                            disabled={isLoading}
                        />
                        {formData.password && (
                            <div className="mt-2">
                                <div className="flex items-center space-x-2">
                                    <div className="flex-1 bg-gray-200 rounded-full h-2">
                                        <div
                                            className={`h-2 rounded-full transition-all duration-300 ${passwordStrength.color}`}
                                            style={{ width: `${(passwordStrength.score / 5) * 100}%` }}
                                        ></div>
                                    </div>
                                    <span className="text-sm text-gray-600">{passwordStrength.label}</span>
                                </div>
                                <div className="mt-1 text-xs text-gray-500">
                                    Password must contain at least 8 characters with uppercase, lowercase, and numbers
                                </div>
                            </div>
                        )}
                    </div>

                    <div>
                        <label htmlFor="confirmPassword" className="block text-sm font-medium text-gray-700 mb-1">
                            Confirm New Password
                        </label>
                        <input
                            id="confirmPassword"
                            name="confirmPassword"
                            type="password"
                            autoComplete="new-password"
                            required
                            value={formData.confirmPassword}
                            onChange={(e) => setFormData({ ...formData, confirmPassword: e.target.value })}
                            placeholder="Confirm your new password"
                            className="appearance-none rounded-md relative block w-full px-3 py-2 border border-gray-300 placeholder-gray-500 text-gray-900 focus:outline-none focus:ring-blue-500 focus:border-blue-500 focus:z-10 sm:text-sm"
                            disabled={isLoading}
                        />
                    </div>

                    <div>
                        <button
                            type="submit"
                            disabled={isLoading}
                            className="group relative w-full flex justify-center py-2 px-4 border border-transparent text-sm font-medium rounded-md text-white bg-blue-600 hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 disabled:opacity-50 disabled:cursor-not-allowed"
                        >
                            {isLoading ? 'Resetting...' : 'Reset Password'}
                        </button>
                    </div>
                </form>
            </div>
        </div>
    )
}

export default ResetPasswordPage