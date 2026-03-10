import {
    BarChart3,
    ChevronRight,
    Database,
    Lock,
    Settings,
    Shield,
    Users,
    Wrench
} from 'lucide-react'
import React from 'react'
import { Link, useLocation } from 'react-router-dom'

interface AdminLayoutProps {
    children: React.ReactNode
}

const AdminLayout: React.FC<AdminLayoutProps> = ({ children }) => {
    const location = useLocation()

    const adminSections = [
        {
            title: 'System Management',
            items: [
                {
                    name: 'Tool Availability',
                    path: '/admin/tools',
                    icon: Wrench,
                    description: 'Configure available build tools and services'
                },
                {
                    name: 'System Settings',
                    path: '/admin/settings',
                    icon: Settings,
                    description: 'General system configuration'
                }
            ]
        },
        {
            title: 'Access Management',
            items: [
                {
                    name: 'Users',
                    path: '/admin/users',
                    icon: Users,
                    description: 'Manage system users and permissions'
                },
                {
                    name: 'Roles',
                    path: '/admin/access/roles',
                    icon: Shield,
                    description: 'Manage user roles and access levels'
                },
                {
                    name: 'Permissions',
                    path: '/admin/access/permissions',
                    icon: Lock,
                    description: 'Configure system permissions'
                },
                {
                    name: 'Permission Definitions',
                    path: '/admin/access/permission-definitions',
                    icon: Lock,
                    description: 'Define and manage permission types'
                }
            ]
        },
        {
            title: 'Organization',
            items: [
                {
                    name: 'Tenants',
                    path: '/admin/tenants',
                    icon: Database,
                    description: 'Manage tenant organizations'
                }
            ]
        },
        {
            title: 'Security & Monitoring',
            items: [
                {
                    name: 'Security Policies',
                    path: '/admin/security',
                    icon: Shield,
                    description: 'Security settings and policies'
                },
                {
                    name: 'System Health',
                    path: '/admin/health',
                    icon: BarChart3,
                    description: 'System monitoring and health checks'
                },
                {
                    name: 'Audit Logs',
                    path: '/admin/audit-logs',
                    icon: BarChart3,
                    description: 'System audit logs and activity'
                }
            ]
        }
    ]

    return (
        <div className="min-h-screen bg-gray-50 dark:bg-gray-900">
            <div className="flex">
                {/* Sidebar */}
                <div className="w-80 bg-white dark:bg-gray-800 shadow-sm border-r border-gray-200 dark:border-gray-700">
                    <div className="p-6">
                        <h1 className="text-xl font-bold text-gray-900 dark:text-white">
                            Admin Panel
                        </h1>
                        <p className="text-sm text-gray-600 dark:text-gray-400 mt-1">
                            System administration and configuration
                        </p>
                    </div>

                    <nav className="px-4 pb-4">
                        {adminSections.map((section, sectionIndex) => (
                            <div key={section.title} className={sectionIndex > 0 ? 'mt-8' : ''}>
                                <h3 className="px-3 text-xs font-semibold text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                                    {section.title}
                                </h3>
                                <div className="mt-2 space-y-1">
                                    {section.items.map((item) => {
                                        const isActive = location.pathname === item.path

                                        return (
                                            <Link
                                                key={item.path}
                                                to={item.path}
                                                className={`group flex items-center px-3 py-2 text-sm font-medium rounded-md transition-colors ${isActive
                                                    ? 'bg-blue-50 text-blue-700 dark:bg-blue-900 dark:text-blue-200'
                                                    : 'text-gray-700 hover:bg-gray-50 dark:text-gray-300 dark:hover:bg-gray-700'
                                                    }`}
                                            >
                                                <Wrench className={`mr-3 h-5 w-5 flex-shrink-0 ${isActive ? 'text-blue-500' : 'text-gray-400 group-hover:text-gray-500'
                                                    }`} />
                                                <div className="flex-1">
                                                    <div className="flex items-center">
                                                        {item.name}
                                                        {isActive && (
                                                            <ChevronRight className="ml-auto h-4 w-4 text-blue-500" />
                                                        )}
                                                    </div>
                                                    <p className="text-xs text-gray-500 dark:text-gray-400 mt-0.5">
                                                        {item.description}
                                                    </p>
                                                </div>
                                            </Link>
                                        )
                                    })}
                                </div>
                            </div>
                        ))}
                    </nav>
                </div>

                {/* Main Content */}
                <div className="flex-1">
                    <div className="p-8">
                        {children}
                    </div>
                </div>
            </div>
        </div>
    )
}

export default AdminLayout