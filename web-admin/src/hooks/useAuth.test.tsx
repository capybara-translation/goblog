import { describe, it, expect, vi, beforeEach } from 'vitest'
import { renderHook, waitFor } from '@testing-library/react'
import { AuthProvider, useAuth } from './useAuth'
import { apiClient } from '../api/client'
import type { ReactNode } from 'react'

// Mock apiClient
vi.mock('../api/client', () => ({
  apiClient: {
    me: vi.fn(),
    login: vi.fn(),
    logout: vi.fn(),
  },
}))

describe('useAuth', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  describe('AuthProvider', () => {
    it('should initialize and check auth on mount', async () => {
      const mockUser = {
        id: 1,
        username: 'testuser',
        created_at: '2024-01-01T00:00:00Z',
        updated_at: '2024-01-01T00:00:00Z',
      }

      vi.mocked(apiClient.me).mockResolvedValue(mockUser)

      const wrapper = ({ children }: { children: ReactNode }) => (
        <AuthProvider>{children}</AuthProvider>
      )

      const { result } = renderHook(() => useAuth(), { wrapper })

      // Initial state: user is null, isLoading is true
      expect(result.current.user).toBeNull()
      expect(result.current.isLoading).toBe(true)

      // Wait for checkAuth to complete
      await waitFor(() => {
        expect(result.current.isLoading).toBe(false)
      })

      // User info is set
      expect(result.current.user).toEqual(mockUser)
      expect(apiClient.me).toHaveBeenCalledTimes(1)
    })

    it('should set user to null if checkAuth fails', async () => {
      vi.mocked(apiClient.me).mockRejectedValue(new Error('Unauthorized'))

      const wrapper = ({ children }: { children: ReactNode }) => (
        <AuthProvider>{children}</AuthProvider>
      )

      const { result } = renderHook(() => useAuth(), { wrapper })

      await waitFor(() => {
        expect(result.current.isLoading).toBe(false)
      })

      expect(result.current.user).toBeNull()
    })
  })

  describe('login', () => {
    it('should login successfully and set user', async () => {
      const mockUser = {
        id: 1,
        username: 'testuser',
        created_at: '2024-01-01T00:00:00Z',
        updated_at: '2024-01-01T00:00:00Z',
      }

      // Make initial checkAuth fail
      vi.mocked(apiClient.me).mockRejectedValue(new Error('Not authenticated'))
      vi.mocked(apiClient.login).mockResolvedValue(mockUser)

      const wrapper = ({ children }: { children: ReactNode }) => (
        <AuthProvider>{children}</AuthProvider>
      )

      const { result } = renderHook(() => useAuth(), { wrapper })

      // Wait for initialization to complete
      await waitFor(() => {
        expect(result.current.isLoading).toBe(false)
      })

      expect(result.current.user).toBeNull()

      // Execute login (without await)
      result.current.login({
        username: 'testuser',
        password: 'password',
      })

      // Wait for state update inside waitFor
      await waitFor(() => {
        expect(result.current.user).toEqual(mockUser)
      })

      expect(apiClient.login).toHaveBeenCalledWith({
        username: 'testuser',
        password: 'password',
      })
    })

    it('should throw error if login fails', async () => {
      vi.mocked(apiClient.me).mockRejectedValue(new Error('Not authenticated'))
      vi.mocked(apiClient.login).mockRejectedValue(
        new Error('Invalid credentials')
      )

      const wrapper = ({ children }: { children: ReactNode }) => (
        <AuthProvider>{children}</AuthProvider>
      )

      const { result } = renderHook(() => useAuth(), { wrapper })

      await waitFor(() => {
        expect(result.current.isLoading).toBe(false)
      })

      await expect(
        result.current.login({
          username: 'wronguser',
          password: 'wrongpass',
        })
      ).rejects.toThrow('Invalid credentials')

      expect(result.current.user).toBeNull()
    })
  })

  describe('logout', () => {
    it('should logout successfully and clear user', async () => {
      const mockUser = {
        id: 1,
        username: 'testuser',
        created_at: '2024-01-01T00:00:00Z',
        updated_at: '2024-01-01T00:00:00Z',
      }

      vi.mocked(apiClient.me).mockResolvedValue(mockUser)
      vi.mocked(apiClient.logout).mockResolvedValue(undefined)

      const wrapper = ({ children }: { children: ReactNode }) => (
        <AuthProvider>{children}</AuthProvider>
      )

      const { result } = renderHook(() => useAuth(), { wrapper })

      await waitFor(() => {
        expect(result.current.isLoading).toBe(false)
      })

      expect(result.current.user).toEqual(mockUser)

      // Execute logout (without await)
      result.current.logout()

      // Wait for state update inside waitFor
      await waitFor(() => {
        expect(result.current.user).toBeNull()
      })

      expect(apiClient.logout).toHaveBeenCalledTimes(1)
    })
  })

  describe('checkAuth', () => {
    it('should update user when called manually', async () => {
      const mockUser = {
        id: 1,
        username: 'testuser',
        created_at: '2024-01-01T00:00:00Z',
        updated_at: '2024-01-01T00:00:00Z',
      }

      // First call fails, second call succeeds
      vi.mocked(apiClient.me)
        .mockRejectedValueOnce(new Error('Not authenticated'))
        .mockResolvedValueOnce(mockUser)

      const wrapper = ({ children }: { children: ReactNode }) => (
        <AuthProvider>{children}</AuthProvider>
      )

      const { result } = renderHook(() => useAuth(), { wrapper })

      await waitFor(() => {
        expect(result.current.isLoading).toBe(false)
      })

      expect(result.current.user).toBeNull()

      // Call checkAuth manually (without await)
      result.current.checkAuth()

      // Wait for state update inside waitFor
      await waitFor(() => {
        expect(result.current.user).toEqual(mockUser)
      })
    })
  })

  describe('useAuth hook', () => {
    it('should throw error when used outside AuthProvider', () => {
      // Suppress console error
      const consoleError = vi
        .spyOn(console, 'error')
        .mockImplementation(() => {})

      expect(() => {
        renderHook(() => useAuth())
      }).toThrow('useAuth must be used within an AuthProvider')

      consoleError.mockRestore()
    })
  })
})
