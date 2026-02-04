import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter, Routes, Route } from 'react-router-dom'
import { PostEdit } from './PostEdit'
import { ModalProvider } from '../hooks/useModal'
import { apiClient } from '../api/client'
import type { Post } from '../api/client'

// apiClient をモック
vi.mock('../api/client', () => ({
  apiClient: {
    getPost: vi.fn(),
    createPost: vi.fn(),
    updatePost: vi.fn(),
    publishPost: vi.fn(),
    unpublishPost: vi.fn(),
    deletePost: vi.fn(),
    getTags: vi.fn(),
  },
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
  }

  beforeEach(() => {
    vi.clearAllMocks()
    // TagInputコンポーネント用のモック
    vi.mocked(apiClient.getTags).mockResolvedValue([
      { name: 'Go', count: 5 },
      { name: 'React', count: 3 },
      { name: 'Testing', count: 2 },
    ])
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
    // TagInput の getTags 呼び出し完了を待つ
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
    // TagInput の getTags 呼び出し完了を待つ
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

    it('should render empty form fields', async () => {
      await renderCreateMode()
      expect(screen.getByLabelText(/Title/)).toHaveValue('')
      expect(screen.getByLabelText(/Slug/)).toHaveValue('')
      // TagInputコンポーネントはlabel関連付けが異なるため、ラベル表示のみ確認
      expect(screen.getByText('Tags')).toBeInTheDocument()
      // MarkdownEditor コンポーネントはlabel関連付けが異なるため、ラベル表示のみ確認
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
      // TagInputコンポーネントはTagsをチップで表示
      expect(screen.getByText('Go')).toBeInTheDocument()
      expect(screen.getByText('Testing')).toBeInTheDocument()
      // MarkdownEditorのtextareaは直接取得できないため、APIコールを確認
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
  })

  describe('Post metadata display', () => {
    it('should display created_at and updated_at for draft post', async () => {
      vi.mocked(apiClient.getPost).mockResolvedValue(mockDraftPost)
      await renderEditMode(1)

      await waitFor(() => {
        expect(screen.getByText('Created:')).toBeInTheDocument()
      })

      expect(screen.getByText('Updated:')).toBeInTheDocument()
      // published_at が null の場合は公開日を表示しない
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

      // ISO 8601形式（yyyy-MM-dd HH:mm:ss）で表示されることを確認
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
      await user.type(titleInput, 'New Post Title')

      expect(titleInput).toHaveValue('New Post Title')
    })

    it('should allow entering slug', async () => {
      const user = userEvent.setup()
      await renderCreateMode()

      const slugInput = screen.getByLabelText(/Slug/)
      await user.type(slugInput, 'new-post-slug')

      expect(slugInput).toHaveValue('new-post-slug')
    })

    it('should allow entering tags', async () => {
      const user = userEvent.setup()
      await renderCreateMode()

      // TagInputコンポーネントのinputを取得
      const tagsInput = screen.getByLabelText('Tag input')
      // Tagsを入力してカンマで区切ると追加される
      await user.type(tagsInput, 'NewTag,')

      // 追加されたTagsがチップとして表示される
      await waitFor(() => {
        expect(screen.getByText('NewTag')).toBeInTheDocument()
      })
    })

    it('should render markdown editor', async () => {
      await renderCreateMode()
      // MarkdownEditorコンポーネントがレンダリングされていることを確認
      expect(screen.getByText('Content (Markdown)')).toBeInTheDocument()
      // MDEditorはツールバーを表示する
      expect(screen.getByRole('button', { name: /bold/i })).toBeInTheDocument()
    })
  })

  describe('Create post', () => {
    it('should show validation error when both are whitespace', async () => {
      const user = userEvent.setup()
      await renderCreateMode()

      await user.type(screen.getByLabelText(/Title/), '   ')
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

      await user.type(screen.getByLabelText(/Title/), 'New Post')
      await user.type(screen.getByLabelText(/Slug/), 'new-post')
      // TagInputでTagsを追加（カンマで区切って追加）
      await user.type(screen.getByLabelText('Tag input'), 'React,')
      // MarkdownEditorはlabel関連付けが異なるため、content入力はスキップ
      await user.click(screen.getByRole('button', { name: 'Save' }))

      await waitFor(() => {
        expect(apiClient.createPost).toHaveBeenCalledWith({
          title: 'New Post',
          slug: 'new-post',
          content: '',
          tags: 'React',
          is_pinned: false,
        })
      })

      // モーダルが表示されることを確認
      await waitFor(() => {
        expect(screen.getByText('Post created')).toBeInTheDocument()
      })
      // OKをクリックしてモーダルを閉じる
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

      await user.type(screen.getByLabelText(/Title/), 'New Post')
      await user.type(screen.getByLabelText(/Slug/), 'duplicate-slug')
      await user.click(screen.getByRole('button', { name: 'Save' }))

      await waitFor(() => {
        expect(screen.getByText('Slug already exists')).toBeInTheDocument()
      })

      // エラー時はモーダルが表示されない
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
        // MarkdownEditorのdisabled状態はコンポーネント内部で管理されるため、
        // ボタンのdisabled状態のみ確認
        expect(saveButton).toBeDisabled()
        expect(screen.getByRole('button', { name: 'Saving...' })).toBeInTheDocument()
      })

      resolveCreate!(mockDraftPost)

      // モーダルが表示されることを確認
      await waitFor(() => {
        expect(screen.getByText('Post created')).toBeInTheDocument()
      })

      // OKをクリックしてモーダルを閉じる
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
        })
      })

      // モーダルが表示されることを確認
      await waitFor(() => {
        expect(screen.getByText('Post updated')).toBeInTheDocument()
      })
      // OKをクリックしてモーダルを閉じる
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

      // エラー時はモーダルが表示されない
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

      // モーダルが表示されることを確認
      await waitFor(() => {
        expect(screen.getByText('Post published')).toBeInTheDocument()
      })
      // OKをクリックしてモーダルを閉じる
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

      // エラー時はモーダルが表示されない
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

      // モーダルが表示されることを確認
      await waitFor(() => {
        expect(screen.getByText('Post unpublished')).toBeInTheDocument()
      })
      // OKをクリックしてモーダルを閉じる
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

      // エラー時はモーダルが表示されない
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

      // 確認ダイアログが表示されることを確認
      await waitFor(() => {
        expect(screen.getByText('Are you sure you want to delete this post? This action cannot be undone.')).toBeInTheDocument()
        expect(screen.getByText('Delete Post')).toBeInTheDocument()
      })

      // Cancelをクリック
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

      // 確認ダイアログが表示されることを確認
      await waitFor(() => {
        expect(screen.getByText('Are you sure you want to delete this post? This action cannot be undone.')).toBeInTheDocument()
      })

      // Deleteボタン（確認ダイアログ内）をクリック
      const dialogDeleteButton = screen.getByRole('dialog').querySelector('button.bg-red-600')
      await user.click(dialogDeleteButton!)

      await waitFor(() => {
        expect(apiClient.deletePost).toHaveBeenCalledWith(1)
      })

      // Delete完了モーダルが表示される
      await waitFor(() => {
        expect(screen.getByText('Post deleted')).toBeInTheDocument()
      })
      // OKをクリックしてモーダルを閉じる
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

      // 確認ダイアログが表示されることを確認
      await waitFor(() => {
        expect(screen.getByText('Are you sure you want to delete this post? This action cannot be undone.')).toBeInTheDocument()
      })

      // Deleteボタン（確認ダイアログ内）をクリック
      const dialogDeleteButton = screen.getByRole('dialog').querySelector('button.bg-red-600')
      await user.click(dialogDeleteButton!)

      await waitFor(() => {
        expect(screen.getByText('Failed to delete')).toBeInTheDocument()
      })

      expect(mockNavigate).not.toHaveBeenCalled()
    })
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
})
