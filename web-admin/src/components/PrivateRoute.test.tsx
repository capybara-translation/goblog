import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen } from '@testing-library/react'
import { MemoryRouter, Routes, Route } from 'react-router-dom'
import { PrivateRoute } from './PrivateRoute'
import { useAuth } from '../hooks/useAuth'

// Mock useAuth
vi.mock('../hooks/useAuth', () => ({
  useAuth: vi.fn(),
}))

describe('PrivateRoute', () => {
  const mockUser = {
    id: 1,
    username: 'testuser',
    created_at: '2024-01-01T00:00:00Z',
    updated_at: '2024-01-01T00:00:00Z',
  }

  beforeEach(() => {
    vi.clearAllMocks()
  })

  const renderPrivateRoute = (initialEntries = ['/protected']) => {
    return render(
      <MemoryRouter initialEntries={initialEntries}>
        <Routes>
          <Route
            path="/protected"
            element={
              <PrivateRoute>
                <div>Protected Content</div>
              </PrivateRoute>
            }
          />
          <Route path="/login" element={<div>Login Page</div>} />
        </Routes>
      </MemoryRouter>
    )
  }

  describe('Loading state', () => {
    it('should show loading indicator when isLoading is true', () => {
      vi.mocked(useAuth).mockReturnValue({
        user: null,
        isLoading: true,
        login: vi.fn(),
        logout: vi.fn(),
        checkAuth: vi.fn(),
      })

      renderPrivateRoute()

      expect(screen.getByText('Loading...')).toBeInTheDocument()
      expect(screen.queryByText('Protected Content')).not.toBeInTheDocument()
      expect(screen.queryByText('Login Page')).not.toBeInTheDocument()
    })
  })

  describe('Unauthenticated state', () => {
    it('should redirect to login when user is null and not loading', () => {
      vi.mocked(useAuth).mockReturnValue({
        user: null,
        isLoading: false,
        login: vi.fn(),
        logout: vi.fn(),
        checkAuth: vi.fn(),
      })

      renderPrivateRoute()

      expect(screen.getByText('Login Page')).toBeInTheDocument()
      expect(screen.queryByText('Protected Content')).not.toBeInTheDocument()
      expect(screen.queryByText('Loading...')).not.toBeInTheDocument()
    })

    it('should not show protected content when user is null', () => {
      vi.mocked(useAuth).mockReturnValue({
        user: null,
        isLoading: false,
        login: vi.fn(),
        logout: vi.fn(),
        checkAuth: vi.fn(),
      })

      renderPrivateRoute()

      expect(screen.queryByText('Protected Content')).not.toBeInTheDocument()
    })
  })

  describe('Authenticated state', () => {
    it('should render children when user is authenticated', () => {
      vi.mocked(useAuth).mockReturnValue({
        user: mockUser,
        isLoading: false,
        login: vi.fn(),
        logout: vi.fn(),
        checkAuth: vi.fn(),
      })

      renderPrivateRoute()

      expect(screen.getByText('Protected Content')).toBeInTheDocument()
      expect(screen.queryByText('Loading...')).not.toBeInTheDocument()
      expect(screen.queryByText('Login Page')).not.toBeInTheDocument()
    })

    it('should render complex children correctly', () => {
      vi.mocked(useAuth).mockReturnValue({
        user: mockUser,
        isLoading: false,
        login: vi.fn(),
        logout: vi.fn(),
        checkAuth: vi.fn(),
      })

      render(
        <MemoryRouter initialEntries={['/dashboard']}>
          <Routes>
            <Route
              path="/dashboard"
              element={
                <PrivateRoute>
                  <div>
                    <h1>Dashboard</h1>
                    <p>Welcome, {mockUser.username}</p>
                    <button>Action</button>
                  </div>
                </PrivateRoute>
              }
            />
            <Route path="/login" element={<div>Login Page</div>} />
          </Routes>
        </MemoryRouter>
      )

      expect(screen.getByText('Dashboard')).toBeInTheDocument()
      expect(screen.getByText(`Welcome, ${mockUser.username}`)).toBeInTheDocument()
      expect(screen.getByRole('button', { name: 'Action' })).toBeInTheDocument()
    })
  })

  describe('State transitions', () => {
    // Helper function: create test component
    const createTestComponent = () => (
      <MemoryRouter initialEntries={['/protected']}>
        <Routes>
          <Route
            path="/protected"
            element={
              <PrivateRoute>
                <div>Protected Content</div>
              </PrivateRoute>
            }
          />
          <Route path="/login" element={<div>Login Page</div>} />
        </Routes>
      </MemoryRouter>
    )

    it('should transition from loading to authenticated', () => {
      // Initial state: loading
      vi.mocked(useAuth).mockReturnValue({
        user: null,
        isLoading: true,
        login: vi.fn(),
        logout: vi.fn(),
        checkAuth: vi.fn(),
      })

      const { rerender } = render(createTestComponent())
      expect(screen.getByText('Loading...')).toBeInTheDocument()

      // Loading complete → authenticated
      vi.mocked(useAuth).mockReturnValue({
        user: mockUser,
        isLoading: false,
        login: vi.fn(),
        logout: vi.fn(),
        checkAuth: vi.fn(),
      })

      rerender(createTestComponent())
      expect(screen.getByText('Protected Content')).toBeInTheDocument()
      expect(screen.queryByText('Loading...')).not.toBeInTheDocument()
    })

    it('should transition from loading to unauthenticated', () => {
      // Initial state: loading
      vi.mocked(useAuth).mockReturnValue({
        user: null,
        isLoading: true,
        login: vi.fn(),
        logout: vi.fn(),
        checkAuth: vi.fn(),
      })

      const { rerender } = render(createTestComponent())
      expect(screen.getByText('Loading...')).toBeInTheDocument()

      // Loading complete → unauthenticated
      vi.mocked(useAuth).mockReturnValue({
        user: null,
        isLoading: false,
        login: vi.fn(),
        logout: vi.fn(),
        checkAuth: vi.fn(),
      })

      rerender(createTestComponent())
      expect(screen.getByText('Login Page')).toBeInTheDocument()
      expect(screen.queryByText('Loading...')).not.toBeInTheDocument()
    })
  })
})
