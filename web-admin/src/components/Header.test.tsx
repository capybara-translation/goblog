import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { Header } from './Header'
import { useAuth } from '../hooks/useAuth'

// useAuth をモック
vi.mock('../hooks/useAuth', () => ({
  useAuth: vi.fn(),
}))

// React Router のナビゲーションをモック
const mockNavigate = vi.fn()
vi.mock('react-router-dom', async () => {
  const actual = await vi.importActual('react-router-dom')
  return {
    ...actual,
    useNavigate: () => mockNavigate,
  }
})

describe('Header', () => {
  const mockUser = {
    id: 1,
    username: 'testuser',
    created_at: '2024-01-01T00:00:00Z',
    updated_at: '2024-01-01T00:00:00Z',
  }

  const mockLogout = vi.fn()

  beforeEach(() => {
    vi.clearAllMocks()
    vi.mocked(useAuth).mockReturnValue({
      user: mockUser,
      isLoading: false,
      login: vi.fn(),
      logout: mockLogout,
      checkAuth: vi.fn(),
    })
  })

  const renderHeader = () => {
    return render(
      <MemoryRouter initialEntries={['/admin/posts']}>
        <Header />
      </MemoryRouter>
    )
  }

  describe('Rendering', () => {
    it('should render the app title', () => {
      renderHeader()
      expect(screen.getByText('goblog 管理画面')).toBeInTheDocument()
    })

    it('should render navigation links', () => {
      renderHeader()
      expect(screen.getByText('記事一覧')).toBeInTheDocument()
      expect(screen.getByText('新規作成')).toBeInTheDocument()
    })

    it('should render user information', () => {
      renderHeader()
      expect(screen.getByText('testuser')).toBeInTheDocument()
    })

    it('should render logout button', () => {
      renderHeader()
      expect(screen.getByRole('button', { name: 'ログアウト' })).toBeInTheDocument()
    })

    it('should have correct links', () => {
      renderHeader()

      const titleLink = screen.getByText('goblog 管理画面').closest('a')
      expect(titleLink).toHaveAttribute('href', '/posts')

      const postsLink = screen.getByText('記事一覧').closest('a')
      expect(postsLink).toHaveAttribute('href', '/posts')

      const newPostLink = screen.getByText('新規作成').closest('a')
      expect(newPostLink).toHaveAttribute('href', '/posts/new')
    })
  })

  describe('Logout functionality', () => {
    it('should call logout and navigate on logout button click', async () => {
      const user = userEvent.setup()
      mockLogout.mockResolvedValue(undefined)

      renderHeader()

      const logoutButton = screen.getByRole('button', { name: 'ログアウト' })
      await user.click(logoutButton)

      await waitFor(() => {
        expect(mockLogout).toHaveBeenCalledTimes(1)
      })

      expect(mockNavigate).toHaveBeenCalledWith('/login')
    })

    it('should handle logout error gracefully', async () => {
      const user = userEvent.setup()
      const consoleErrorSpy = vi.spyOn(console, 'error').mockImplementation(() => {})
      const error = new Error('Logout failed')
      mockLogout.mockRejectedValue(error)

      renderHeader()

      const logoutButton = screen.getByRole('button', { name: 'ログアウト' })
      await user.click(logoutButton)

      await waitFor(() => {
        expect(mockLogout).toHaveBeenCalledTimes(1)
      })

      expect(consoleErrorSpy).toHaveBeenCalledWith('Logout failed:', error)
      expect(mockNavigate).not.toHaveBeenCalled()

      consoleErrorSpy.mockRestore()
    })
  })

  describe('User display', () => {
    it('should display username when user is logged in', () => {
      renderHeader()
      expect(screen.getByText('testuser')).toBeInTheDocument()
    })

    it('should not display username when user is null', () => {
      vi.mocked(useAuth).mockReturnValue({
        user: null,
        isLoading: false,
        login: vi.fn(),
        logout: mockLogout,
        checkAuth: vi.fn(),
      })

      renderHeader()
      expect(screen.queryByText('testuser')).not.toBeInTheDocument()
    })
  })
})
