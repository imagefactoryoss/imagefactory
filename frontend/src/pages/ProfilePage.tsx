import TenantRoleSelector from '@/components/TenantRoleSelector'
import TenantSelector from '@/components/TenantSelector'
import { profileService } from '@/services/profileService'
import { useTenantStore, type TenantContext } from '@/store/tenant'
import { UserRoleWithPermissions } from '@/types'
import React, { useEffect, useState } from 'react'

interface ProfileData {
    id: string
    email: string
    first_name: string
    last_name: string
    status: string
    is_active: boolean
    roles: UserRoleWithPermissions[]
    rolesByTenant?: Record<string, UserRoleWithPermissions[]>
    groups: Array<{
        id: string
        name: string
        role_type: string
        tenant_id: string
        is_admin: boolean
    }>
    avatar?: string
    created_at?: string
    has_multi_tenant: boolean
}

const ProfilePage: React.FC = () => {
    const { userTenants } = useTenantStore()
    const [profile, setProfile] = useState<ProfileData | null>(null)
    const [tenants, setTenants] = useState<TenantContext[]>([])
    const [loading, setLoading] = useState(true)
    const [error, setError] = useState<string | null>(null)

    useEffect(() => {
        fetchProfileData()
    }, [userTenants])

    // Note: Tenant data is now loaded globally in Layout component
    // This page focuses on profile-specific information

    const fetchProfileData = async () => {
        try {
            setLoading(true)
            setError(null)
            const profileData = await profileService.getProfile()

            // Transform profile data to include rolesByTenant from roles_by_tenant
            const transformedProfile: ProfileData = {
                id: profileData.id,
                email: profileData.email,
                first_name: profileData.first_name,
                last_name: profileData.last_name,
                status: profileData.status,
                is_active: profileData.is_active,
                roles: profileData.roles || [],
                rolesByTenant: profileData.roles_by_tenant || {},
                groups: profileData.groups || [],
                avatar: profileData.avatar,
                created_at: profileData.created_at,
                has_multi_tenant: profileData.has_multi_tenant || false,
            }

            setProfile(transformedProfile)
            // Use userTenants from store instead of loading separately
            setTenants(userTenants)
        } catch (err: any) {
            setError(err.response?.data?.error || 'Failed to load profile')
        } finally {
            setLoading(false)
        }
    }

    if (loading) {
        return (
            <div className="flex items-center justify-center min-h-screen">
                <div className="text-lg text-muted-foreground">Loading profile...</div>
            </div>
        )
    }

    if (error) {
        return (
            <div className="px-4 py-6 sm:px-6 lg:px-8">
                <div className="rounded-lg bg-red-50 p-4">
                    <div className="text-red-800">{error}</div>
                </div>
            </div>
        )
    }

    if (!profile) {
        return (
            <div className="px-4 py-6 sm:px-6 lg:px-8">
                <div className="text-lg text-muted-foreground">No profile data found</div>
            </div>
        )
    }

    const initials = profileService.getInitials(profile.first_name, profile.last_name)
    const avatarColor = profileService.getAvatarColor(profile.id)

    return (
        <div className="min-h-screen bg-slate-50 dark:bg-slate-900 px-4 py-6 sm:px-6 lg:px-8">
            <div className="max-w-2xl">
                {/* Profile Header */}
                <div className="mb-8">
                    <h1 className="text-3xl font-bold text-foreground">Profile</h1>
                    <p className="mt-2 text-muted-foreground">View and manage your account information</p>
                </div>

                {/* Avatar and Basic Info */}
                <div className="card mb-8 bg-white dark:bg-gray-800">
                    <div className="card-body">
                        <div className="flex items-center space-x-6">
                            {profile.avatar ? (
                                <img
                                    src={profile.avatar}
                                    alt={`${profile.first_name} ${profile.last_name}`}
                                    className="w-20 h-20 rounded-full object-cover"
                                />
                            ) : (
                                <div
                                    className={`w-20 h-20 rounded-full ${avatarColor} flex items-center justify-center text-white font-bold text-xl`}
                                >
                                    {initials}
                                </div>
                            )}
                            <div>
                                <h2 className="text-2xl font-semibold text-gray-900 dark:text-white">
                                    {profile.first_name} {profile.last_name}
                                </h2>
                                <p className="text-gray-600 dark:text-gray-400">{profile.email}</p>
                                <p className="text-sm text-gray-600 dark:text-gray-400 mt-1">
                                    Status:{' '}
                                    <span
                                        className={`inline-block px-2 py-1 rounded text-xs font-medium ${profile.is_active
                                            ? 'bg-green-100 text-green-800'
                                            : 'bg-red-100 text-red-800'
                                            }`}
                                    >
                                        {profile.status}
                                    </span>
                                </p>
                            </div>
                        </div>
                    </div>
                </div>

                {/* Tenant Context - Only show if user has multiple tenants or roles */}
                {userTenants && userTenants.length > 1 && (
                    <div className="card mb-8 bg-white dark:bg-gray-800">
                        <div className="card-header">
                            <h3 className="text-lg font-semibold text-gray-900 dark:text-white">Tenant Context</h3>
                            <p className="text-sm text-gray-600 dark:text-gray-400">Switch between tenants you have access to</p>
                        </div>
                        <div className="card-body">
                            <TenantSelector />
                            <div className="mt-4 text-sm text-gray-600 dark:text-gray-400">
                                <p>Changing your tenant context will affect all subsequent API requests and the data you can access.</p>
                            </div>
                        </div>
                    </div>
                )}

                {/* Tenant & Role Selection */}
                <TenantRoleSelector
                    rolesByTenant={profile?.rolesByTenant || {}}
                    tenants={tenants}
                    onSelectionChange={() => {
                    }}
                />

                {/* Account Information */}
                <div className="card mb-8 bg-white dark:bg-gray-800">
                    <div className="card-header">
                        <h3 className="text-lg font-semibold text-gray-900 dark:text-white">Account Information</h3>
                    </div>
                    <div className="card-body space-y-4">
                        <div>
                            <label className="text-sm font-medium text-gray-600 dark:text-gray-400">Email</label>
                            <p className="mt-1 text-gray-900 dark:text-white">{profile.email}</p>
                        </div>
                        <div className="grid grid-cols-2 gap-4">
                            <div>
                                <label className="text-sm font-medium text-gray-600 dark:text-gray-400">First Name</label>
                                <p className="mt-1 text-gray-900 dark:text-white">{profile.first_name}</p>
                            </div>
                            <div>
                                <label className="text-sm font-medium text-gray-600 dark:text-gray-400">Last Name</label>
                                <p className="mt-1 text-gray-900 dark:text-white">{profile.last_name}</p>
                            </div>
                        </div>
                        {profile.created_at && (
                            <div>
                                <label className="text-sm font-medium text-gray-600 dark:text-gray-400">Member Since</label>
                                <p className="mt-1 text-gray-900 dark:text-white">
                                    {new Date(profile.created_at).toLocaleDateString()}
                                </p>
                            </div>
                        )}
                    </div>
                </div>

                {/* Groups */}
                {profile.groups && profile.groups.length > 0 && (
                    <div className="card mb-8 bg-white dark:bg-gray-800">
                        <div className="card-header">
                            <h3 className="text-lg font-semibold text-gray-900 dark:text-white">Groups</h3>
                        </div>
                        <div className="card-body">
                            <div className="space-y-2">
                                {profile.groups.map((group) => {
                                    const tenant = userTenants.find(t => t.id === group.tenant_id)
                                    return (
                                        <div key={group.id} className="flex items-center justify-between p-3 bg-gray-50 dark:bg-gray-700 rounded">
                                            <div>
                                                <p className="font-medium text-gray-900 dark:text-white">{group.name}</p>
                                                <p className="text-sm text-gray-600 dark:text-gray-400">Role: {group.role_type}</p>
                                                {tenant && (
                                                    <p className="text-sm text-gray-600 dark:text-gray-400">Tenant: {tenant.name}</p>
                                                )}
                                            </div>
                                            {group.is_admin && (
                                                <span className="px-2 py-1 rounded text-xs font-medium bg-blue-100 dark:bg-blue-900/30 text-blue-800 dark:text-blue-200">
                                                    Admin
                                                </span>
                                            )}
                                        </div>
                                    )
                                })}
                            </div>
                        </div>
                    </div>
                )}

            </div>
        </div>
    )
}

export default ProfilePage
