import { renderHook } from '@testing-library/react'
import { act } from 'react'
import { afterEach, describe, expect, it } from 'vitest'
import { useDashboardPath } from '@/hooks/useAccess'
import { useAuthStore } from '@/store/auth'

const resetAuthState = () => {
  useAuthStore.setState({
    user: null,
    token: null,
    refreshToken: null,
    tokenExpiry: null,
    isAuthenticated: false,
    isLoading: false,
    avatar: undefined,
    preferences: undefined,
    roles: undefined,
    groups: undefined,
    rolesByTenant: undefined,
    isSystemAdmin: undefined,
    canAccessAdmin: undefined,
    defaultLandingRoute: undefined,
    setupRequired: false,
    requiresPasswordChange: false,
  })
}

describe('useDashboardPath', () => {
  afterEach(() => {
    act(() => {
      resetAuthState()
    })
  })

  it('uses backend-provided default landing route', () => {
    act(() => {
      useAuthStore.setState({
        defaultLandingRoute: '/reviewer/dashboard',
      })
    })
    const { result } = renderHook(() => useDashboardPath())
    expect(result.current).toBe('/reviewer/dashboard')
  })

  it('fails closed to /no-access when landing route is missing', () => {
    const { result } = renderHook(() => useDashboardPath())
    expect(result.current).toBe('/no-access')
  })
})
