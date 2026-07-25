import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, waitFor, fireEvent } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter, Routes, Route } from 'react-router-dom'
import { PostEdit, formatHealthSummaryLine } from './PostEdit'
import { ModalProvider } from '../hooks/useModal'
import { apiClient } from '../api/client'
import type { Post, HealthDailySummary } from '../api/client'

// Fixed date for deterministic tests (UTC: 2025-06-15)
const FIXED_DATE = new Date('2025-06-15T12:00:00Z')
const EXPECTED_DEFAULT_SLUG = '2025-06-15'
const EXPECTED_DEFAULT_TITLE = '2025.06.15'

// Mock apiClient
vi.mock('../api/client', () => ({
  apiClient: {
    getPost: vi.fn(),
    createPost: vi.fn(),
    updatePost: vi.fn(),
    publishPost: vi.fn(),
    unpublishPost: vi.fn(),
    deletePost: vi.fn(),
    getTags: vi.fn(),
    previewMarkdown: vi.fn().mockResolvedValue(''),
    getHealthPlanetStatus: vi.fn(),
    getHealthDailySummary: vi.fn(),
  },
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

describe('PostEdit', () => {
  const mockDraftPost: Post = {
    id: 1,
    title: 'Draft Post',
    slug: 'draft-post',
    content: 'Draft content',
    status: 'draft',
    tags: 'Go, Testing',
    is_pinned: false,
    created_at: '2024-01-01T00:00:00Z',
    updated_at: '2024-01-01T00:00:00Z',
    published_at: null,
    view_count: 0,
    health_date: null,
  }

  const mockPublishedPost: Post = {
    id: 2,
    title: 'Published Post',
    slug: 'published-post',
    content: 'Published content',
    status: 'published',
    tags: 'Go',
    is_pinned: false,
    created_at: '2024-01-01T00:00:00Z',
    updated_at: '2024-01-01T00:00:00Z',
    published_at: '2024-01-01T10:00:00Z',
    view_count: 42,
    health_date: null,
  }

  beforeEach(() => {
    vi.clearAllMocks()
    vi.useFakeTimers({ shouldAdvanceTime: true })
    vi.setSystemTime(FIXED_DATE)
    // Mock for TagInput component
    vi.mocked(apiClient.getTags).mockResolvedValue([
      { name: 'Go', count: 5 },
      { name: 'React', count: 3 },
      { name: 'Testing', count: 2 },
    ])
    // Default: healthplanet integration disabled unless a test overrides it.
    vi.mocked(apiClient.getHealthPlanetStatus).mockResolvedValue({ enabled: false })
    // Default: no data, so tests that enable the integration but don't care
    // about the summary preview aren't required to mock it explicitly.
    vi.mocked(apiClient.getHealthDailySummary).mockResolvedValue({ found: false })
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  const renderCreateMode = async () => {
    const result = render(
      <MemoryRouter initialEntries={['/posts/new']}>
        <ModalProvider>
          <Routes>
            <Route path="/posts/new" element={<PostEdit />} />
            <Route path="/posts/:id/edit" element={<div>Edit Page</div>} />
          </Routes>
        </ModalProvider>
      </MemoryRouter>
    )
    // Wait for TagInput's getTags call to complete
    await waitFor(() => {
      expect(apiClient.getTags).toHaveBeenCalled()
    })
    return result
  }

  const renderEditMode = async (postId: number) => {
    const result = render(
      <MemoryRouter initialEntries={[`/posts/${postId}/edit`]}>
        <ModalProvider>
          <Routes>
            <Route path="/posts/:id/edit" element={<PostEdit />} />
            <Route path="/posts" element={<div>Post List</div>} />
          </Routes>
        </ModalProvider>
      </MemoryRouter>
    )
    // Wait for TagInput's getTags call to complete
    await waitFor(() => {
      expect(apiClient.getTags).toHaveBeenCalled()
    })
    return result
  }

  describe('Create mode rendering', () => {
    it('should render page title as "New Post"', async () => {
      await renderCreateMode()
      expect(screen.getByText('New Post')).toBeInTheDocument()
    })

    it('should render form fields with default values', async () => {
      await renderCreateMode()
      expect(screen.getByLabelText(/Title/)).toHaveValue(EXPECTED_DEFAULT_TITLE)
      expect(screen.getByLabelText(/Slug/)).toHaveValue(EXPECTED_DEFAULT_SLUG)
      // TagInput component has different label association, so only verify label display
      expect(screen.getByText('Tags')).toBeInTheDocument()
      // MarkdownEditor component has different label association, so only verify label display
      expect(screen.getByText('Content (Markdown)')).toBeInTheDocument()
    })

    it('should render save button', async () => {
      await renderCreateMode()
      expect(screen.getByRole('button', { name: 'Save' })).toBeInTheDocument()
    })

    it('should not render publish/unpublish/delete buttons', async () => {
      await renderCreateMode()
      expect(screen.queryByRole('button', { name: 'Publish' })).not.toBeInTheDocument()
      expect(screen.queryByRole('button', { name: 'Unpublish' })).not.toBeInTheDocument()
      expect(screen.queryByRole('button', { name: 'Delete' })).not.toBeInTheDocument()
    })

    it('should render back link', async () => {
      await renderCreateMode()
      expect(screen.getByText('← Back to posts')).toBeInTheDocument()
    })

    it('should have required attributes on title and slug fields', async () => {
      await renderCreateMode()
      expect(screen.getByLabelText(/Title/)).toBeRequired()
      expect(screen.getByLabelText(/Slug/)).toBeRequired()
    })
  })

  describe('Edit mode rendering', () => {
    it('should show loading state initially', async () => {
      vi.mocked(apiClient.getPost).mockReturnValue(new Promise(() => {}))
      await renderEditMode(1)
      expect(screen.getByText('Loading...')).toBeInTheDocument()
    })

    it('should not apply date defaults to title and slug in edit mode', async () => {
      vi.mocked(apiClient.getPost).mockResolvedValue(mockDraftPost)
      await renderEditMode(1)

      await waitFor(() => {
        expect(screen.getByLabelText(/Title/)).toHaveValue('Draft Post')
      })
      // Verify slug is from the post data, not a date default
      expect(screen.getByLabelText(/Slug/)).toHaveValue('draft-post')
    })

    it('should render page title as "Edit Post"', async () => {
      vi.mocked(apiClient.getPost).mockResolvedValue(mockDraftPost)
      await renderEditMode(1)

      await waitFor(() => {
        expect(screen.getByText('Edit Post')).toBeInTheDocument()
      })
    })

    it('should load and display post data', async () => {
      vi.mocked(apiClient.getPost).mockResolvedValue(mockDraftPost)
      await renderEditMode(1)

      await waitFor(() => {
        expect(screen.getByLabelText(/Title/)).toHaveValue('Draft Post')
      })

      expect(screen.getByLabelText(/Slug/)).toHaveValue('draft-post')
      // TagInput component displays tags as chips
      expect(screen.getByText('Go')).toBeInTheDocument()
      expect(screen.getByText('Testing')).toBeInTheDocument()
      // MarkdownEditor's textarea cannot be accessed directly, so verify API call
      expect(apiClient.getPost).toHaveBeenCalledWith(1)
    })

    it('should show error message when post load fails', async () => {
      vi.mocked(apiClient.getPost).mockRejectedValue(new Error('Post not found'))
      await renderEditMode(1)

      await waitFor(() => {
        expect(screen.getByText('Post not found')).toBeInTheDocument()
      })
    })

    it('should render publish button for draft post', async () => {
      vi.mocked(apiClient.getPost).mockResolvedValue(mockDraftPost)
      await renderEditMode(1)

      await waitFor(() => {
        expect(screen.getByRole('button', { name: 'Publish' })).toBeInTheDocument()
      })

      expect(screen.queryByRole('button', { name: 'Unpublish' })).not.toBeInTheDocument()
    })

    it('should render unpublish button for published post', async () => {
      vi.mocked(apiClient.getPost).mockResolvedValue(mockPublishedPost)
      await renderEditMode(2)

      await waitFor(() => {
        expect(screen.getByRole('button', { name: 'Unpublish' })).toBeInTheDocument()
      })

      expect(screen.queryByRole('button', { name: 'Publish' })).not.toBeInTheDocument()
    })

    it('should render delete button', async () => {
      vi.mocked(apiClient.getPost).mockResolvedValue(mockDraftPost)
      await renderEditMode(1)

      await waitFor(() => {
        expect(screen.getByRole('button', { name: 'Delete' })).toBeInTheDocument()
      })
    })

    it('should render status badge', async () => {
      vi.mocked(apiClient.getPost).mockResolvedValue(mockDraftPost)
      await renderEditMode(1)

      await waitFor(() => {
        expect(screen.getByText('Draft')).toBeInTheDocument()
      })
    })

    it('renders View ↗ as a clickable link that opens the public page in a new tab for a published post', async () => {
      vi.mocked(apiClient.getPost).mockResolvedValue(mockPublishedPost)
      await renderEditMode(2)

      const viewLink = await screen.findByRole('link', { name: /View/ })
      expect(viewLink).toHaveAttribute('href', '/posts/published-post')
      expect(viewLink).toHaveAttribute('target', '_blank')
      expect(viewLink).toHaveAttribute('rel', expect.stringContaining('noopener'))
    })

    it('renders View ↗ as a disabled span for a draft post', async () => {
      vi.mocked(apiClient.getPost).mockResolvedValue(mockDraftPost)
      await renderEditMode(1)

      await waitFor(() => {
        expect(screen.getByLabelText(/Title/)).toHaveValue('Draft Post')
      })

      // No link role — a draft View label is a non-interactive span.
      expect(screen.queryByRole('link', { name: /View/ })).not.toBeInTheDocument()

      const viewLabel = screen.getByText(/View ↗/)
      expect(viewLabel.tagName).toBe('SPAN')
      expect(viewLabel).toHaveAttribute('aria-disabled', 'true')
    })
  })

  describe('Post metadata display', () => {
    it('should display created_at and updated_at for draft post', async () => {
      vi.mocked(apiClient.getPost).mockResolvedValue(mockDraftPost)
      await renderEditMode(1)

      await waitFor(() => {
        expect(screen.getByText('Created:')).toBeInTheDocument()
      })

      expect(screen.getByText('Updated:')).toBeInTheDocument()
      // Don't display published date when published_at is null
      expect(screen.queryByText('Published:')).not.toBeInTheDocument()
    })

    it('should display published_at for published post', async () => {
      vi.mocked(apiClient.getPost).mockResolvedValue(mockPublishedPost)
      await renderEditMode(2)

      await waitFor(() => {
        expect(screen.getByText('Published:')).toBeInTheDocument()
      })
    })

    it('should format dates in ISO 8601 format (yyyy-MM-dd HH:mm:ss)', async () => {
      const postWithSpecificDates: Post = {
        ...mockPublishedPost,
        created_at: '2024-06-15T10:30:45Z',
        updated_at: '2024-06-20T14:15:30Z',
        published_at: '2024-06-18T09:00:00Z',
      }
      vi.mocked(apiClient.getPost).mockResolvedValue(postWithSpecificDates)
      await renderEditMode(2)

      await waitFor(() => {
        expect(screen.getByText('Created:')).toBeInTheDocument()
      })

      // Verify displayed in ISO 8601 format (yyyy-MM-dd HH:mm:ss)
      const datePattern = /\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}/
      const metadataSection = screen.getByText('Created:').closest('div')?.parentElement
      expect(metadataSection?.textContent).toMatch(datePattern)
    })

    it('should not display metadata in create mode', async () => {
      await renderCreateMode()

      expect(screen.queryByText('Created:')).not.toBeInTheDocument()
      expect(screen.queryByText('Updated:')).not.toBeInTheDocument()
      expect(screen.queryByText('Published:')).not.toBeInTheDocument()
    })
  })

  describe('Form input', () => {
    it('should allow entering title', async () => {
      const user = userEvent.setup()
      await renderCreateMode()

      const titleInput = screen.getByLabelText(/Title/)
      await user.clear(titleInput)
      await user.type(titleInput, 'New Post Title')

      expect(titleInput).toHaveValue('New Post Title')
    })

    it('should allow entering slug', async () => {
      const user = userEvent.setup()
      await renderCreateMode()

      const slugInput = screen.getByLabelText(/Slug/)
      await user.clear(slugInput)
      await user.type(slugInput, 'new-post-slug')

      expect(slugInput).toHaveValue('new-post-slug')
    })

    it('should allow entering tags', async () => {
      const user = userEvent.setup()
      await renderCreateMode()

      // Get TagInput component's input
      const tagsInput = screen.getByLabelText('Tag input')
      // Tags are added when separated by comma
      await user.type(tagsInput, 'NewTag,')

      // Added tags are displayed as chips
      await waitFor(() => {
        expect(screen.getByText('NewTag')).toBeInTheDocument()
      })
    })

    it('should render markdown editor', async () => {
      await renderCreateMode()
      // Verify MarkdownEditor component is rendered
      expect(screen.getByText('Content (Markdown)')).toBeInTheDocument()
      // MDEditor displays toolbar
      expect(screen.getByRole('button', { name: /bold/i })).toBeInTheDocument()
    })
  })

  describe('Create post', () => {
    it('should show validation error when both are whitespace', async () => {
      const user = userEvent.setup()
      await renderCreateMode()

      await user.clear(screen.getByLabelText(/Title/))
      await user.type(screen.getByLabelText(/Title/), '   ')
      await user.clear(screen.getByLabelText(/Slug/))
      await user.type(screen.getByLabelText(/Slug/), '   ')
      await user.click(screen.getByRole('button', { name: 'Save' }))

      await waitFor(() => {
        expect(screen.getByText('Title and slug are required')).toBeInTheDocument()
      })

      expect(apiClient.createPost).not.toHaveBeenCalled()
    })

    it('should create post successfully and navigate to edit page', async () => {
      const user = userEvent.setup()
      const createdPost: Post = {
        ...mockDraftPost,
        id: 999,
        title: 'New Post',
        slug: 'new-post',
        content: '',
        tags: 'React',
      }

      vi.mocked(apiClient.createPost).mockResolvedValue(createdPost)

      await renderCreateMode()

      await user.clear(screen.getByLabelText(/Title/))
      await user.type(screen.getByLabelText(/Title/), 'New Post')
      await user.clear(screen.getByLabelText(/Slug/))
      await user.type(screen.getByLabelText(/Slug/), 'new-post')
      // Add tags via TagInput (separated by comma)
      await user.type(screen.getByLabelText('Tag input'), 'React,')
      // Skip content input since MarkdownEditor has different label association
      await user.click(screen.getByRole('button', { name: 'Save' }))

      await waitFor(() => {
        expect(apiClient.createPost).toHaveBeenCalledWith({
          title: 'New Post',
          slug: 'new-post',
          content: '',
          tags: 'React',
          is_pinned: false,
          health_date: null,
        })
      })

      // Verify modal is displayed
      await waitFor(() => {
        expect(screen.getByText('Post created')).toBeInTheDocument()
      })
      // Click OK to close modal
      await user.click(screen.getByRole('button', { name: 'OK' }))

      await waitFor(() => {
        expect(mockNavigate).toHaveBeenCalledWith('/posts/999/edit')
      })
    })

    it('should show error message when create fails', async () => {
      const user = userEvent.setup()
      vi.mocked(apiClient.createPost).mockRejectedValue(
        new Error('Slug already exists')
      )

      await renderCreateMode()

      await user.clear(screen.getByLabelText(/Title/))
      await user.type(screen.getByLabelText(/Title/), 'New Post')
      await user.type(screen.getByLabelText(/Slug/), 'duplicate-slug')
      await user.click(screen.getByRole('button', { name: 'Save' }))

      await waitFor(() => {
        expect(screen.getByText('Slug already exists')).toBeInTheDocument()
      })

      // Modal should not be displayed on error
      expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
      expect(mockNavigate).not.toHaveBeenCalled()
    })

    it('should disable form fields during save', async () => {
      const user = userEvent.setup()
      let resolveCreate: (post: Post) => void
      vi.mocked(apiClient.createPost).mockReturnValue(
        new Promise((resolve) => {
          resolveCreate = resolve as (post: Post) => void
        })
      )

      await renderCreateMode()

      const titleInput = screen.getByLabelText(/Title/)
      const slugInput = screen.getByLabelText(/Slug/)
      const tagsInput = screen.getByLabelText('Tag input')
      const saveButton = screen.getByRole('button', { name: 'Save' })

      await user.type(titleInput, 'Test')
      await user.type(slugInput, 'test')
      await user.click(saveButton)

      await waitFor(() => {
        expect(titleInput).toBeDisabled()
        expect(slugInput).toBeDisabled()
        expect(tagsInput).toBeDisabled()
        // MarkdownEditor's disabled state is managed internally,
        // so only verify button's disabled state
        expect(saveButton).toBeDisabled()
        expect(screen.getByRole('button', { name: 'Saving...' })).toBeInTheDocument()
      })

      resolveCreate!(mockDraftPost)

      // Verify modal is displayed
      await waitFor(() => {
        expect(screen.getByText('Post created')).toBeInTheDocument()
      })

      // Click OK to close modal
      await user.click(screen.getByRole('button', { name: 'OK' }))

      await waitFor(() => {
        expect(titleInput).not.toBeDisabled()
      })
    })
  })

  describe('Update post', () => {
    it('should update post successfully', async () => {
      const user = userEvent.setup()
      vi.mocked(apiClient.getPost).mockResolvedValue(mockDraftPost)
      vi.mocked(apiClient.updatePost).mockResolvedValue({
        ...mockDraftPost,
        title: 'Updated Title',
      })

      await renderEditMode(1)

      await waitFor(() => {
        expect(screen.getByLabelText(/Title/)).toHaveValue('Draft Post')
      })

      const titleInput = screen.getByLabelText(/Title/)
      await user.clear(titleInput)
      await user.type(titleInput, 'Updated Title')
      await user.click(screen.getByRole('button', { name: 'Save' }))

      await waitFor(() => {
        expect(apiClient.updatePost).toHaveBeenCalledWith(1, {
          title: 'Updated Title',
          slug: 'draft-post',
          content: 'Draft content',
          tags: 'Go, Testing',
          is_pinned: false,
          health_date: null,
        })
      })

      // Verify modal is displayed
      await waitFor(() => {
        expect(screen.getByText('Post updated')).toBeInTheDocument()
      })
      // Click OK to close modal
      await user.click(screen.getByRole('button', { name: 'OK' }))

      expect(mockNavigate).not.toHaveBeenCalled()
    })

    it('should show error message when update fails', async () => {
      const user = userEvent.setup()
      vi.mocked(apiClient.getPost).mockResolvedValue(mockDraftPost)
      vi.mocked(apiClient.updatePost).mockRejectedValue(
        new Error('Failed to update')
      )

      await renderEditMode(1)

      await waitFor(() => {
        expect(screen.getByLabelText(/Title/)).toHaveValue('Draft Post')
      })

      await user.click(screen.getByRole('button', { name: 'Save' }))

      await waitFor(() => {
        expect(screen.getByText('Failed to update')).toBeInTheDocument()
      })

      // Modal should not be displayed on error
      expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
    })
  })

  describe('Publish post', () => {
    it('should publish draft post successfully', async () => {
      const user = userEvent.setup()
      vi.mocked(apiClient.getPost).mockResolvedValue(mockDraftPost)
      vi.mocked(apiClient.publishPost).mockResolvedValue({
        ...mockDraftPost,
        status: 'published',
      })

      await renderEditMode(1)

      await waitFor(() => {
        expect(screen.getByRole('button', { name: 'Publish' })).toBeInTheDocument()
      })

      await user.click(screen.getByRole('button', { name: 'Publish' }))

      await waitFor(() => {
        expect(apiClient.publishPost).toHaveBeenCalledWith(1)
      })

      // Verify modal is displayed
      await waitFor(() => {
        expect(screen.getByText('Post published')).toBeInTheDocument()
      })
      // Click OK to close modal
      await user.click(screen.getByRole('button', { name: 'OK' }))
    })

    it('should show error message when publish fails', async () => {
      const user = userEvent.setup()
      vi.mocked(apiClient.getPost).mockResolvedValue(mockDraftPost)
      vi.mocked(apiClient.publishPost).mockRejectedValue(
        new Error('Failed to publish')
      )

      await renderEditMode(1)

      await waitFor(() => {
        expect(screen.getByRole('button', { name: 'Publish' })).toBeInTheDocument()
      })

      await user.click(screen.getByRole('button', { name: 'Publish' }))

      await waitFor(() => {
        expect(screen.getByText('Failed to publish')).toBeInTheDocument()
      })

      // Modal should not be displayed on error
      expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
    })

    it('should update status badge after publish', async () => {
      const user = userEvent.setup()
      vi.mocked(apiClient.getPost).mockResolvedValue(mockDraftPost)
      vi.mocked(apiClient.publishPost).mockResolvedValue({
        ...mockDraftPost,
        status: 'published',
      })

      await renderEditMode(1)

      await waitFor(() => {
        expect(screen.getByText('Draft')).toBeInTheDocument()
      })

      await user.click(screen.getByRole('button', { name: 'Publish' }))

      await waitFor(() => {
        expect(screen.getByText('Published')).toBeInTheDocument()
      })

      expect(screen.queryByText('Draft')).not.toBeInTheDocument()
    })
  })

  describe('Unpublish post', () => {
    it('should unpublish published post successfully', async () => {
      const user = userEvent.setup()
      vi.mocked(apiClient.getPost).mockResolvedValue(mockPublishedPost)
      vi.mocked(apiClient.unpublishPost).mockResolvedValue({
        ...mockPublishedPost,
        status: 'draft',
      })

      await renderEditMode(2)

      await waitFor(() => {
        expect(screen.getByRole('button', { name: 'Unpublish' })).toBeInTheDocument()
      })

      await user.click(screen.getByRole('button', { name: 'Unpublish' }))

      await waitFor(() => {
        expect(apiClient.unpublishPost).toHaveBeenCalledWith(2)
      })

      // Verify modal is displayed
      await waitFor(() => {
        expect(screen.getByText('Post unpublished')).toBeInTheDocument()
      })
      // Click OK to close modal
      await user.click(screen.getByRole('button', { name: 'OK' }))
    })

    it('should show error message when unpublish fails', async () => {
      const user = userEvent.setup()
      vi.mocked(apiClient.getPost).mockResolvedValue(mockPublishedPost)
      vi.mocked(apiClient.unpublishPost).mockRejectedValue(
        new Error('Failed to unpublish')
      )

      await renderEditMode(2)

      await waitFor(() => {
        expect(screen.getByRole('button', { name: 'Unpublish' })).toBeInTheDocument()
      })

      await user.click(screen.getByRole('button', { name: 'Unpublish' }))

      await waitFor(() => {
        expect(screen.getByText('Failed to unpublish')).toBeInTheDocument()
      })

      // Modal should not be displayed on error
      expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
    })
  })

  describe('Delete post', () => {
    it('should not delete when confirm is cancelled', async () => {
      const user = userEvent.setup()
      vi.mocked(apiClient.getPost).mockResolvedValue(mockDraftPost)

      await renderEditMode(1)

      await waitFor(() => {
        expect(screen.getByRole('button', { name: 'Delete' })).toBeInTheDocument()
      })

      await user.click(screen.getByRole('button', { name: 'Delete' }))

      // Verify confirmation dialog is displayed
      await waitFor(() => {
        expect(screen.getByText('Are you sure you want to delete this post? This action cannot be undone.')).toBeInTheDocument()
        expect(screen.getByText('Delete Post')).toBeInTheDocument()
      })

      // Click Cancel
      await user.click(screen.getByRole('button', { name: 'Cancel' }))

      expect(apiClient.deletePost).not.toHaveBeenCalled()
    })

    it('should delete post when confirm is accepted', async () => {
      const user = userEvent.setup()
      vi.mocked(apiClient.getPost).mockResolvedValue(mockDraftPost)
      vi.mocked(apiClient.deletePost).mockResolvedValue(undefined)

      await renderEditMode(1)

      await waitFor(() => {
        expect(screen.getByRole('button', { name: 'Delete' })).toBeInTheDocument()
      })

      await user.click(screen.getByRole('button', { name: 'Delete' }))

      // Verify confirmation dialog is displayed
      await waitFor(() => {
        expect(screen.getByText('Are you sure you want to delete this post? This action cannot be undone.')).toBeInTheDocument()
      })

      // Click Delete button (inside confirmation dialog)
      const dialogDeleteButton = screen.getByRole('dialog').querySelector('button.bg-red-600')
      await user.click(dialogDeleteButton!)

      await waitFor(() => {
        expect(apiClient.deletePost).toHaveBeenCalledWith(1)
      })

      // Delete completion modal is displayed
      await waitFor(() => {
        expect(screen.getByText('Post deleted')).toBeInTheDocument()
      })
      // Click OK to close modal
      await user.click(screen.getByRole('button', { name: 'OK' }))

      await waitFor(() => {
        expect(mockNavigate).toHaveBeenCalledWith('/posts')
      })
    })

    it('should show error message when delete fails', async () => {
      const user = userEvent.setup()
      vi.mocked(apiClient.getPost).mockResolvedValue(mockDraftPost)
      vi.mocked(apiClient.deletePost).mockRejectedValue(
        new Error('Failed to delete')
      )

      await renderEditMode(1)

      await waitFor(() => {
        expect(screen.getByRole('button', { name: 'Delete' })).toBeInTheDocument()
      })

      await user.click(screen.getByRole('button', { name: 'Delete' }))

      // Verify confirmation dialog is displayed
      await waitFor(() => {
        expect(screen.getByText('Are you sure you want to delete this post? This action cannot be undone.')).toBeInTheDocument()
      })

      // Click Delete button (inside confirmation dialog)
      const dialogDeleteButton = screen.getByRole('dialog').querySelector('button.bg-red-600')
      await user.click(dialogDeleteButton!)

      await waitFor(() => {
        expect(screen.getByText('Failed to delete')).toBeInTheDocument()
      })

      expect(mockNavigate).not.toHaveBeenCalled()
    })
  })

  it('shows the reaction breakdown and keeps it after saving', async () => {
    const user = userEvent.setup()
    const postWithReactions: Post = {
      id: 1,
      title: 'Reacted',
      slug: 'reacted',
      content: 'body',
      status: 'published',
      tags: '',
      is_pinned: false,
      created_at: '2024-01-01T00:00:00Z',
      updated_at: '2024-01-01T00:00:00Z',
      published_at: '2024-01-01T00:00:00Z',
      view_count: 0,
      health_date: null,
      reactions: [
        { id: 1, emoji: '👍', label: 'いいね', count: 4, is_active: true },
        { id: 5, emoji: '🤔', label: 'なるほど', count: 1, is_active: false },
      ],
    }
    vi.mocked(apiClient.getPost).mockResolvedValue(postWithReactions)
    // Update response intentionally omits reactions (mutation responses don't carry them).
    vi.mocked(apiClient.updatePost).mockResolvedValue({ ...postWithReactions, reactions: undefined })

    render(
      <ModalProvider>
        <MemoryRouter initialEntries={['/posts/1/edit']}>
          <Routes>
            <Route path="/posts/:id/edit" element={<PostEdit />} />
          </Routes>
        </MemoryRouter>
      </ModalProvider>,
    )

    // Inline breakdown is shown; the inactive type (🤔 / なるほど) is greyed out,
    // identified via its title (no visible 無効 marker).
    expect(await screen.findByTitle('なるほど（無効）')).toBeInTheDocument()

    // Save; dismiss the "Post updated" alert the same way the other save tests do.
    await user.click(screen.getByRole('button', { name: 'Save' }))
    await waitFor(() => expect(apiClient.updatePost).toHaveBeenCalled())
    await waitFor(() => expect(screen.getByText('Post updated')).toBeInTheDocument())
    await user.click(screen.getByRole('button', { name: 'OK' }))

    // The update response carried no reactions, but the breakdown must persist.
    expect(screen.getByTitle('なるほど（無効）')).toBeInTheDocument()
    // Active reaction count is preserved too (whole array kept, not just the inactive entry).
    expect(screen.getByText('👍 4')).toBeInTheDocument()
  })

  describe('Error handling', () => {
    it('should clear previous error on new save attempt', async () => {
      const user = userEvent.setup()
      vi.mocked(apiClient.createPost)
        .mockRejectedValueOnce(new Error('First error'))
        .mockResolvedValueOnce(mockDraftPost)

      await renderCreateMode()

      await user.type(screen.getByLabelText(/Title/), 'Test')
      await user.type(screen.getByLabelText(/Slug/), 'test')
      await user.click(screen.getByRole('button', { name: 'Save' }))

      await waitFor(() => {
        expect(screen.getByText('First error')).toBeInTheDocument()
      })

      await user.click(screen.getByRole('button', { name: 'Save' }))

      await waitFor(() => {
        expect(screen.queryByText('First error')).not.toBeInTheDocument()
      })
    })

    it('should show default error message for non-Error exceptions', async () => {
      const user = userEvent.setup()
      vi.mocked(apiClient.createPost).mockRejectedValue('Unknown error')

      await renderCreateMode()

      await user.type(screen.getByLabelText(/Title/), 'Test')
      await user.type(screen.getByLabelText(/Slug/), 'test')
      await user.click(screen.getByRole('button', { name: 'Save' }))

      await waitFor(() => {
        expect(screen.getByText('Failed to save')).toBeInTheDocument()
      })
    })
  })

  describe('health_date field', () => {
    it('hides the field when healthplanet is disabled', async () => {
      vi.mocked(apiClient.getHealthPlanetStatus).mockResolvedValue({ enabled: false })
      await renderCreateMode()

      await waitFor(() => {
        expect(screen.queryByLabelText('健康データの日付')).not.toBeInTheDocument()
      })
    })

    it('shows the field when enabled and sends the date on create', async () => {
      const user = userEvent.setup()
      vi.mocked(apiClient.getHealthPlanetStatus).mockResolvedValue({ enabled: true })
      vi.mocked(apiClient.createPost).mockResolvedValue(mockDraftPost)
      await renderCreateMode()

      const field = await screen.findByLabelText('健康データの日付')
      fireEvent.change(field, { target: { value: '2026-07-20' } })

      await user.click(screen.getByRole('button', { name: 'Save' }))

      await waitFor(() => {
        expect(apiClient.createPost).toHaveBeenCalledWith(
          expect.objectContaining({ health_date: '2026-07-20' })
        )
      })
    })

    it('sends null when the field is left empty', async () => {
      const user = userEvent.setup()
      vi.mocked(apiClient.getHealthPlanetStatus).mockResolvedValue({ enabled: true })
      vi.mocked(apiClient.createPost).mockResolvedValue(mockDraftPost)
      await renderCreateMode()

      await screen.findByLabelText('健康データの日付')
      await user.click(screen.getByRole('button', { name: 'Save' }))

      await waitFor(() => {
        expect(apiClient.createPost).toHaveBeenCalledWith(
          expect.objectContaining({ health_date: null })
        )
      })
    })

    it('seeds the field from the loaded post in edit mode', async () => {
      vi.mocked(apiClient.getHealthPlanetStatus).mockResolvedValue({ enabled: true })
      vi.mocked(apiClient.getPost).mockResolvedValue({
        ...mockDraftPost,
        health_date: '2026-07-15',
      })
      await renderEditMode(1)

      const field = await screen.findByLabelText('健康データの日付')
      expect(field).toHaveValue('2026-07-15')
    })

    it('preserves health_date when integration is disabled (regression)', async () => {
      const user = userEvent.setup()
      // Regression: when healthplanet is disabled, the field is hidden,
      // but saving must preserve the stored health_date value
      vi.mocked(apiClient.getHealthPlanetStatus).mockResolvedValue({ enabled: false })
      const postWithHealthDate: Post = {
        ...mockDraftPost,
        health_date: '2026-07-10',
      }
      vi.mocked(apiClient.getPost).mockResolvedValue(postWithHealthDate)
      vi.mocked(apiClient.updatePost).mockResolvedValue({
        ...postWithHealthDate,
        title: 'Updated Title',
      })

      await renderEditMode(1)

      // Verify the health_date field is NOT displayed
      await waitFor(() => {
        expect(screen.getByLabelText(/Title/)).toHaveValue('Draft Post')
      })
      expect(screen.queryByLabelText('健康データの日付')).not.toBeInTheDocument()

      // Change title and save
      const titleInput = screen.getByLabelText(/Title/)
      await user.clear(titleInput)
      await user.type(titleInput, 'Updated Title')
      await user.click(screen.getByRole('button', { name: 'Save' }))

      // Assert that updatePost was called with the health_date preserved
      await waitFor(() => {
        expect(apiClient.updatePost).toHaveBeenCalledWith(
          1,
          expect.objectContaining({ health_date: '2026-07-10' })
        )
      })

      // Verify success modal appears
      await waitFor(() => {
        expect(screen.getByText('Post updated')).toBeInTheDocument()
      })
    })
  })

  describe('health data preview', () => {
    it('shows the formatted summary line after selecting a date', async () => {
      vi.mocked(apiClient.getHealthPlanetStatus).mockResolvedValue({ enabled: true })
      vi.mocked(apiClient.getHealthDailySummary).mockResolvedValue({
        found: true,
        weight: 72.1,
        systolic: 119,
        diastolic: 82,
        pulse: 63,
        body_fat: 20.8,
      })
      await renderCreateMode()

      const field = await screen.findByLabelText('健康データの日付')
      fireEvent.change(field, { target: { value: '2026-07-20' } })

      await waitFor(() => {
        expect(apiClient.getHealthDailySummary).toHaveBeenCalledWith('2026-07-20')
      })
      await waitFor(() => {
        expect(
          screen.getByText('体重 72.1kg ・ 体脂肪率 20.8% ・ 血圧 119/82 ・ 脈拍 63bpm')
        ).toBeInTheDocument()
      })
    })

    it('shows a no-data message when the day has no health data', async () => {
      vi.mocked(apiClient.getHealthPlanetStatus).mockResolvedValue({ enabled: true })
      vi.mocked(apiClient.getHealthDailySummary).mockResolvedValue({ found: false })
      await renderCreateMode()

      const field = await screen.findByLabelText('健康データの日付')
      fireEvent.change(field, { target: { value: '2026-07-21' } })

      await waitFor(() => {
        expect(screen.getByText('この日の健康データはありません')).toBeInTheDocument()
      })
    })

    it('hides the preview when the date is cleared', async () => {
      vi.mocked(apiClient.getHealthPlanetStatus).mockResolvedValue({ enabled: true })
      vi.mocked(apiClient.getHealthDailySummary).mockResolvedValue({
        found: true,
        weight: 72.1,
      })
      await renderCreateMode()

      const field = await screen.findByLabelText('健康データの日付')
      fireEvent.change(field, { target: { value: '2026-07-20' } })
      await waitFor(() => {
        expect(screen.getByText('体重 72.1kg')).toBeInTheDocument()
      })

      fireEvent.change(field, { target: { value: '' } })
      await waitFor(() => {
        expect(screen.queryByText('体重 72.1kg')).not.toBeInTheDocument()
        expect(screen.queryByText('この日の健康データはありません')).not.toBeInTheDocument()
      })
    })

    it('shows nothing when the fetch fails', async () => {
      vi.mocked(apiClient.getHealthPlanetStatus).mockResolvedValue({ enabled: true })
      vi.mocked(apiClient.getHealthDailySummary).mockRejectedValue(new Error('network error'))
      await renderCreateMode()

      const field = await screen.findByLabelText('健康データの日付')
      fireEvent.change(field, { target: { value: '2026-07-20' } })

      await waitFor(() => {
        expect(apiClient.getHealthDailySummary).toHaveBeenCalled()
      })
      expect(screen.queryByText('この日の健康データはありません')).not.toBeInTheDocument()
    })
  })

  describe('formatHealthSummaryLine', () => {
    it('returns null when not found', () => {
      expect(formatHealthSummaryLine({ found: false })).toBeNull()
    })

    it('returns null when found but no displayable metrics', () => {
      expect(formatHealthSummaryLine({ found: true })).toBeNull()
    })

    it('orders 体重 → 体脂肪率 → 血圧 → 脈拍 with ・ separators', () => {
      const s: HealthDailySummary = {
        found: true,
        weight: 72.1,
        systolic: 119,
        diastolic: 82,
        pulse: 63,
        body_fat: 20.8,
      }
      expect(formatHealthSummaryLine(s)).toBe('体重 72.1kg ・ 体脂肪率 20.8% ・ 血圧 119/82 ・ 脈拍 63bpm')
    })

    it('omits blood pressure when only systolic is present', () => {
      const s: HealthDailySummary = { found: true, weight: 72.1, systolic: 119 }
      expect(formatHealthSummaryLine(s)).toBe('体重 72.1kg')
    })

    it('omits blood pressure when only diastolic is present', () => {
      const s: HealthDailySummary = { found: true, weight: 72.1, diastolic: 82 }
      expect(formatHealthSummaryLine(s)).toBe('体重 72.1kg')
    })

    it('renders a single metric with no separator', () => {
      expect(formatHealthSummaryLine({ found: true, pulse: 63 })).toBe('脈拍 63bpm')
    })

    it('places separators only between present items', () => {
      const s: HealthDailySummary = { found: true, systolic: 119, diastolic: 82, body_fat: 20.8 }
      expect(formatHealthSummaryLine(s)).toBe('体脂肪率 20.8% ・ 血圧 119/82')
    })
  })
})
