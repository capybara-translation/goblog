import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { Header } from './Header'
import { useAuth } from '../hooks/useAuth'

// Mock useAuth
vi.mock('../hooks/useAuth', () => ({
  useAuth: vi.fn(),
}))

// Mock React Router navigation
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
      // Title link opens public site in new tab (text comes from VITE_BLOG_TITLE env var)
      const titleLink = screen.getAllByRole('link')[0]
      expect(titleLink).toHaveAttribute('href', '/')
      expect(titleLink).toHaveAttribute('target', '_blank')
      expect(titleLink).toHaveClass('text-xl', 'font-bold')
    })

    it('should render navigation links', () => {
      renderHeader()
      expect(screen.getByText('Posts')).toBeInTheDocument()
      expect(screen.getByText('New Post')).toBeInTheDocument()
    })

    it('should render user information', () => {
      renderHeader()
      expect(screen.getByText('testuser')).toBeInTheDocument()
    })

    it('should render logout button', () => {
      renderHeader()
      expect(screen.getByRole('button', { name: 'Logout' })).toBeInTheDocument()
    })

    it('should have correct links', () => {
      renderHeader()

      const links = screen.getAllByRole('link')
      // First link is the title (opens public site)
      expect(links[0]).toHaveAttribute('href', '/')

      const postsLink = screen.getByText('Posts').closest('a')
      expect(postsLink).toHaveAttribute('href', '/posts')

      const newPostLink = screen.getByText('New Post').closest('a')
      expect(newPostLink).toHaveAttribute('href', '/posts/new')
    })

    it('should have keyboard tab order matching desktop visual order (Brand -> Posts -> New Post -> Reactions -> Devices -> Logout)', async () => {
      const user = userEvent.setup()
      renderHeader()

      const titleLink = screen.getAllByRole('link').find(
        (el) => el.getAttribute('target') === '_blank'
      )
      const postsLink = screen.getByText('Posts').closest('a')
      const newPostLink = screen.getByText('New Post').closest('a')
      const reactionsLink = screen.getByRole('link', { name: 'Reactions' })
      const devicesLink = screen.getByRole('link', { name: 'Devices' })
      const logoutButton = screen.getByRole('button', { name: 'Logout' })

      await user.tab()
      expect(titleLink).toHaveFocus()

      await user.tab()
      expect(postsLink).toHaveFocus()

      await user.tab()
      expect(newPostLink).toHaveFocus()

      await user.tab()
      expect(reactionsLink).toHaveFocus()

      await user.tab()
      expect(devicesLink).toHaveFocus()

      await user.tab()
      expect(logoutButton).toHaveFocus()
    })

    it('shows the Reactions nav link', () => {
      renderHeader()
      expect(screen.getByRole('link', { name: 'Reactions' })).toBeInTheDocument()
    })
  })

  describe('Logout functionality', () => {
    it('should call logout and navigate on logout button click', async () => {
      const user = userEvent.setup()
      mockLogout.mockResolvedValue(undefined)

      renderHeader()

      const logoutButton = screen.getByRole('button', { name: 'Logout' })
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

      const logoutButton = screen.getByRole('button', { name: 'Logout' })
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
