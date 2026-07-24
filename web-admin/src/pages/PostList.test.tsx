import { describe, it, expect, vi, beforeEach } from 'vitest'
import { act, fireEvent, render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter, useLocation, useNavigationType } from 'react-router-dom'
import { PostList } from './PostList'
import { apiClient } from '../api/client'
import type { Post, PostsResponse } from '../api/client'

// Helper that exposes the current router location and navigation type so
// URL-sync tests can assert both the resulting URL and whether the last
// transition pushed or replaced.
function LocationProbe() {
  const location = useLocation()
  const navType = useNavigationType()
  return (
    <>
      <div data-testid="location">
        {location.pathname}
        {location.search}
      </div>
      <div data-testid="nav-type">{navType}</div>
    </>
  )
}

// Mock apiClient
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
      is_pinned: false,
      created_at: '2024-01-01T00:00:00Z',
      updated_at: '2024-01-05T00:00:00Z',
      published_at: '2024-01-02T00:00:00Z',
      view_count: 0,
      health_date: null,
    },
    {
      id: 2,
      title: 'Second Draft Post',
      slug: 'second-draft-post',
      content: 'Content 2',
      status: 'draft',
      tags: 'React',
      is_pinned: false,
      created_at: '2024-01-03T00:00:00Z',
      updated_at: '2024-01-06T00:00:00Z',
      published_at: null,
      view_count: 0,
      health_date: null,
    },
    {
      id: 3,
      title: 'Third Published Post',
      slug: 'third-published-post',
      content: 'Content 3',
      status: 'published',
      tags: '',
      is_pinned: false,
      created_at: '2024-01-04T00:00:00Z',
      updated_at: '2024-01-07T00:00:00Z',
      published_at: '2024-01-05T00:00:00Z',
      view_count: 0,
      health_date: null,
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
      expect(screen.getByText('Loading...')).toBeInTheDocument()
    })

    it('should hide loading message after posts are loaded', async () => {
      vi.mocked(apiClient.getPosts).mockResolvedValue(mockResponse)
      renderPostList()

      await waitFor(() => {
        expect(screen.queryByText('Loading...')).not.toBeInTheDocument()
      })
    })
  })

  describe('Error state', () => {
    it('should show error message when posts fail to load', async () => {
      vi.mocked(apiClient.getPosts).mockRejectedValue(
        new Error('Network error')
      )
      renderPostList()

      await waitFor(() => {
        expect(screen.getByText('Network error')).toBeInTheDocument()
      })
    })

    it('should show default error message for non-Error exceptions', async () => {
      vi.mocked(apiClient.getPosts).mockRejectedValue('Unknown error')
      renderPostList()

      await waitFor(() => {
        expect(screen.getByText('Failed to load posts')).toBeInTheDocument()
      })
    })

    it('should not show posts table when error occurs', async () => {
      vi.mocked(apiClient.getPosts).mockRejectedValue(new Error('Error'))
      renderPostList()

      await waitFor(() => {
        expect(screen.getByText('Error')).toBeInTheDocument()
      })

      expect(screen.queryByRole('table')).not.toBeInTheDocument()
    })
  })

  describe('Posts list rendering', () => {
    it('should render page title', async () => {
      vi.mocked(apiClient.getPosts).mockResolvedValue(mockResponse)
      renderPostList()

      await waitFor(() => {
        expect(screen.getByText('Posts')).toBeInTheDocument()
      })
    })

    it('should render new post button', async () => {
      vi.mocked(apiClient.getPosts).mockResolvedValue(mockResponse)
      renderPostList()

      await waitFor(() => {
        expect(screen.getByText('New Post')).toBeInTheDocument()
      })

      expect(screen.getByText('New Post').closest('a')).toHaveAttribute(
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
        expect(screen.getByText('3 posts')).toBeInTheDocument()
      })
    })

    it('should render status badges', async () => {
      vi.mocked(apiClient.getPosts).mockResolvedValue(mockResponse)
      renderPostList()

      await waitFor(() => {
        const table = screen.getByRole('table')
        // 2 status badges + 1 column header "Published" = 3 total
        expect(within(table).getAllByText('Published')).toHaveLength(3)
        expect(within(table).getByText('Draft')).toBeInTheDocument()
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
        // Display: yyyy-MM-dd format
        const dateCells = cells.filter(cell => {
          const text = cell.textContent || ''
          return /^\d{4}-\d{2}-\d{2}$/.test(text)
        })
        // updated_at 3 + published_at 2 = 5 date cells
        expect(dateCells.length).toBeGreaterThanOrEqual(5)

        // Verify timezone abbreviation is included in title attribute
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

  describe('Pin column display', () => {
    it('should render pin emoji for pinned posts', async () => {
      const postsWithPinned: Post[] = [
        {
          id: 1,
          title: 'Pinned Post',
          slug: 'pinned-post',
          content: 'Content',
          status: 'published',
          tags: '',
          is_pinned: true,
          created_at: '2024-01-01T00:00:00Z',
          updated_at: '2024-01-01T00:00:00Z',
          published_at: '2024-01-01T00:00:00Z',
      view_count: 0,
          health_date: null,
        },
        {
          id: 2,
          title: 'Normal Post',
          slug: 'normal-post',
          content: 'Content',
          status: 'published',
          tags: '',
          is_pinned: false,
          created_at: '2024-01-02T00:00:00Z',
          updated_at: '2024-01-02T00:00:00Z',
          published_at: '2024-01-02T00:00:00Z',
      view_count: 0,
          health_date: null,
        },
      ]

      vi.mocked(apiClient.getPosts).mockResolvedValue({ posts: postsWithPinned, total: 2 })
      renderPostList()

      await waitFor(() => {
        const table = screen.getByRole('table')
        // Pinned posts show 📌 emoji
        expect(within(table).getByText('📌')).toBeInTheDocument()
      })
    })

    it('should not render pin emoji for non-pinned posts', async () => {
      const postsWithoutPinned: Post[] = [
        {
          id: 1,
          title: 'Normal Post 1',
          slug: 'normal-post-1',
          content: 'Content',
          status: 'published',
          tags: '',
          is_pinned: false,
          created_at: '2024-01-01T00:00:00Z',
          updated_at: '2024-01-01T00:00:00Z',
          published_at: '2024-01-01T00:00:00Z',
      view_count: 0,
          health_date: null,
        },
        {
          id: 2,
          title: 'Normal Post 2',
          slug: 'normal-post-2',
          content: 'Content',
          status: 'published',
          tags: '',
          is_pinned: false,
          created_at: '2024-01-02T00:00:00Z',
          updated_at: '2024-01-02T00:00:00Z',
          published_at: '2024-01-02T00:00:00Z',
      view_count: 0,
          health_date: null,
        },
      ]

      vi.mocked(apiClient.getPosts).mockResolvedValue({ posts: postsWithoutPinned, total: 2 })
      renderPostList()

      await waitFor(() => {
        expect(screen.getByText('Normal Post 1')).toBeInTheDocument()
      })

      const table = screen.getByRole('table')
      expect(within(table).queryByText('📌')).not.toBeInTheDocument()
    })

    it('should render pin column header', async () => {
      vi.mocked(apiClient.getPosts).mockResolvedValue(mockResponse)
      renderPostList()

      await waitFor(() => {
        const table = screen.getByRole('table')
        expect(within(table).getByText('Pinned')).toBeInTheDocument()
      })
    })
  })

  describe('Empty state', () => {
    it('should show empty message when no posts exist', async () => {
      vi.mocked(apiClient.getPosts).mockResolvedValue({ posts: [], total: 0 })
      renderPostList()

      await waitFor(() => {
        expect(screen.getByText('No posts found')).toBeInTheDocument()
      })

      expect(screen.getByText('0 posts')).toBeInTheDocument()
    })

    it('should not show table when no posts exist', async () => {
      vi.mocked(apiClient.getPosts).mockResolvedValue({ posts: [], total: 0 })
      renderPostList()

      await waitFor(() => {
        expect(screen.getByText('No posts found')).toBeInTheDocument()
      })

      expect(screen.queryByRole('table')).not.toBeInTheDocument()
    })
  })

  describe('Status filter', () => {
    it('should render status filter dropdown', async () => {
      vi.mocked(apiClient.getPosts).mockResolvedValue(mockResponse)
      renderPostList()

      await waitFor(() => {
        expect(screen.getByLabelText('Status')).toBeInTheDocument()
      })
    })

    it('should have default value "all"', async () => {
      vi.mocked(apiClient.getPosts).mockResolvedValue(mockResponse)
      renderPostList()

      await waitFor(() => {
        expect(screen.getByLabelText('Status')).toHaveValue('all')
      })
    })

    it('should call API with status filter when "draft" is selected', async () => {
      const user = userEvent.setup()
      const draftPosts = mockPosts.filter(p => p.status === 'draft')
      vi.mocked(apiClient.getPosts)
        .mockResolvedValueOnce(mockResponse) // Initial load
        .mockResolvedValueOnce({ posts: draftPosts, total: 1 }) // After filter

      renderPostList()

      await waitFor(() => {
        expect(screen.getByText('First Published Post')).toBeInTheDocument()
      })

      await user.selectOptions(screen.getByLabelText('Status'), 'draft')

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
        .mockResolvedValueOnce(mockResponse) // Initial load
        .mockResolvedValueOnce({ posts: publishedPosts, total: 2 }) // After filter

      renderPostList()

      await waitFor(() => {
        expect(screen.getByText('First Published Post')).toBeInTheDocument()
      })

      await user.selectOptions(screen.getByLabelText('Status'), 'published')

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
        .mockResolvedValueOnce(mockResponse) // Initial load
        .mockResolvedValueOnce({ posts: draftPosts, total: 1 }) // Draft filter
        .mockResolvedValueOnce(mockResponse) // Back to all

      renderPostList()

      await waitFor(() => {
        expect(screen.getByText('First Published Post')).toBeInTheDocument()
      })

      await user.selectOptions(screen.getByLabelText('Status'), 'draft')
      await waitFor(() => {
        expect(screen.getByText('1 posts')).toBeInTheDocument()
      })

      await user.selectOptions(screen.getByLabelText('Status'), 'all')

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
        is_pinned: false,
        created_at: '2024-01-01T00:00:00Z',
        updated_at: '2024-01-01T00:00:00Z',
        published_at: '2024-01-01T00:00:00Z',
      view_count: 0,
        health_date: null,
      }))
      return { posts, total: count }
    }

    it('should not show pagination when total is 20 or less', async () => {
      vi.mocked(apiClient.getPosts).mockResolvedValue(createManyPostsResponse(20, 1))
      renderPostList()

      await waitFor(() => {
        expect(screen.getByText('20 posts')).toBeInTheDocument()
      })

      expect(screen.queryByRole('button', { name: 'Prev' })).not.toBeInTheDocument()
      expect(screen.queryByRole('button', { name: 'Next' })).not.toBeInTheDocument()
    })

    it('should show pagination when total is more than 20', async () => {
      vi.mocked(apiClient.getPosts).mockResolvedValue(createManyPostsResponse(25, 1))
      renderPostList()

      await waitFor(() => {
        expect(screen.getByRole('button', { name: 'Prev' })).toBeInTheDocument()
      })

      expect(screen.getByRole('button', { name: 'Next' })).toBeInTheDocument()
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

    it('should disable "Prev" button on first page', async () => {
      vi.mocked(apiClient.getPosts).mockResolvedValue(createManyPostsResponse(25, 1))
      renderPostList()

      await waitFor(() => {
        expect(screen.getByRole('button', { name: 'Prev' })).toBeDisabled()
      })
    })

    it('should enable "Next" button on first page when there are more pages', async () => {
      vi.mocked(apiClient.getPosts).mockResolvedValue(createManyPostsResponse(25, 1))
      renderPostList()

      await waitFor(() => {
        expect(screen.getByRole('button', { name: 'Next' })).not.toBeDisabled()
      })
    })

    it('should call API with offset when "Next" is clicked', async () => {
      const user = userEvent.setup()
      vi.mocked(apiClient.getPosts)
        .mockResolvedValueOnce(createManyPostsResponse(25, 1)) // Initial load (page 1)
        .mockResolvedValueOnce(createManyPostsResponse(25, 2)) // Next page (page 2)

      renderPostList()

      await waitFor(() => {
        expect(screen.getByRole('button', { name: 'Next' })).toBeInTheDocument()
      })

      await user.click(screen.getByRole('button', { name: 'Next' }))

      await waitFor(() => {
        expect(apiClient.getPosts).toHaveBeenLastCalledWith({
          status: undefined,
          limit: 20,
          offset: 20,
        })
      })

      expect(screen.getByText('2 / 2')).toBeInTheDocument()
    })

    it('should call API with previous offset when "Prev" is clicked', async () => {
      const user = userEvent.setup()
      vi.mocked(apiClient.getPosts)
        .mockResolvedValueOnce(createManyPostsResponse(25, 1)) // Initial load (page 1)
        .mockResolvedValueOnce(createManyPostsResponse(25, 2)) // Next page (page 2)
        .mockResolvedValueOnce(createManyPostsResponse(25, 1)) // Previous page (page 1)

      renderPostList()

      await waitFor(() => {
        expect(screen.getByRole('button', { name: 'Next' })).toBeInTheDocument()
      })

      await user.click(screen.getByRole('button', { name: 'Next' }))
      await waitFor(() => {
        expect(screen.getByText('2 / 2')).toBeInTheDocument()
      })

      await user.click(screen.getByRole('button', { name: 'Prev' }))

      await waitFor(() => {
        expect(apiClient.getPosts).toHaveBeenLastCalledWith({
          status: undefined,
          limit: 20,
          offset: 0,
        })
      })

      expect(screen.getByText('1 / 2')).toBeInTheDocument()
    })

    it('should disable "Next" button on last page', async () => {
      const user = userEvent.setup()
      vi.mocked(apiClient.getPosts)
        .mockResolvedValueOnce(createManyPostsResponse(25, 1)) // Initial load (page 1)
        .mockResolvedValueOnce(createManyPostsResponse(25, 2)) // Next page (page 2)

      renderPostList()

      await waitFor(() => {
        expect(screen.getByRole('button', { name: 'Next' })).toBeInTheDocument()
      })

      await user.click(screen.getByRole('button', { name: 'Next' }))

      await waitFor(() => {
        expect(screen.getByRole('button', { name: 'Next' })).toBeDisabled()
        expect(screen.getByRole('button', { name: 'Prev' })).not.toBeDisabled()
      })
    })

    it('should reset to page 1 when status filter changes', async () => {
      const user = userEvent.setup()
      vi.mocked(apiClient.getPosts)
        .mockResolvedValueOnce(createManyPostsResponse(25, 1)) // Initial load
        .mockResolvedValueOnce(createManyPostsResponse(25, 2)) // Page 2
        .mockResolvedValueOnce({ posts: [], total: 0 }) // After filter change

      renderPostList()

      await waitFor(() => {
        expect(screen.getByRole('button', { name: 'Next' })).toBeInTheDocument()
      })

      await user.click(screen.getByRole('button', { name: 'Next' }))
      await waitFor(() => {
        expect(screen.getByText('2 / 2')).toBeInTheDocument()
      })

      await user.selectOptions(screen.getByLabelText('Status'), 'draft')

      // Verify API is called with offset: 0 when status changes
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

  describe('Search functionality', () => {
    it('should render search input field', async () => {
      vi.mocked(apiClient.getPosts).mockResolvedValue(mockResponse)
      renderPostList()

      await waitFor(() => {
        expect(screen.getByLabelText('Search')).toBeInTheDocument()
      })

      expect(screen.getByPlaceholderText('Search by title or content...')).toBeInTheDocument()
    })

    it('should call API with search query after debounce', async () => {
      const user = userEvent.setup()

      const searchResults: PostsResponse = {
        posts: [mockPosts[0]!],
        total: 1,
      }

      vi.mocked(apiClient.getPosts)
        .mockResolvedValueOnce(mockResponse) // Initial load
        .mockResolvedValue(searchResults) // After search

      renderPostList()

      await waitFor(() => {
        expect(screen.getByText('First Published Post')).toBeInTheDocument()
      })

      const searchInput = screen.getByLabelText('Search')
      await user.type(searchInput, 'First')

      // API is called after debounce (wait 300ms + extra)
      await waitFor(() => {
        expect(apiClient.getPosts).toHaveBeenLastCalledWith({
          status: undefined,
          q: 'First',
          limit: 20,
          offset: 0,
        })
      }, { timeout: 2000 })
    })

    it('should show search results count', async () => {
      const user = userEvent.setup()

      const searchResults: PostsResponse = {
        posts: [mockPosts[0]!],
        total: 1,
      }

      vi.mocked(apiClient.getPosts)
        .mockResolvedValueOnce(mockResponse)
        .mockResolvedValue(searchResults)

      renderPostList()

      await waitFor(() => {
        expect(screen.getByText('3 posts')).toBeInTheDocument()
      })

      const searchInput = screen.getByLabelText('Search')
      await user.type(searchInput, 'First')

      await waitFor(() => {
        expect(screen.getByText(/Search results for "First": 1/)).toBeInTheDocument()
      }, { timeout: 2000 })
    })

    it('should show clear button when search query is entered', async () => {
      const user = userEvent.setup()

      vi.mocked(apiClient.getPosts).mockResolvedValue(mockResponse)
      renderPostList()

      await waitFor(() => {
        expect(screen.getByText('First Published Post')).toBeInTheDocument()
      })

      // Clear button is not displayed initially
      expect(screen.queryByTitle('Clear search')).not.toBeInTheDocument()

      const searchInput = screen.getByLabelText('Search')
      await user.type(searchInput, 'test')

      // Clear button is displayed
      expect(screen.getByTitle('Clear search')).toBeInTheDocument()
    })

    it('should clear search when clear button is clicked', async () => {
      const user = userEvent.setup()

      const searchResults: PostsResponse = {
        posts: [mockPosts[0]!],
        total: 1,
      }

      vi.mocked(apiClient.getPosts)
        .mockResolvedValueOnce(mockResponse) // Initial load
        .mockResolvedValueOnce(searchResults) // After search
        .mockResolvedValue(mockResponse) // After clear

      renderPostList()

      await waitFor(() => {
        expect(screen.getByText('First Published Post')).toBeInTheDocument()
      })

      const searchInput = screen.getByLabelText('Search')
      await user.type(searchInput, 'First')

      await waitFor(() => {
        expect(screen.getByText(/Search results for "First"/)).toBeInTheDocument()
      }, { timeout: 2000 })

      await user.click(screen.getByTitle('Clear search'))

      await waitFor(() => {
        expect(apiClient.getPosts).toHaveBeenLastCalledWith({
          status: undefined,
          q: undefined,
          limit: 20,
          offset: 0,
        })
      }, { timeout: 2000 })

      // Verify search input is cleared
      expect(searchInput).toHaveValue('')
    })

    it('should reset to page 1 when search query changes', async () => {
      const user = userEvent.setup()

      const createManyPostsResponse = (count: number): PostsResponse => {
        const posts = Array.from({ length: 20 }, (_, i) => ({
          id: i + 1,
          title: `Post ${i + 1}`,
          slug: `post-${i + 1}`,
          content: `Content ${i + 1}`,
          status: 'published' as const,
          tags: '',
          is_pinned: false,
          created_at: '2024-01-01T00:00:00Z',
          updated_at: '2024-01-01T00:00:00Z',
          published_at: '2024-01-01T00:00:00Z',
      view_count: 0,
          health_date: null,
        }))
        return { posts, total: count }
      }

      vi.mocked(apiClient.getPosts)
        .mockResolvedValueOnce(createManyPostsResponse(25)) // Initial load
        .mockResolvedValueOnce(createManyPostsResponse(25)) // Page 2
        .mockResolvedValue({ posts: [mockPosts[0]!], total: 1 }) // After search

      renderPostList()

      await waitFor(() => {
        expect(screen.getByRole('button', { name: 'Next' })).toBeInTheDocument()
      })

      // Move to page 2
      await user.click(screen.getByRole('button', { name: 'Next' }))
      await waitFor(() => {
        expect(screen.getByText('2 / 2')).toBeInTheDocument()
      })

      // Execute search
      const searchInput = screen.getByLabelText('Search')
      await user.type(searchInput, 'First')

      // Verify API is called with offset: 0 (reset to page 1)
      await waitFor(() => {
        expect(apiClient.getPosts).toHaveBeenLastCalledWith({
          status: undefined,
          q: 'First',
          limit: 20,
          offset: 0,
        })
      }, { timeout: 2000 })
    })

    it('should combine search with status filter', async () => {
      const user = userEvent.setup()

      const searchResults: PostsResponse = {
        posts: [mockPosts[0]!],
        total: 1,
      }

      vi.mocked(apiClient.getPosts)
        .mockResolvedValueOnce(mockResponse) // Initial load
        .mockResolvedValueOnce(searchResults) // After status change
        .mockResolvedValue(searchResults) // After search

      renderPostList()

      await waitFor(() => {
        expect(screen.getByText('First Published Post')).toBeInTheDocument()
      })

      // Change status filter
      await user.selectOptions(screen.getByLabelText('Status'), 'published')
      await waitFor(() => {
        expect(apiClient.getPosts).toHaveBeenCalledWith({
          status: 'published',
          limit: 20,
          offset: 0,
        })
      })

      // Execute search
      const searchInput = screen.getByLabelText('Search')
      await user.type(searchInput, 'First')

      await waitFor(() => {
        expect(apiClient.getPosts).toHaveBeenLastCalledWith({
          status: 'published',
          q: 'First',
          limit: 20,
          offset: 0,
        })
      }, { timeout: 2000 })
    })

    it('should show empty message for no search results', async () => {
      const user = userEvent.setup()

      vi.mocked(apiClient.getPosts)
        .mockResolvedValueOnce(mockResponse) // Initial load
        .mockResolvedValue({ posts: [], total: 0 }) // After search (0 results)

      renderPostList()

      await waitFor(() => {
        expect(screen.getByText('First Published Post')).toBeInTheDocument()
      })

      const searchInput = screen.getByLabelText('Search')
      await user.type(searchInput, 'nonexistent')

      await waitFor(() => {
        expect(screen.getByText(/No posts found matching "nonexistent"/)).toBeInTheDocument()
      }, { timeout: 2000 })
    })

    it('should not include q parameter value on initial load', async () => {
      vi.mocked(apiClient.getPosts).mockResolvedValue(mockResponse)
      renderPostList()

      await waitFor(() => {
        expect(screen.getByText('First Published Post')).toBeInTheDocument()
      })

      // On initial load, q parameter is undefined
      const calls = vi.mocked(apiClient.getPosts).mock.calls
      expect(calls[0]![0]!.q).toBeUndefined()
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
          is_pinned: false,
          created_at: '2024-12-25T15:30:00Z',
          updated_at: '2024-12-25T15:30:00Z',
          published_at: '2024-12-25T15:30:00Z',
      view_count: 0,
          health_date: null,
        },
      ]

      vi.mocked(apiClient.getPosts).mockResolvedValue({ posts, total: 1 })
      renderPostList()

      await waitFor(() => {
        const table = screen.getByRole('table')
        const cells = within(table).getAllByRole('cell')
        // Display: yyyy-MM-dd format
        const hasDate = cells.some(cell => {
          const text = cell.textContent || ''
          return /^\d{4}-\d{2}-\d{2}$/.test(text)
        })
        expect(hasDate).toBe(true)

        // Verify timezone abbreviation is included in title attribute
        const hasDateWithTimezoneInTitle = cells.some(cell => {
          const title = cell.getAttribute('title') || ''
          return /\d{4}-\d{2}-\d{2} \d{2}:\d{2} \([A-Z+\-\d:]+\)/.test(title)
        })
        expect(hasDateWithTimezoneInTitle).toBe(true)
      })
    })
  })

  describe('URL synchronization', () => {
    const renderWithLocation = (initialEntries: string[] = ['/']) =>
      render(
        <MemoryRouter initialEntries={initialEntries}>
          <PostList />
          <LocationProbe />
        </MemoryRouter>,
      )

    it('initializes state from URL search params', async () => {
      vi.mocked(apiClient.getPosts).mockResolvedValue({
        posts: mockPosts,
        total: 100,
      })
      renderWithLocation(['/?q=foo&status=draft&page=3'])

      await waitFor(() => {
        expect(apiClient.getPosts).toHaveBeenCalledWith({
          status: 'draft',
          q: 'foo',
          limit: 20,
          offset: 40,
        })
      })

      const searchInput = screen.getByLabelText('Search') as HTMLInputElement
      expect(searchInput.value).toBe('foo')

      const statusSelect = screen.getByLabelText('Status') as HTMLSelectElement
      expect(statusSelect.value).toBe('draft')

      expect(screen.getByText('3 / 5')).toBeInTheDocument()
    })

    it('ignores invalid page/status query params and falls back to defaults', async () => {
      vi.mocked(apiClient.getPosts).mockResolvedValue(mockResponse)
      renderWithLocation(['/?page=abc&status=bogus'])

      await waitFor(() => {
        expect(apiClient.getPosts).toHaveBeenCalledWith({
          status: undefined,
          q: undefined,
          limit: 20,
          offset: 0,
        })
      })

      const statusSelect = screen.getByLabelText('Status') as HTMLSelectElement
      expect(statusSelect.value).toBe('all')
    })

    it('updates URL when status filter changes (no defaults persisted)', async () => {
      const user = userEvent.setup()
      vi.mocked(apiClient.getPosts).mockResolvedValue(mockResponse)
      renderWithLocation()

      await waitFor(() => {
        expect(apiClient.getPosts).toHaveBeenCalled()
      })

      const statusSelect = screen.getByLabelText('Status') as HTMLSelectElement
      await user.selectOptions(statusSelect, 'draft')

      await waitFor(() => {
        expect(screen.getByTestId('location').textContent).toBe('/?status=draft')
      })

      // Switching back to "all" should drop the param entirely.
      await user.selectOptions(statusSelect, 'all')

      await waitFor(() => {
        expect(screen.getByTestId('location').textContent).toBe('/')
      })
    })

    it('updates URL when Next/Prev are clicked', async () => {
      const user = userEvent.setup()
      vi.mocked(apiClient.getPosts).mockResolvedValue({
        posts: mockPosts,
        total: 60,
      })
      renderWithLocation()

      await waitFor(() => {
        expect(apiClient.getPosts).toHaveBeenCalled()
      })

      await user.click(screen.getByText('Next'))

      await waitFor(() => {
        expect(screen.getByTestId('location').textContent).toBe('/?page=2')
      })

      await user.click(screen.getByText('Prev'))

      // Page 1 is the default, so the param is removed instead of set to "1".
      await waitFor(() => {
        expect(screen.getByTestId('location').textContent).toBe('/')
      })
    })

    it('writes q to the URL after the search input debounces', async () => {
      const user = userEvent.setup()
      vi.mocked(apiClient.getPosts).mockResolvedValue(mockResponse)
      renderWithLocation()

      await waitFor(() => {
        expect(apiClient.getPosts).toHaveBeenCalled()
      })

      const searchInput = screen.getByLabelText('Search')
      await user.type(searchInput, 'react')

      await waitFor(
        () => {
          expect(screen.getByTestId('location').textContent).toBe('/?q=react')
        },
        { timeout: 1500 },
      )
    })

    it('clearing the search drops both q and page from the URL', async () => {
      const user = userEvent.setup()
      vi.mocked(apiClient.getPosts).mockResolvedValue(mockResponse)
      renderWithLocation(['/?q=foo&page=2'])

      const clearButton = await screen.findByTitle('Clear search')
      await user.click(clearButton)

      await waitFor(() => {
        expect(screen.getByTestId('location').textContent).toBe('/')
      })
    })

    it('changing the status filter resets the page in the URL', async () => {
      const user = userEvent.setup()
      vi.mocked(apiClient.getPosts).mockResolvedValue({
        posts: mockPosts,
        total: 60,
      })
      renderWithLocation(['/?page=3'])

      const statusSelect = await screen.findByLabelText('Status') as HTMLSelectElement
      await user.selectOptions(statusSelect, 'published')

      await waitFor(() => {
        expect(screen.getByTestId('location').textContent).toBe('/?status=published')
      })
    })

    it.each([
      ['/?page=0', 'zero'],
      ['/?page=-3', 'negative'],
      ['/?page=', 'empty'],
      ['/?page=abc', 'non-numeric'],
      ['/?page=1.9', 'decimal'],
    ])('invalid page param %s (%s) falls back to page 1', async (url) => {
      vi.mocked(apiClient.getPosts).mockResolvedValue(mockResponse)
      renderWithLocation([url])

      await waitFor(() => {
        expect(apiClient.getPosts).toHaveBeenCalledWith({
          status: undefined,
          q: undefined,
          limit: 20,
          offset: 0,
        })
      })
    })

    it('normalizes the URL when the requested page is past the end (via replace, no back-button trap)', async () => {
      // First fetch (page=999) returns empty; the component should rewrite
      // the URL to drop ?page= and re-fetch page 1.
      vi.mocked(apiClient.getPosts)
        .mockResolvedValueOnce({ posts: [], total: 5 })
        .mockResolvedValue(mockResponse)
      renderWithLocation(['/?page=999'])

      await waitFor(() => {
        expect(screen.getByTestId('location').textContent).toBe('/')
      })
      await waitFor(() => {
        expect(apiClient.getPosts).toHaveBeenCalledWith({
          status: undefined,
          q: undefined,
          limit: 20,
          offset: 0,
        })
      })

      // Replace, not push: if this normalization pushed, pressing Back would
      // return to ?page=999, which would normalize again, trapping the user.
      expect(screen.getByTestId('nav-type').textContent).toBe('REPLACE')
    })

    it('does not update the URL while IME composition is active', async () => {
      vi.mocked(apiClient.getPosts).mockResolvedValue(mockResponse)
      renderWithLocation()

      // Initial fetch is async via microtasks; settle it with real timers
      // before switching to fake timers so that waitFor can poll.
      await waitFor(() => {
        expect(apiClient.getPosts).toHaveBeenCalled()
      })

      // Fake timers from here on so debounce timing is deterministic.
      vi.useFakeTimers()
      try {
        const searchInput = screen.getByLabelText('Search')

        fireEvent.compositionStart(searchInput)
        fireEvent.change(searchInput, { target: { value: 'こんにちは' } })

        // Advance well past the 300ms debounce; URL must remain unchanged.
        await act(async () => {
          await vi.advanceTimersByTimeAsync(500)
        })
        expect(screen.getByTestId('location').textContent).toBe('/')

        fireEvent.compositionEnd(searchInput)

        // After compositionend the effect re-schedules the debounce timer.
        // Run it to completion and let React flush the resulting state update.
        await act(async () => {
          await vi.advanceTimersByTimeAsync(500)
        })
        expect(screen.getByTestId('location').textContent).toContain('q=')
      } finally {
        vi.useRealTimers()
      }
    })

    it('Clear keeps the status filter intact in the URL', async () => {
      const user = userEvent.setup()
      vi.mocked(apiClient.getPosts).mockResolvedValue(mockResponse)
      renderWithLocation(['/?status=draft&q=foo'])

      const clearButton = await screen.findByTitle('Clear search')
      await user.click(clearButton)

      await waitFor(() => {
        expect(screen.getByTestId('location').textContent).toBe('/?status=draft')
      })
    })

    it.each([
      ['/?page=1', '/', 'explicit default page'],
      ['/?status=all', '/', 'explicit default status'],
      ['/?q=', '/', 'empty query'],
      ['/?page=abc', '/', 'unparseable page'],
      ['/?page=1.9', '/', 'decimal page that truncates to 1'],
      ['/?page=02', '/?page=2', 'page with leading zero'],
      ['/?page=2.5', '/?page=2', 'decimal page that truncates to >1'],
      ['/?page=2&status=all&q=', '/?page=2', 'mixed defaults alongside a kept param'],
    ])('canonicalizes %s to %s (%s)', async (initialUrl, expectedUrl) => {
      vi.mocked(apiClient.getPosts).mockResolvedValue(mockResponse)
      renderWithLocation([initialUrl])

      await waitFor(() => {
        expect(screen.getByTestId('location').textContent).toBe(expectedUrl)
      })

      // Canonicalization uses replace so back doesn't bounce to the
      // non-canonical form.
      expect(screen.getByTestId('nav-type').textContent).toBe('REPLACE')
    })

    it.each([
      '/?page=2',
      '/?status=draft',
      '/?q=foo',
      '/?page=3&status=published&q=react',
    ])('leaves already-canonical URL %s untouched', async (url) => {
      vi.mocked(apiClient.getPosts).mockResolvedValue(mockResponse)
      renderWithLocation([url])

      // Wait long enough that any pending canon rewrite would have shown up.
      await waitFor(() => {
        expect(apiClient.getPosts).toHaveBeenCalled()
      })

      expect(screen.getByTestId('location').textContent).toBe(url)
      // Initial mount navigation type is POP, not REPLACE — confirms no
      // rewrite happened.
      expect(screen.getByTestId('nav-type').textContent).toBe('POP')
    })
  })

  describe('Reactions column', () => {
    it('shows the reaction total and reveals the breakdown on hover', async () => {
      const user = userEvent.setup()
      const postWithReactions: Post = {
        id: 99,
        title: 'Reacted Post',
        slug: 'reacted-post',
        content: 'x',
        status: 'published',
        tags: '',
        is_pinned: false,
        created_at: '2024-01-01T00:00:00Z',
        updated_at: '2024-01-01T00:00:00Z',
        published_at: '2024-01-01T00:00:00Z',
        view_count: 0,
        health_date: null,
        reactions: [
          { id: 1, emoji: '👍', label: 'いいね', count: 3, is_active: true },
          { id: 5, emoji: '🤔', label: 'なるほど', count: 2, is_active: false },
        ],
      }
      vi.mocked(apiClient.getPosts).mockResolvedValue({ posts: [postWithReactions], total: 1 })

      render(
        <MemoryRouter>
          <PostList />
        </MemoryRouter>,
      )

      // Total (3 + 2) is rendered as a focusable trigger.
      const trigger = await screen.findByRole('button', { name: 'Reactions: 5' })
      expect(trigger).toBeInTheDocument()

      // No popover until hover/focus.
      expect(screen.queryByRole('tooltip')).not.toBeInTheDocument()

      await user.hover(trigger)
      const tip = await screen.findByRole('tooltip')
      // The inactive type (🤔 / なるほど) is greyed out inside the popover; its
      // note lives in the title attribute (no visible 無効 marker).
      expect(within(tip).getByTitle('なるほど（無効）')).toBeInTheDocument()
    })
  })
})
