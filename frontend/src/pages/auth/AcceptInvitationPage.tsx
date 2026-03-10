import { useEffect, useState } from 'react'
import { toast } from 'react-hot-toast'
import { useNavigate, useSearchParams } from 'react-router-dom'

import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'

import { invitationService } from '@/services/invitationService'

export default function AcceptInvitationPage() {
    const [searchParams] = useSearchParams()
    const navigate = useNavigate()
    const [isLoading, setIsLoading] = useState(false)
    const [isLDAP, setIsLDAP] = useState(false)
    const [formData, setFormData] = useState({
        firstName: '',
        lastName: '',
        password: '',
        confirmPassword: ''
    })

    const token = searchParams.get('token')

    useEffect(() => {
        if (!token) {
            toast.error('Invalid invitation link')
            navigate('/login')
        }
    }, [token, navigate])

    const handleSubmit = async (e: React.FormEvent) => {
        e.preventDefault()

        if (!token) return

        // Password validation only required for non-LDAP users
        if (!isLDAP) {
            if (formData.password !== formData.confirmPassword) {
                toast.error('Passwords do not match')
                return
            }

            if (formData.password.length < 8) {
                toast.error('Password must be at least 8 characters long')
                return
            }
        }

        setIsLoading(true)

        try {
            await invitationService.acceptInvitation({
                token,
                first_name: formData.firstName,
                last_name: formData.lastName,
                password: isLDAP ? '' : formData.password,
                is_ldap: isLDAP
            })

            toast.success('Account created successfully! You can now log in.')
            navigate('/login')
        } catch (error: any) {
            toast.error(error?.message || 'Failed to accept invitation')
        } finally {
            setIsLoading(false)
        }
    }

    const handleInputChange = (e: React.ChangeEvent<HTMLInputElement>) => {
        const { name, value } = e.target
        setFormData(prev => ({
            ...prev,
            [name]: value
        }))
    }

    if (!token) {
        return null
    }

    return (
        <div className="min-h-screen flex items-center justify-center bg-slate-50 dark:bg-slate-950 px-4">
            <Card className="w-full max-w-md">
                <CardHeader className="text-center">
                    <CardTitle className="text-2xl">Accept Invitation</CardTitle>
                    <CardDescription>
                        Complete your account setup to join the team
                    </CardDescription>
                </CardHeader>
                <CardContent>
                    <form onSubmit={handleSubmit} className="space-y-4">
                        {/* LDAP User Toggle - shown if accessible via query param */}
                        {typeof window !== 'undefined' && new URLSearchParams(window.location.search).has('ldap') && (
                            <div className="p-3 bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-800 rounded-lg">
                                <label className="flex items-center gap-2 cursor-pointer">
                                    <input
                                        type="checkbox"
                                        checked={isLDAP}
                                        onChange={(e) => setIsLDAP(e.target.checked)}
                                        className="w-4 h-4 rounded border-gray-300"
                                    />
                                    <span className="text-sm text-blue-900 dark:text-blue-100">
                                        This is an LDAP/Directory user (no password needed)
                                    </span>
                                </label>
                            </div>
                        )}

                        <div className="grid grid-cols-2 gap-4">
                            <div className="space-y-2">
                                <Label htmlFor="firstName">First Name</Label>
                                <Input
                                    id="firstName"
                                    name="firstName"
                                    type="text"
                                    required
                                    value={formData.firstName}
                                    onChange={handleInputChange}
                                    placeholder="John"
                                />
                            </div>
                            <div className="space-y-2">
                                <Label htmlFor="lastName">Last Name</Label>
                                <Input
                                    id="lastName"
                                    name="lastName"
                                    type="text"
                                    required
                                    value={formData.lastName}
                                    onChange={handleInputChange}
                                    placeholder="Doe"
                                />
                            </div>
                        </div>

                        {/* Password fields - only show for non-LDAP users */}
                        {!isLDAP && (
                            <>
                                <div className="space-y-2">
                                    <Label htmlFor="password">Password</Label>
                                    <Input
                                        id="password"
                                        name="password"
                                        type="password"
                                        required={!isLDAP}
                                        value={formData.password}
                                        onChange={handleInputChange}
                                        placeholder="Enter your password"
                                        minLength={8}
                                    />
                                </div>

                                <div className="space-y-2">
                                    <Label htmlFor="confirmPassword">Confirm Password</Label>
                                    <Input
                                        id="confirmPassword"
                                        name="confirmPassword"
                                        type="password"
                                        required={!isLDAP}
                                        value={formData.confirmPassword}
                                        onChange={handleInputChange}
                                        placeholder="Confirm your password"
                                        minLength={8}
                                    />
                                </div>
                            </>
                        )}

                        {/* LDAP message */}
                        {isLDAP && (
                            <div className="p-3 bg-green-50 dark:bg-green-900/20 border border-green-200 dark:border-green-800 rounded-lg">
                                <p className="text-sm text-green-900 dark:text-green-100">
                                    You'll authenticate using your LDAP credentials
                                </p>
                            </div>
                        )}

                        <Button
                            type="submit"
                            className="w-full"
                            disabled={isLoading}
                        >
                            {isLoading ? 'Creating Account...' : 'Create Account'}
                        </Button>
                    </form>

                    <div className="mt-6 text-center">
                        <p className="text-sm text-slate-600 dark:text-slate-400">
                            Already have an account?{' '}
                            <button
                                onClick={() => navigate('/login')}
                                className="text-blue-600 hover:text-blue-500 dark:text-blue-400 dark:hover:text-blue-300 font-medium"
                            >
                                Sign in
                            </button>
                        </p>
                    </div>
                </CardContent>
            </Card>
        </div>
    )
}