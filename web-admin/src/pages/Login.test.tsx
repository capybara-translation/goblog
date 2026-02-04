import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { Login } from './Login'
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

describe('Login', () => {
  const mockLogin = vi.fn()

  beforeEach(() => {
    vi.clearAllMocks()
    vi.mocked(useAuth).mockReturnValue({
      user: null,
      isLoading: false,
      login: mockLogin,
      logout: vi.fn(),
      checkAuth: vi.fn(),
    })
  })

  const renderLogin = () => {
    return render(
      <MemoryRouter>
        <Login />
      </MemoryRouter>
    )
  }

  describe('Rendering', () => {
    it('should render login form with title', () => {
      renderLogin()
      expect(screen.getByText(/Admin$/)).toBeInTheDocument()
      expect(screen.getByText('Please sign in')).toBeInTheDocument()
    })

    it('should render username field', () => {
      renderLogin()
      expect(screen.getByLabelText('Username')).toBeInTheDocument()
    })

    it('should render password field', () => {
      renderLogin()
      expect(screen.getByLabelText('Password')).toBeInTheDocument()
    })

    it('should render login button', () => {
      renderLogin()
      expect(screen.getByRole('button', { name: 'Sign in' })).toBeInTheDocument()
    })

    it('should not show error message initially', () => {
      renderLogin()
      expect(screen.queryByText(/Login failed/)).not.toBeInTheDocument()
    })
  })

  describe('User interaction', () => {
    it('should allow entering username', async () => {
      const user = userEvent.setup()
      renderLogin()

      const usernameInput = screen.getByLabelText('Username')
      await user.type(usernameInput, 'testuser')

      expect(usernameInput).toHaveValue('testuser')
    })

    it('should allow entering password', async () => {
      const user = userEvent.setup()
      renderLogin()

      const passwordInput = screen.getByLabelText('Password')
      await user.type(passwordInput, 'password123')

      expect(passwordInput).toHaveValue('password123')
    })

    it('should allow entering both username and password', async () => {
      const user = userEvent.setup()
      renderLogin()

      await user.type(screen.getByLabelText('Username'), 'admin')
      await user.type(screen.getByLabelText('Password'), 'secret')

      expect(screen.getByLabelText('Username')).toHaveValue('admin')
      expect(screen.getByLabelText('Password')).toHaveValue('secret')
    })
  })

  describe('Login success', () => {
    it('should call login with credentials and navigate to /posts', async () => {
      const user = userEvent.setup()
      mockLogin.mockResolvedValue(undefined)

      renderLogin()

      await user.type(screen.getByLabelText('Username'), 'admin')
      await user.type(screen.getByLabelText('Password'), 'password')
      await user.click(screen.getByRole('button', { name: 'Sign in' }))

      await waitFor(() => {
        expect(mockLogin).toHaveBeenCalledWith({
          username: 'admin',
          password: 'password',
        })
      })

      expect(mockNavigate).toHaveBeenCalledWith('/posts')
    })

    it('should not show error message on successful login', async () => {
      const user = userEvent.setup()
      mockLogin.mockResolvedValue(undefined)

      renderLogin()

      await user.type(screen.getByLabelText('Username'), 'admin')
      await user.type(screen.getByLabelText('Password'), 'password')
      await user.click(screen.getByRole('button', { name: 'Sign in' }))

      await waitFor(() => {
        expect(mockNavigate).toHaveBeenCalled()
      })

      expect(screen.queryByText(/Login failed/)).not.toBeInTheDocument()
    })
  })

  describe('Login error', () => {
    it('should show error message when login fails with Error', async () => {
      const user = userEvent.setup()
      mockLogin.mockRejectedValue(new Error('Authentication failed'))

      renderLogin()

      await user.type(screen.getByLabelText('Username'), 'wronguser')
      await user.type(screen.getByLabelText('Password'), 'wrongpass')
      await user.click(screen.getByRole('button', { name: 'Sign in' }))

      await waitFor(() => {
        expect(screen.getByText('Authentication failed')).toBeInTheDocument()
      })

      expect(mockNavigate).not.toHaveBeenCalled()
    })

    it('should show default error message for non-Error exceptions', async () => {
      const user = userEvent.setup()
      mockLogin.mockRejectedValue('Unknown error')

      renderLogin()

      await user.type(screen.getByLabelText('Username'), 'admin')
      await user.type(screen.getByLabelText('Password'), 'password')
      await user.click(screen.getByRole('button', { name: 'Sign in' }))

      await waitFor(() => {
        expect(screen.getByText('Login failed')).toBeInTheDocument()
      })
    })

    it('should clear previous error on new submission', async () => {
      const user = userEvent.setup()
      mockLogin.mockRejectedValueOnce(new Error('First error'))

      renderLogin()

      // 最初の失敗
      await user.type(screen.getByLabelText('Username'), 'user1')
      await user.type(screen.getByLabelText('Password'), 'pass1')
      await user.click(screen.getByRole('button', { name: 'Sign in' }))

      await waitFor(() => {
        expect(screen.getByText('First error')).toBeInTheDocument()
      })

      // 2回目の試行（成功）
      mockLogin.mockResolvedValue(undefined)
      await user.clear(screen.getByLabelText('Username'))
      await user.clear(screen.getByLabelText('Password'))
      await user.type(screen.getByLabelText('Username'), 'user2')
      await user.type(screen.getByLabelText('Password'), 'pass2')
      await user.click(screen.getByRole('button', { name: 'Sign in' }))

      await waitFor(() => {
        expect(screen.queryByText('First error')).not.toBeInTheDocument()
      })
    })
  })

  describe('Submitting state', () => {
    it('should show "Signing in..." during submission', async () => {
      const user = userEvent.setup()
      let resolveLogin: () => void
      mockLogin.mockReturnValue(
        new Promise((resolve) => {
          resolveLogin = resolve as () => void
        })
      )

      renderLogin()

      await user.type(screen.getByLabelText('Username'), 'admin')
      await user.type(screen.getByLabelText('Password'), 'password')
      await user.click(screen.getByRole('button', { name: 'Sign in' }))

      // 送信中の表示を確認
      await waitFor(() => {
        expect(screen.getByRole('button', { name: 'Signing in...' })).toBeInTheDocument()
      })

      // Sign inを完了
      resolveLogin!()

      // 元に戻ることを確認
      await waitFor(() => {
        expect(screen.queryByRole('button', { name: 'Signing in...' })).not.toBeInTheDocument()
      })
    })

    it('should disable form fields during submission', async () => {
      const user = userEvent.setup()
      let resolveLogin: () => void
      mockLogin.mockReturnValue(
        new Promise((resolve) => {
          resolveLogin = resolve as () => void
        })
      )

      renderLogin()

      const usernameInput = screen.getByLabelText('Username')
      const passwordInput = screen.getByLabelText('Password')
      const submitButton = screen.getByRole('button', { name: 'Sign in' })

      await user.type(usernameInput, 'admin')
      await user.type(passwordInput, 'password')
      await user.click(submitButton)

      // 送信中はフィールドが無効化される
      await waitFor(() => {
        expect(usernameInput).toBeDisabled()
        expect(passwordInput).toBeDisabled()
        expect(submitButton).toBeDisabled()
      })

      // Sign inを完了
      resolveLogin!()

      // 有効に戻る
      await waitFor(() => {
        expect(usernameInput).not.toBeDisabled()
        expect(passwordInput).not.toBeDisabled()
      })
    })

    it('should re-enable fields after failed login', async () => {
      const user = userEvent.setup()

      // Promise を手動で制御して送信中の状態を確認できるようにする
      let rejectLogin: (error: Error) => void
      mockLogin.mockReturnValue(
        new Promise((_, reject) => {
          rejectLogin = reject as (error: Error) => void
        })
      )

      renderLogin()

      const usernameInput = screen.getByLabelText('Username')
      const passwordInput = screen.getByLabelText('Password')
      const submitButton = screen.getByRole('button', { name: 'Sign in' })

      await user.type(usernameInput, 'admin')
      await user.type(passwordInput, 'password')
      await user.click(submitButton)

      // 送信中はフィールドが無効化される
      await waitFor(() => {
        expect(usernameInput).toBeDisabled()
        expect(passwordInput).toBeDisabled()
        expect(submitButton).toBeDisabled()
      })

      // Sign in失敗
      rejectLogin!(new Error('Login failed'))

      // エラー後にフィールドが再度有効になる
      await waitFor(() => {
        expect(usernameInput).not.toBeDisabled()
        expect(passwordInput).not.toBeDisabled()
        expect(submitButton).not.toBeDisabled()
      })
    })
  })

  describe('Form validation', () => {
    it('should have required attribute on username field', () => {
      renderLogin()
      expect(screen.getByLabelText('Username')).toBeRequired()
    })

    it('should have required attribute on password field', () => {
      renderLogin()
      expect(screen.getByLabelText('Password')).toBeRequired()
    })

    it('should have password type on password field', () => {
      renderLogin()
      expect(screen.getByLabelText('Password')).toHaveAttribute('type', 'password')
    })
  })
})
