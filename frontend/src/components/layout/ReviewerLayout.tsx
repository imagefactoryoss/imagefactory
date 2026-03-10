import { useConfirmDialog } from '@/context/ConfirmDialogContext'
import { authService } from '@/services/authService'
import { profileService } from '@/services/profileService'
import { useAuthStore } from '@/store/auth'
import { useThemeStore } from '@/store/theme'
import React, { useEffect, useRef, useState } from 'react'
import { Link, useLocation, useNavigate } from 'react-router-dom'
import { TokenExpirationWarning } from '../TokenExpirationWarning'

interface ReviewerLayoutProps {
  children: React.ReactNode
}

const ReviewerLayout: React.FC<ReviewerLayoutProps> = ({ children }) => {
  const { user, avatar, logout } = useAuthStore()
  const { isDark, toggleTheme } = useThemeStore()
  const confirmDialog = useConfirmDialog()
  const navigate = useNavigate()
  const location = useLocation()
  const [sidebarOpen, setSidebarOpen] = useState(true)
  const [showProfileMenu, setShowProfileMenu] = useState(false)
  const profileMenuRef = useRef<HTMLDivElement | null>(null)

  const initials = user ? profileService.getInitials(user.name || user.email || 'U', '') : 'U'
  const avatarColor = user ? profileService.getAvatarColor(user.id) : 'bg-slate-500'

  const isActive = (href: string) => {
    if (href === '/reviewer/dashboard') {
      return location.pathname === '/reviewer/dashboard' || location.pathname === '/reviewer'
    }
    return location.pathname.startsWith(href)
  }

  const handleLogout = async () => {
    const confirmed = await confirmDialog({
      title: 'Log Out',
      message: 'Are you sure you want to logout?',
      confirmLabel: 'Log Out',
    })
    if (!confirmed) return
    try {
      await authService.logout()
    } catch {
      // Continue with local logout even if server-side logout fails.
    }
    logout()
    navigate('/login')
  }

  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (!profileMenuRef.current) return
      const target = event.target as Node
      if (!profileMenuRef.current.contains(target)) {
        setShowProfileMenu(false)
      }
    }
    document.addEventListener('mousedown', handleClickOutside)
    return () => document.removeEventListener('mousedown', handleClickOutside)
  }, [])

  return (
    <div className="min-h-screen bg-white dark:bg-slate-900 text-slate-900 dark:text-slate-50">
      <header className="sticky top-0 z-40 border-b border-slate-200 dark:border-slate-700 bg-white/95 dark:bg-slate-800/95 backdrop-blur supports-[backdrop-filter]:bg-white/60 dark:supports-[backdrop-filter]:bg-slate-800/60">
        <div className="px-4 sm:px-6 lg:px-8 py-3 flex items-center justify-between">
          <div className="flex items-center gap-4">
            <button
              type="button"
              onClick={() => setSidebarOpen(!sidebarOpen)}
              className="p-2 hover:bg-slate-100 dark:hover:bg-slate-700 rounded-lg transition-colors lg:hidden"
            >
              <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 6h16M4 12h16M4 18h16" />
              </svg>
            </button>
            <div>
              <h1 className="text-base font-semibold text-slate-900 dark:text-slate-100">Security Reviewer Console</h1>
              <p className="text-xs text-slate-600 dark:text-slate-400">Central quarantine and EPR approval queue</p>
            </div>
          </div>
          <div className="flex items-center gap-2">
            <button
              type="button"
              onClick={toggleTheme}
              className="rounded-md border border-slate-300 bg-white px-2.5 py-1.5 text-xs font-medium text-slate-700 hover:bg-slate-100 dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200 dark:hover:bg-slate-700"
            >
              {isDark ? 'Light' : 'Dark'}
            </button>
            <div className="relative" ref={profileMenuRef}>
              <button
                type="button"
                onClick={() => setShowProfileMenu((prev) => !prev)}
                className="flex items-center gap-2 rounded-md border border-slate-200 bg-slate-50 px-2 py-1 hover:bg-slate-100 dark:border-slate-700 dark:bg-slate-800 dark:hover:bg-slate-700"
                title={user?.email}
              >
                {avatar ? (
                  <img
                    src={avatar}
                    alt={user?.name || user?.email || 'Reviewer'}
                    className="h-7 w-7 rounded-full object-cover"
                  />
                ) : (
                  <div className={`flex h-7 w-7 items-center justify-center rounded-full text-xs font-semibold text-white ${avatarColor}`}>
                    {initials || 'U'}
                  </div>
                )}
                <span className="max-w-[180px] truncate text-xs font-medium text-slate-700 dark:text-slate-200">
                  {user?.name || user?.email}
                </span>
              </button>
              {showProfileMenu ? (
                <div className="absolute right-0 mt-2 w-56 rounded-lg border border-slate-200 bg-white shadow-lg dark:border-slate-700 dark:bg-slate-800 z-50">
                  <div className="border-b border-slate-200 p-3 dark:border-slate-700">
                    <p className="text-sm font-medium text-slate-900 dark:text-slate-100">{user?.name || 'User'}</p>
                    <p className="text-xs text-slate-600 dark:text-slate-400">{user?.email}</p>
                  </div>
                  <div className="py-1">
                    <Link
                      to="/profile"
                      onClick={() => setShowProfileMenu(false)}
                      className="block px-3 py-2 text-sm text-slate-800 hover:bg-slate-100 dark:text-slate-200 dark:hover:bg-slate-700"
                    >
                      View Profile
                    </Link>
                    <Link
                      to="/reviewer/quarantine/requests"
                      onClick={() => setShowProfileMenu(false)}
                      className="block px-3 py-2 text-sm text-slate-800 hover:bg-slate-100 dark:text-slate-200 dark:hover:bg-slate-700"
                    >
                      Open Quarantine Requests
                    </Link>
                    <button
                      type="button"
                      onClick={() => {
                        setShowProfileMenu(false)
                        void handleLogout()
                      }}
                      className="w-full border-t border-slate-200 px-3 py-2 text-left text-sm text-slate-800 hover:bg-slate-100 dark:border-slate-700 dark:text-slate-200 dark:hover:bg-slate-700"
                    >
                      Sign Out
                    </button>
                  </div>
                </div>
              ) : null}
            </div>
          </div>
        </div>
      </header>

      <div className="flex h-[calc(100vh-4rem)]">
        <aside
          className={`${sidebarOpen ? 'translate-x-0' : '-translate-x-full'
            } lg:translate-x-0 w-64 border-r border-slate-200 dark:border-slate-700 bg-slate-50 dark:bg-slate-800 transition-transform duration-200 ease-in-out fixed lg:sticky top-16 h-[calc(100vh-4rem)] z-30 lg:z-0`}
        >
          <nav className="p-4 space-y-2">
            <p className="px-3 py-1 text-xs font-semibold uppercase tracking-wider text-slate-500 dark:text-slate-400">
              Reviewer Workspace
            </p>
            <Link
              to="/reviewer/dashboard"
              className={`block rounded-lg px-3 py-2 text-sm transition-all ${
                isActive('/reviewer/dashboard')
                  ? 'bg-blue-600 text-white shadow-lg'
                  : 'text-slate-700 dark:text-slate-300 hover:bg-slate-100 dark:hover:bg-slate-700/50'
              }`}
            >
              Summary Dashboard
            </Link>
            <Link
              to="/reviewer/quarantine/requests"
              className={`block rounded-lg px-3 py-2 text-sm transition-all ${
                isActive('/reviewer/quarantine/requests')
                  ? 'bg-blue-600 text-white shadow-lg'
                  : 'text-slate-700 dark:text-slate-300 hover:bg-slate-100 dark:hover:bg-slate-700/50'
              }`}
            >
              Quarantine Requests
            </Link>
            <Link
              to="/reviewer/epr/approvals"
              className={`block rounded-lg px-3 py-2 text-sm transition-all ${
                isActive('/reviewer/epr/approvals')
                  ? 'bg-blue-600 text-white shadow-lg'
                  : 'text-slate-700 dark:text-slate-300 hover:bg-slate-100 dark:hover:bg-slate-700/50'
              }`}
            >
              EPR Approvals
            </Link>
          </nav>
        </aside>

        {sidebarOpen && (
          <div
            className="fixed inset-0 bg-black/20 backdrop-blur-sm z-20 lg:hidden"
            onClick={() => setSidebarOpen(false)}
          />
        )}

        <main className="flex-1 overflow-y-auto p-4 sm:p-6 lg:p-8">
          <TokenExpirationWarning />
          {children}
        </main>
      </div>
    </div>
  )
}

export default ReviewerLayout
