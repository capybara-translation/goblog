import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { PostList } from './PostList'
import { apiClient } from '../api/client'
import type { Post } from '../api/client'

// apiClient をモック
vi.mock('../api/client', () => ({
  apiClient: {
    getPosts: vi.fn(),
  },
}))

describe('PostList', () => {
  const mockPosts: Post[] = [
    {
      id: 1,
      title: 'First Published Post',
      slug: 'first-published-post',
      content: 'Content 1',
      status: 'published',
      tags: 'Go, Web',
      created_at: '2024-01-01T00:00:00Z',
      updated_at: '2024-01-05T00:00:00Z',
      published_at: '2024-01-02T00:00:00Z',
    },
    {
      id: 2,
      title: 'Second Draft Post',
      slug: 'second-draft-post',
      content: 'Content 2',
      status: 'draft',
      tags: 'React',
      created_at: '2024-01-03T00:00:00Z',
      updated_at: '2024-01-06T00:00:00Z',
      published_at: null,
    },
    {
      id: 3,
      title: 'Third Published Post',
      slug: 'third-published-post',
      content: 'Content 3',
      status: 'published',
      tags: '',
      created_at: '2024-01-04T00:00:00Z',
      updated_at: '2024-01-07T00:00:00Z',
      published_at: '2024-01-05T00:00:00Z',
    },
  ]

  beforeEach(() => {
    vi.clearAllMocks()
  })

  const renderPostList = () => {
    return render(
      <MemoryRouter>
        <PostList />
      </MemoryRouter>
    )
  }

  describe('Loading state', () => {
    it('should show loading message initially', () => {
      vi.mocked(apiClient.getPosts).mockReturnValue(new Promise(() => {}))
      renderPostList()
      expect(screen.getByText('読み込み中...')).toBeInTheDocument()
    })

    it('should hide loading message after posts are loaded', async () => {
      vi.mocked(apiClient.getPosts).mockResolvedValue(mockPosts)
      renderPostList()

      await waitFor(() => {
        expect(screen.queryByText('読み込み中...')).not.toBeInTheDocument()
      })
    })
  })

  describe('Error state', () => {
    it('should show error message when posts fail to load', async () => {
      vi.mocked(apiClient.getPosts).mockRejectedValue(
        new Error('ネットワークエラー')
      )
      renderPostList()

      await waitFor(() => {
        expect(screen.getByText('ネットワークエラー')).toBeInTheDocument()
      })
    })

    it('should show default error message for non-Error exceptions', async () => {
      vi.mocked(apiClient.getPosts).mockRejectedValue('Unknown error')
      renderPostList()

      await waitFor(() => {
        expect(screen.getByText('記事の取得に失敗しました')).toBeInTheDocument()
      })
    })

    it('should not show posts list when error occurs', async () => {
      vi.mocked(apiClient.getPosts).mockRejectedValue(new Error('Error'))
      renderPostList()

      await waitFor(() => {
        expect(screen.getByText('Error')).toBeInTheDocument()
      })

      expect(screen.queryByText('記事一覧')).not.toBeInTheDocument()
    })
  })

  describe('Posts list rendering', () => {
    it('should render page title', async () => {
      vi.mocked(apiClient.getPosts).mockResolvedValue(mockPosts)
      renderPostList()

      await waitFor(() => {
        expect(screen.getByText('記事一覧')).toBeInTheDocument()
      })
    })

    it('should render new post button', async () => {
      vi.mocked(apiClient.getPosts).mockResolvedValue(mockPosts)
      renderPostList()

      await waitFor(() => {
        expect(screen.getByText('新規作成')).toBeInTheDocument()
      })

      expect(screen.getByText('新規作成').closest('a')).toHaveAttribute(
        'href',
        '/posts/new'
      )
    })

    it('should render all posts', async () => {
      vi.mocked(apiClient.getPosts).mockResolvedValue(mockPosts)
      renderPostList()

      await waitFor(() => {
        expect(screen.getByText('First Published Post')).toBeInTheDocument()
      })

      expect(screen.getByText('Second Draft Post')).toBeInTheDocument()
      expect(screen.getByText('Third Published Post')).toBeInTheDocument()
    })

    it('should render post count', async () => {
      vi.mocked(apiClient.getPosts).mockResolvedValue(mockPosts)
      renderPostList()

      await waitFor(() => {
        expect(screen.getByText('3件の記事')).toBeInTheDocument()
      })
    })

    it('should render status badges', async () => {
      vi.mocked(apiClient.getPosts).mockResolvedValue(mockPosts)
      renderPostList()

      await waitFor(() => {
        const table = screen.getByRole('table')
        expect(within(table).getAllByText('公開済み')).toHaveLength(2)
        expect(within(table).getByText('下書き')).toBeInTheDocument()
      })
    })

    it('should render tags', async () => {
      vi.mocked(apiClient.getPosts).mockResolvedValue(mockPosts)
      renderPostList()

      await waitFor(() => {
        expect(screen.getByText('Go')).toBeInTheDocument()
      })

      expect(screen.getByText('Web')).toBeInTheDocument()
      expect(screen.getByText('React')).toBeInTheDocument()
    })

    it('should render formatted dates with timezone', async () => {
      vi.mocked(apiClient.getPosts).mockResolvedValue(mockPosts)
      renderPostList()

      await waitFor(() => {
        const table = screen.getByRole('table')
        // 日付 + タイムゾーン略称が表示されていることを確認
        const cells = within(table).getAllByRole('cell')
        // パターン: "日付 (タイムゾーン)"
        // 例: "2024/12/26 (JST)", "01/05/2024 (GMT+9)", "12/26/2024 (UTC+9)"
        const dateCells = cells.filter(cell => {
          const text = cell.textContent || ''
          // 数字とセパレータ + 括弧内のタイムゾーン（略称またはGMT/UTC形式）
          return /\d+[\/\.\-]\d+[\/\.\-]\d+\s*\([A-Z+\-\d:]+\)/.test(text)
        })
        // updated_at 3つ + published_at 2つ = 5つの日付セル
        expect(dateCells.length).toBeGreaterThanOrEqual(5)
      })
    })

    it('should render "-" for null published_at', async () => {
      vi.mocked(apiClient.getPosts).mockResolvedValue(mockPosts)
      renderPostList()

      await waitFor(() => {
        const table = screen.getByRole('table')
        expect(within(table).getAllByText('-')).toHaveLength(1)
      })
    })

    it('should render post links with correct href', async () => {
      vi.mocked(apiClient.getPosts).mockResolvedValue(mockPosts)
      renderPostList()

      await waitFor(() => {
        const firstPostLink = screen.getByText('First Published Post').closest('a')
        expect(firstPostLink).toHaveAttribute('href', '/posts/1/edit')
      })

      const secondPostLink = screen.getByText('Second Draft Post').closest('a')
      expect(secondPostLink).toHaveAttribute('href', '/posts/2/edit')
    })

    it('should call getPosts with undefined status on mount', async () => {
      vi.mocked(apiClient.getPosts).mockResolvedValue(mockPosts)
      renderPostList()

      await waitFor(() => {
        expect(apiClient.getPosts).toHaveBeenCalledWith({ status: undefined })
      })
    })
  })

  describe('Empty state', () => {
    it('should show empty message when no posts exist', async () => {
      vi.mocked(apiClient.getPosts).mockResolvedValue([])
      renderPostList()

      await waitFor(() => {
        expect(screen.getByText('記事が見つかりませんでした')).toBeInTheDocument()
      })

      expect(screen.getByText('0件の記事')).toBeInTheDocument()
    })

    it('should not show table when no posts exist', async () => {
      vi.mocked(apiClient.getPosts).mockResolvedValue([])
      renderPostList()

      await waitFor(() => {
        expect(screen.getByText('記事が見つかりませんでした')).toBeInTheDocument()
      })

      expect(screen.queryByRole('table')).not.toBeInTheDocument()
    })
  })

  describe('Status filter', () => {
    it('should render status filter dropdown', async () => {
      vi.mocked(apiClient.getPosts).mockResolvedValue(mockPosts)
      renderPostList()

      await waitFor(() => {
        expect(screen.getByLabelText('ステータス')).toBeInTheDocument()
      })
    })

    it('should have default value "all"', async () => {
      vi.mocked(apiClient.getPosts).mockResolvedValue(mockPosts)
      renderPostList()

      await waitFor(() => {
        expect(screen.getByLabelText('ステータス')).toHaveValue('all')
      })
    })

    it('should filter draft posts when "draft" is selected', async () => {
      const user = userEvent.setup()
      vi.mocked(apiClient.getPosts).mockResolvedValue(mockPosts)
      renderPostList()

      await waitFor(() => {
        expect(screen.getByText('First Published Post')).toBeInTheDocument()
      })

      await user.selectOptions(screen.getByLabelText('ステータス'), 'draft')

      expect(screen.getByText('Second Draft Post')).toBeInTheDocument()
      expect(screen.queryByText('First Published Post')).not.toBeInTheDocument()
      expect(screen.queryByText('Third Published Post')).not.toBeInTheDocument()
      expect(screen.getByText('1件の記事')).toBeInTheDocument()
    })

    it('should filter published posts when "published" is selected', async () => {
      const user = userEvent.setup()
      vi.mocked(apiClient.getPosts).mockResolvedValue(mockPosts)
      renderPostList()

      await waitFor(() => {
        expect(screen.getByText('First Published Post')).toBeInTheDocument()
      })

      await user.selectOptions(screen.getByLabelText('ステータス'), 'published')

      expect(screen.getByText('First Published Post')).toBeInTheDocument()
      expect(screen.getByText('Third Published Post')).toBeInTheDocument()
      expect(screen.queryByText('Second Draft Post')).not.toBeInTheDocument()
      expect(screen.getByText('2件の記事')).toBeInTheDocument()
    })

    it('should show all posts when "all" is selected after filtering', async () => {
      const user = userEvent.setup()
      vi.mocked(apiClient.getPosts).mockResolvedValue(mockPosts)
      renderPostList()

      await waitFor(() => {
        expect(screen.getByText('First Published Post')).toBeInTheDocument()
      })

      await user.selectOptions(screen.getByLabelText('ステータス'), 'draft')
      expect(screen.getByText('1件の記事')).toBeInTheDocument()

      await user.selectOptions(screen.getByLabelText('ステータス'), 'all')
      expect(screen.getByText('3件の記事')).toBeInTheDocument()
    })
  })

  describe('Title search', () => {
    it('should render search input', async () => {
      vi.mocked(apiClient.getPosts).mockResolvedValue(mockPosts)
      renderPostList()

      await waitFor(() => {
        expect(screen.getByLabelText('タイトル検索')).toBeInTheDocument()
      })
    })

    it('should filter posts by title search', async () => {
      const user = userEvent.setup()
      vi.mocked(apiClient.getPosts).mockResolvedValue(mockPosts)
      renderPostList()

      await waitFor(() => {
        expect(screen.getByText('First Published Post')).toBeInTheDocument()
      })

      await user.type(screen.getByLabelText('タイトル検索'), 'Draft')

      expect(screen.getByText('Second Draft Post')).toBeInTheDocument()
      expect(screen.queryByText('First Published Post')).not.toBeInTheDocument()
      expect(screen.getByText('1件の記事')).toBeInTheDocument()
    })

    it('should be case insensitive', async () => {
      const user = userEvent.setup()
      vi.mocked(apiClient.getPosts).mockResolvedValue(mockPosts)
      renderPostList()

      await waitFor(() => {
        expect(screen.getByText('First Published Post')).toBeInTheDocument()
      })

      await user.type(screen.getByLabelText('タイトル検索'), 'first')

      expect(screen.getByText('First Published Post')).toBeInTheDocument()
      expect(screen.getByText('1件の記事')).toBeInTheDocument()
    })

    it('should show empty state when no matches found', async () => {
      const user = userEvent.setup()
      vi.mocked(apiClient.getPosts).mockResolvedValue(mockPosts)
      renderPostList()

      await waitFor(() => {
        expect(screen.getByText('First Published Post')).toBeInTheDocument()
      })

      await user.type(screen.getByLabelText('タイトル検索'), 'nonexistent')

      expect(screen.getByText('記事が見つかりませんでした')).toBeInTheDocument()
      expect(screen.getByText('0件の記事')).toBeInTheDocument()
    })

    it('should ignore whitespace in search query', async () => {
      const user = userEvent.setup()
      vi.mocked(apiClient.getPosts).mockResolvedValue(mockPosts)
      renderPostList()

      await waitFor(() => {
        expect(screen.getByText('First Published Post')).toBeInTheDocument()
      })

      await user.type(screen.getByLabelText('タイトル検索'), '   ')

      // 空白のみの検索は無視され、全記事表示
      expect(screen.getByText('3件の記事')).toBeInTheDocument()
    })
  })

  describe('Combined filters', () => {
    it('should apply both status filter and search together', async () => {
      const user = userEvent.setup()
      vi.mocked(apiClient.getPosts).mockResolvedValue(mockPosts)
      renderPostList()

      await waitFor(() => {
        expect(screen.getByText('First Published Post')).toBeInTheDocument()
      })

      await user.selectOptions(screen.getByLabelText('ステータス'), 'published')
      await user.type(screen.getByLabelText('タイトル検索'), 'Third')

      expect(screen.getByText('Third Published Post')).toBeInTheDocument()
      expect(screen.queryByText('First Published Post')).not.toBeInTheDocument()
      expect(screen.queryByText('Second Draft Post')).not.toBeInTheDocument()
      expect(screen.getByText('1件の記事')).toBeInTheDocument()
    })
  })

  describe('Pagination', () => {
    const createManyPosts = (count: number): Post[] => {
      return Array.from({ length: count }, (_, i) => ({
        id: i + 1,
        title: `Post ${i + 1}`,
        slug: `post-${i + 1}`,
        content: `Content ${i + 1}`,
        status: 'published' as const,
        tags: '',
        created_at: '2024-01-01T00:00:00Z',
        updated_at: '2024-01-01T00:00:00Z',
        published_at: '2024-01-01T00:00:00Z',
      }))
    }

    it('should not show pagination when posts are 20 or less', async () => {
      vi.mocked(apiClient.getPosts).mockResolvedValue(createManyPosts(20))
      renderPostList()

      await waitFor(() => {
        expect(screen.getByText('20件の記事')).toBeInTheDocument()
      })

      expect(screen.queryByRole('button', { name: '前へ' })).not.toBeInTheDocument()
      expect(screen.queryByRole('button', { name: '次へ' })).not.toBeInTheDocument()
    })

    it('should show pagination when posts are more than 20', async () => {
      vi.mocked(apiClient.getPosts).mockResolvedValue(createManyPosts(25))
      renderPostList()

      await waitFor(() => {
        expect(screen.getByRole('button', { name: '前へ' })).toBeInTheDocument()
      })

      expect(screen.getByRole('button', { name: '次へ' })).toBeInTheDocument()
      expect(screen.getByText('1 / 2')).toBeInTheDocument()
    })

    it('should show only first 20 posts on page 1', async () => {
      vi.mocked(apiClient.getPosts).mockResolvedValue(createManyPosts(25))
      renderPostList()

      await waitFor(() => {
        expect(screen.getByText('Post 1')).toBeInTheDocument()
      })

      expect(screen.getByText('Post 20')).toBeInTheDocument()
      expect(screen.queryByText('Post 21')).not.toBeInTheDocument()
    })

    it('should disable "前へ" button on first page', async () => {
      vi.mocked(apiClient.getPosts).mockResolvedValue(createManyPosts(25))
      renderPostList()

      await waitFor(() => {
        expect(screen.getByRole('button', { name: '前へ' })).toBeDisabled()
      })
    })

    it('should enable "次へ" button on first page when there are more pages', async () => {
      vi.mocked(apiClient.getPosts).mockResolvedValue(createManyPosts(25))
      renderPostList()

      await waitFor(() => {
        expect(screen.getByRole('button', { name: '次へ' })).not.toBeDisabled()
      })
    })

    it('should navigate to next page when "次へ" is clicked', async () => {
      const user = userEvent.setup()
      vi.mocked(apiClient.getPosts).mockResolvedValue(createManyPosts(25))
      renderPostList()

      await waitFor(() => {
        expect(screen.getByRole('button', { name: '次へ' })).toBeInTheDocument()
      })

      await user.click(screen.getByRole('button', { name: '次へ' }))

      expect(screen.getByText('2 / 2')).toBeInTheDocument()
      expect(screen.getByText('Post 21')).toBeInTheDocument()
      expect(screen.getByText('Post 25')).toBeInTheDocument()
      expect(screen.queryByText('Post 20')).not.toBeInTheDocument()
    })

    it('should navigate to previous page when "前へ" is clicked', async () => {
      const user = userEvent.setup()
      vi.mocked(apiClient.getPosts).mockResolvedValue(createManyPosts(25))
      renderPostList()

      await waitFor(() => {
        expect(screen.getByRole('button', { name: '次へ' })).toBeInTheDocument()
      })

      await user.click(screen.getByRole('button', { name: '次へ' }))
      expect(screen.getByText('2 / 2')).toBeInTheDocument()

      await user.click(screen.getByRole('button', { name: '前へ' }))
      expect(screen.getByText('1 / 2')).toBeInTheDocument()
      expect(screen.getByText('Post 1')).toBeInTheDocument()
    })

    it('should disable "次へ" button on last page', async () => {
      const user = userEvent.setup()
      vi.mocked(apiClient.getPosts).mockResolvedValue(createManyPosts(25))
      renderPostList()

      await waitFor(() => {
        expect(screen.getByRole('button', { name: '次へ' })).toBeInTheDocument()
      })

      await user.click(screen.getByRole('button', { name: '次へ' }))

      expect(screen.getByRole('button', { name: '次へ' })).toBeDisabled()
      expect(screen.getByRole('button', { name: '前へ' })).not.toBeDisabled()
    })

    it('should reset to page 1 when status filter changes', async () => {
      const user = userEvent.setup()
      vi.mocked(apiClient.getPosts).mockResolvedValue(createManyPosts(25))
      renderPostList()

      await waitFor(() => {
        expect(screen.getByRole('button', { name: '次へ' })).toBeInTheDocument()
      })

      await user.click(screen.getByRole('button', { name: '次へ' }))
      expect(screen.getByText('2 / 2')).toBeInTheDocument()

      await user.selectOptions(screen.getByLabelText('ステータス'), 'draft')

      expect(screen.queryByText('2 / 2')).not.toBeInTheDocument()
    })

    it('should reset to page 1 when search query changes', async () => {
      const user = userEvent.setup()
      vi.mocked(apiClient.getPosts).mockResolvedValue(createManyPosts(25))
      renderPostList()

      await waitFor(() => {
        expect(screen.getByRole('button', { name: '次へ' })).toBeInTheDocument()
      })

      await user.click(screen.getByRole('button', { name: '次へ' }))
      expect(screen.getByText('2 / 2')).toBeInTheDocument()

      await user.type(screen.getByLabelText('タイトル検索'), 'Post')

      // ページ番号が消える（全てフィルタされて1ページに収まる）
      // または1ページ目に戻る
      expect(screen.queryByText('2 / 2')).not.toBeInTheDocument()
    })

    it('should calculate correct total pages', async () => {
      vi.mocked(apiClient.getPosts).mockResolvedValue(createManyPosts(60))
      renderPostList()

      await waitFor(() => {
        expect(screen.getByText('1 / 3')).toBeInTheDocument()
      })
    })
  })

  describe('Date formatting', () => {
    it('should format dates with timezone abbreviation', async () => {
      const posts: Post[] = [
        {
          id: 1,
          title: 'Test',
          slug: 'test',
          content: 'test',
          status: 'published',
          tags: '',
          created_at: '2024-12-25T15:30:00Z',
          updated_at: '2024-12-25T15:30:00Z',
          published_at: '2024-12-25T15:30:00Z',
        },
      ]

      vi.mocked(apiClient.getPosts).mockResolvedValue(posts)
      renderPostList()

      await waitFor(() => {
        const table = screen.getByRole('table')
        const cells = within(table).getAllByRole('cell')
        // 日付 + タイムゾーン略称が表示されていることを確認
        const hasDateWithTimezone = cells.some(cell => {
          const text = cell.textContent || ''
          // パターン: "日付 (TZ)"
          // 例: "2024/12/26 (JST)", "01/05/2024 (GMT+9)", "12/26/2024 (UTC+9)"
          return /\d+[\/\.\-]\d+[\/\.\-]\d+\s*\([A-Z+\-\d:]+\)/.test(text)
        })
        expect(hasDateWithTimezone).toBe(true)
      })
    })
  })
})
