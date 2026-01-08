import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { PostList } from './PostList'
import { apiClient } from '../api/client'
import type { Post, PostsResponse } from '../api/client'

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

  const mockResponse: PostsResponse = {
    posts: mockPosts,
    total: 3,
  }

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
      vi.mocked(apiClient.getPosts).mockResolvedValue(mockResponse)
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
      vi.mocked(apiClient.getPosts).mockResolvedValue(mockResponse)
      renderPostList()

      await waitFor(() => {
        expect(screen.getByText('記事一覧')).toBeInTheDocument()
      })
    })

    it('should render new post button', async () => {
      vi.mocked(apiClient.getPosts).mockResolvedValue(mockResponse)
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
      vi.mocked(apiClient.getPosts).mockResolvedValue(mockResponse)
      renderPostList()

      await waitFor(() => {
        expect(screen.getByText('First Published Post')).toBeInTheDocument()
      })

      expect(screen.getByText('Second Draft Post')).toBeInTheDocument()
      expect(screen.getByText('Third Published Post')).toBeInTheDocument()
    })

    it('should render post count from server total', async () => {
      vi.mocked(apiClient.getPosts).mockResolvedValue(mockResponse)
      renderPostList()

      await waitFor(() => {
        expect(screen.getByText('3件の記事')).toBeInTheDocument()
      })
    })

    it('should render status badges', async () => {
      vi.mocked(apiClient.getPosts).mockResolvedValue(mockResponse)
      renderPostList()

      await waitFor(() => {
        const table = screen.getByRole('table')
        expect(within(table).getAllByText('公開済み')).toHaveLength(2)
        expect(within(table).getByText('下書き')).toBeInTheDocument()
      })
    })

    it('should render tags', async () => {
      vi.mocked(apiClient.getPosts).mockResolvedValue(mockResponse)
      renderPostList()

      await waitFor(() => {
        expect(screen.getByText('Go')).toBeInTheDocument()
      })

      expect(screen.getByText('Web')).toBeInTheDocument()
      expect(screen.getByText('React')).toBeInTheDocument()
    })

    it('should render formatted dates with timezone in title', async () => {
      vi.mocked(apiClient.getPosts).mockResolvedValue(mockResponse)
      renderPostList()

      await waitFor(() => {
        const table = screen.getByRole('table')
        const cells = within(table).getAllByRole('cell')
        // 表示: yyyy-MM-dd 形式
        const dateCells = cells.filter(cell => {
          const text = cell.textContent || ''
          return /^\d{4}-\d{2}-\d{2}$/.test(text)
        })
        // updated_at 3つ + published_at 2つ = 5つの日付セル
        expect(dateCells.length).toBeGreaterThanOrEqual(5)

        // title属性にタイムゾーン略称が含まれていることを確認
        const cellsWithTitle = dateCells.filter(cell => {
          const title = cell.getAttribute('title') || ''
          return /\d{4}-\d{2}-\d{2} \d{2}:\d{2} \([A-Z+\-\d:]+\)/.test(title)
        })
        expect(cellsWithTitle.length).toBeGreaterThanOrEqual(5)
      })
    })

    it('should render "-" for null published_at', async () => {
      vi.mocked(apiClient.getPosts).mockResolvedValue(mockResponse)
      renderPostList()

      await waitFor(() => {
        const table = screen.getByRole('table')
        expect(within(table).getAllByText('-')).toHaveLength(1)
      })
    })

    it('should render post links with correct href', async () => {
      vi.mocked(apiClient.getPosts).mockResolvedValue(mockResponse)
      renderPostList()

      await waitFor(() => {
        const firstPostLink = screen.getByText('First Published Post').closest('a')
        expect(firstPostLink).toHaveAttribute('href', '/posts/1/edit')
      })

      const secondPostLink = screen.getByText('Second Draft Post').closest('a')
      expect(secondPostLink).toHaveAttribute('href', '/posts/2/edit')
    })

    it('should call getPosts with correct parameters on mount', async () => {
      vi.mocked(apiClient.getPosts).mockResolvedValue(mockResponse)
      renderPostList()

      await waitFor(() => {
        expect(apiClient.getPosts).toHaveBeenCalledWith({
          status: undefined,
          limit: 20,
          offset: 0,
        })
      })
    })
  })

  describe('Empty state', () => {
    it('should show empty message when no posts exist', async () => {
      vi.mocked(apiClient.getPosts).mockResolvedValue({ posts: [], total: 0 })
      renderPostList()

      await waitFor(() => {
        expect(screen.getByText('記事が見つかりませんでした')).toBeInTheDocument()
      })

      expect(screen.getByText('0件の記事')).toBeInTheDocument()
    })

    it('should not show table when no posts exist', async () => {
      vi.mocked(apiClient.getPosts).mockResolvedValue({ posts: [], total: 0 })
      renderPostList()

      await waitFor(() => {
        expect(screen.getByText('記事が見つかりませんでした')).toBeInTheDocument()
      })

      expect(screen.queryByRole('table')).not.toBeInTheDocument()
    })
  })

  describe('Status filter', () => {
    it('should render status filter dropdown', async () => {
      vi.mocked(apiClient.getPosts).mockResolvedValue(mockResponse)
      renderPostList()

      await waitFor(() => {
        expect(screen.getByLabelText('ステータス')).toBeInTheDocument()
      })
    })

    it('should have default value "all"', async () => {
      vi.mocked(apiClient.getPosts).mockResolvedValue(mockResponse)
      renderPostList()

      await waitFor(() => {
        expect(screen.getByLabelText('ステータス')).toHaveValue('all')
      })
    })

    it('should call API with status filter when "draft" is selected', async () => {
      const user = userEvent.setup()
      const draftPosts = mockPosts.filter(p => p.status === 'draft')
      vi.mocked(apiClient.getPosts)
        .mockResolvedValueOnce(mockResponse) // 初回ロード
        .mockResolvedValueOnce({ posts: draftPosts, total: 1 }) // フィルタ後

      renderPostList()

      await waitFor(() => {
        expect(screen.getByText('First Published Post')).toBeInTheDocument()
      })

      await user.selectOptions(screen.getByLabelText('ステータス'), 'draft')

      await waitFor(() => {
        expect(apiClient.getPosts).toHaveBeenLastCalledWith({
          status: 'draft',
          limit: 20,
          offset: 0,
        })
      })
    })

    it('should call API with status filter when "published" is selected', async () => {
      const user = userEvent.setup()
      const publishedPosts = mockPosts.filter(p => p.status === 'published')
      vi.mocked(apiClient.getPosts)
        .mockResolvedValueOnce(mockResponse) // 初回ロード
        .mockResolvedValueOnce({ posts: publishedPosts, total: 2 }) // フィルタ後

      renderPostList()

      await waitFor(() => {
        expect(screen.getByText('First Published Post')).toBeInTheDocument()
      })

      await user.selectOptions(screen.getByLabelText('ステータス'), 'published')

      await waitFor(() => {
        expect(apiClient.getPosts).toHaveBeenLastCalledWith({
          status: 'published',
          limit: 20,
          offset: 0,
        })
      })
    })

    it('should call API without status filter when "all" is selected after filtering', async () => {
      const user = userEvent.setup()
      const draftPosts = mockPosts.filter(p => p.status === 'draft')
      vi.mocked(apiClient.getPosts)
        .mockResolvedValueOnce(mockResponse) // 初回ロード
        .mockResolvedValueOnce({ posts: draftPosts, total: 1 }) // draft フィルタ
        .mockResolvedValueOnce(mockResponse) // all に戻す

      renderPostList()

      await waitFor(() => {
        expect(screen.getByText('First Published Post')).toBeInTheDocument()
      })

      await user.selectOptions(screen.getByLabelText('ステータス'), 'draft')
      await waitFor(() => {
        expect(screen.getByText('1件の記事')).toBeInTheDocument()
      })

      await user.selectOptions(screen.getByLabelText('ステータス'), 'all')

      await waitFor(() => {
        expect(apiClient.getPosts).toHaveBeenLastCalledWith({
          status: undefined,
          limit: 20,
          offset: 0,
        })
      })
    })
  })

  describe('Pagination', () => {
    const createManyPostsResponse = (count: number, page: number, limit: number = 20): PostsResponse => {
      const offset = (page - 1) * limit
      const posts = Array.from({ length: Math.min(limit, count - offset) }, (_, i) => ({
        id: offset + i + 1,
        title: `Post ${offset + i + 1}`,
        slug: `post-${offset + i + 1}`,
        content: `Content ${offset + i + 1}`,
        status: 'published' as const,
        tags: '',
        created_at: '2024-01-01T00:00:00Z',
        updated_at: '2024-01-01T00:00:00Z',
        published_at: '2024-01-01T00:00:00Z',
      }))
      return { posts, total: count }
    }

    it('should not show pagination when total is 20 or less', async () => {
      vi.mocked(apiClient.getPosts).mockResolvedValue(createManyPostsResponse(20, 1))
      renderPostList()

      await waitFor(() => {
        expect(screen.getByText('20件の記事')).toBeInTheDocument()
      })

      expect(screen.queryByRole('button', { name: '前へ' })).not.toBeInTheDocument()
      expect(screen.queryByRole('button', { name: '次へ' })).not.toBeInTheDocument()
    })

    it('should show pagination when total is more than 20', async () => {
      vi.mocked(apiClient.getPosts).mockResolvedValue(createManyPostsResponse(25, 1))
      renderPostList()

      await waitFor(() => {
        expect(screen.getByRole('button', { name: '前へ' })).toBeInTheDocument()
      })

      expect(screen.getByRole('button', { name: '次へ' })).toBeInTheDocument()
      expect(screen.getByText('1 / 2')).toBeInTheDocument()
    })

    it('should show posts from server response', async () => {
      vi.mocked(apiClient.getPosts).mockResolvedValue(createManyPostsResponse(25, 1))
      renderPostList()

      await waitFor(() => {
        expect(screen.getByText('Post 1')).toBeInTheDocument()
      })

      expect(screen.getByText('Post 20')).toBeInTheDocument()
      expect(screen.queryByText('Post 21')).not.toBeInTheDocument()
    })

    it('should disable "前へ" button on first page', async () => {
      vi.mocked(apiClient.getPosts).mockResolvedValue(createManyPostsResponse(25, 1))
      renderPostList()

      await waitFor(() => {
        expect(screen.getByRole('button', { name: '前へ' })).toBeDisabled()
      })
    })

    it('should enable "次へ" button on first page when there are more pages', async () => {
      vi.mocked(apiClient.getPosts).mockResolvedValue(createManyPostsResponse(25, 1))
      renderPostList()

      await waitFor(() => {
        expect(screen.getByRole('button', { name: '次へ' })).not.toBeDisabled()
      })
    })

    it('should call API with offset when "次へ" is clicked', async () => {
      const user = userEvent.setup()
      vi.mocked(apiClient.getPosts)
        .mockResolvedValueOnce(createManyPostsResponse(25, 1)) // 初回ロード (page 1)
        .mockResolvedValueOnce(createManyPostsResponse(25, 2)) // 次ページ (page 2)

      renderPostList()

      await waitFor(() => {
        expect(screen.getByRole('button', { name: '次へ' })).toBeInTheDocument()
      })

      await user.click(screen.getByRole('button', { name: '次へ' }))

      await waitFor(() => {
        expect(apiClient.getPosts).toHaveBeenLastCalledWith({
          status: undefined,
          limit: 20,
          offset: 20,
        })
      })

      expect(screen.getByText('2 / 2')).toBeInTheDocument()
    })

    it('should call API with previous offset when "前へ" is clicked', async () => {
      const user = userEvent.setup()
      vi.mocked(apiClient.getPosts)
        .mockResolvedValueOnce(createManyPostsResponse(25, 1)) // 初回ロード (page 1)
        .mockResolvedValueOnce(createManyPostsResponse(25, 2)) // 次ページ (page 2)
        .mockResolvedValueOnce(createManyPostsResponse(25, 1)) // 前ページ (page 1)

      renderPostList()

      await waitFor(() => {
        expect(screen.getByRole('button', { name: '次へ' })).toBeInTheDocument()
      })

      await user.click(screen.getByRole('button', { name: '次へ' }))
      await waitFor(() => {
        expect(screen.getByText('2 / 2')).toBeInTheDocument()
      })

      await user.click(screen.getByRole('button', { name: '前へ' }))

      await waitFor(() => {
        expect(apiClient.getPosts).toHaveBeenLastCalledWith({
          status: undefined,
          limit: 20,
          offset: 0,
        })
      })

      expect(screen.getByText('1 / 2')).toBeInTheDocument()
    })

    it('should disable "次へ" button on last page', async () => {
      const user = userEvent.setup()
      vi.mocked(apiClient.getPosts)
        .mockResolvedValueOnce(createManyPostsResponse(25, 1)) // 初回ロード (page 1)
        .mockResolvedValueOnce(createManyPostsResponse(25, 2)) // 次ページ (page 2)

      renderPostList()

      await waitFor(() => {
        expect(screen.getByRole('button', { name: '次へ' })).toBeInTheDocument()
      })

      await user.click(screen.getByRole('button', { name: '次へ' }))

      await waitFor(() => {
        expect(screen.getByRole('button', { name: '次へ' })).toBeDisabled()
        expect(screen.getByRole('button', { name: '前へ' })).not.toBeDisabled()
      })
    })

    it('should reset to page 1 when status filter changes', async () => {
      const user = userEvent.setup()
      vi.mocked(apiClient.getPosts)
        .mockResolvedValueOnce(createManyPostsResponse(25, 1)) // 初回ロード
        .mockResolvedValueOnce(createManyPostsResponse(25, 2)) // page 2
        .mockResolvedValueOnce({ posts: [], total: 0 }) // filter変更後

      renderPostList()

      await waitFor(() => {
        expect(screen.getByRole('button', { name: '次へ' })).toBeInTheDocument()
      })

      await user.click(screen.getByRole('button', { name: '次へ' }))
      await waitFor(() => {
        expect(screen.getByText('2 / 2')).toBeInTheDocument()
      })

      await user.selectOptions(screen.getByLabelText('ステータス'), 'draft')

      // ステータス変更時にoffset: 0でAPIが呼ばれることを確認
      await waitFor(() => {
        expect(apiClient.getPosts).toHaveBeenLastCalledWith({
          status: 'draft',
          limit: 20,
          offset: 0,
        })
      })
    })

    it('should calculate correct total pages', async () => {
      vi.mocked(apiClient.getPosts).mockResolvedValue(createManyPostsResponse(60, 1))
      renderPostList()

      await waitFor(() => {
        expect(screen.getByText('1 / 3')).toBeInTheDocument()
      })
    })
  })

  describe('Date formatting', () => {
    it('should format dates with timezone abbreviation in title', async () => {
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

      vi.mocked(apiClient.getPosts).mockResolvedValue({ posts, total: 1 })
      renderPostList()

      await waitFor(() => {
        const table = screen.getByRole('table')
        const cells = within(table).getAllByRole('cell')
        // 表示: yyyy-MM-dd 形式
        const hasDate = cells.some(cell => {
          const text = cell.textContent || ''
          return /^\d{4}-\d{2}-\d{2}$/.test(text)
        })
        expect(hasDate).toBe(true)

        // title属性にタイムゾーン略称が含まれていることを確認
        const hasDateWithTimezoneInTitle = cells.some(cell => {
          const title = cell.getAttribute('title') || ''
          return /\d{4}-\d{2}-\d{2} \d{2}:\d{2} \([A-Z+\-\d:]+\)/.test(title)
        })
        expect(hasDateWithTimezoneInTitle).toBe(true)
      })
    })
  })
})
