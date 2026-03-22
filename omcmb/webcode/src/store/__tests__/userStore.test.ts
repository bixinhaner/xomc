import { describe, it, expect, beforeEach } from 'vitest'
import { useUserStore } from '@/store/userStore'
import type { User } from '@/types/system'

const mockUser: User = {
  id: 'user-001',
  username: 'testuser',
  displayName: 'Test User',
  email: 'test@example.com',
  phone: '13800138000',
  role: 'operator',
  status: 'active',
  lastLoginTime: '2026-01-01T00:00:00Z',
  createTime: '2025-01-01T00:00:00Z',
}

describe('userStore', () => {
  beforeEach(() => {
    // Reset store to initial state before each test
    useUserStore.setState({
      currentUser: null,
      accessToken: null,
      refreshToken: null,
      tokenExpiresAt: null,
      isAuthenticated: false,
      permissions: [],
      loading: false,
    })
  })

  it('has correct initial state', () => {
    const state = useUserStore.getState()
    expect(state.currentUser).toBeNull()
    expect(state.accessToken).toBeNull()
    expect(state.isAuthenticated).toBe(false)
    expect(state.permissions).toEqual([])
    expect(state.loading).toBe(false)
  })

  it('login sets user and marks authenticated', () => {
    useUserStore.getState().login(mockUser)
    const state = useUserStore.getState()
    expect(state.currentUser).toEqual(mockUser)
    expect(state.isAuthenticated).toBe(true)
  })

  it('setTokenPair stores both tokens and expiry', () => {
    const expiresAt = new Date(Date.now() + 3600_000).toISOString()
    useUserStore.getState().setTokenPair({
      access_token: 'access-abc',
      refresh_token: 'refresh-xyz',
      expires_at: expiresAt,
      token_type: 'Bearer',
    })

    const state = useUserStore.getState()
    expect(state.accessToken).toBe('access-abc')
    expect(state.refreshToken).toBe('refresh-xyz')
    expect(state.isAuthenticated).toBe(true)
    expect(state.tokenExpiresAt).toBe(new Date(expiresAt).getTime())
  })

  it('clearAuth resets all auth state', () => {
    useUserStore.getState().login(mockUser)
    useUserStore.getState().setTokenPair({
      access_token: 'token',
      refresh_token: 'refresh',
      expires_at: new Date().toISOString(),
    })
    useUserStore.getState().setPermissions(['device:read'])

    useUserStore.getState().clearAuth()
    const state = useUserStore.getState()
    expect(state.currentUser).toBeNull()
    expect(state.accessToken).toBeNull()
    expect(state.refreshToken).toBeNull()
    expect(state.isAuthenticated).toBe(false)
    expect(state.permissions).toEqual([])
  })

  it('isTokenExpired returns true when no token expiry set', () => {
    expect(useUserStore.getState().isTokenExpired()).toBe(true)
  })

  it('isTokenExpired returns false when token is still valid', () => {
    const futureExpiry = new Date(Date.now() + 3600_000).toISOString()
    useUserStore.getState().setTokenPair({
      access_token: 'token',
      refresh_token: 'refresh',
      expires_at: futureExpiry,
    })
    expect(useUserStore.getState().isTokenExpired()).toBe(false)
  })

  it('isTokenExpired returns true when token has expired', () => {
    const pastExpiry = new Date(Date.now() - 60_000).toISOString()
    useUserStore.getState().setTokenPair({
      access_token: 'token',
      refresh_token: 'refresh',
      expires_at: pastExpiry,
    })
    expect(useUserStore.getState().isTokenExpired()).toBe(true)
  })

  it('hasPermission returns true for admin regardless of permissions list', () => {
    const adminUser: User = { ...mockUser, role: 'admin' }
    useUserStore.getState().login(adminUser)
    expect(useUserStore.getState().hasPermission('anything')).toBe(true)
  })

  it('hasPermission checks permissions list for non-admin users', () => {
    useUserStore.getState().login(mockUser) // role: operator
    useUserStore.getState().setPermissions(['device:read', 'device:write'])

    expect(useUserStore.getState().hasPermission('device:read')).toBe(true)
    expect(useUserStore.getState().hasPermission('admin:manage')).toBe(false)
  })

  it('hasPermission returns false when no user is logged in', () => {
    expect(useUserStore.getState().hasPermission('device:read')).toBe(false)
  })
})
