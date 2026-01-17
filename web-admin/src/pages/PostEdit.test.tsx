import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter, Routes, Route } from 'react-router-dom'
import { PostEdit } from './PostEdit'
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

// window.alert と window.confirm をモック
const mockAlert = vi.fn()
const mockConfirm = vi.fn()
global.alert = mockAlert
global.confirm = mockConfirm

describe('PostEdit', () => {
  const mockDraftPost: Post = {
    id: 1,
    title: 'Draft Post',
    slug: 'draft-post',
    content: 'Draft content',
    status: 'draft',
    tags: 'Go, Testing',
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
        <Routes>
          <Route path="/posts/new" element={<PostEdit />} />
          <Route path="/posts/:id/edit" element={<div>Edit Page</div>} />
        </Routes>
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
        <Routes>
          <Route path="/posts/:id/edit" element={<PostEdit />} />
          <Route path="/posts" element={<div>Post List</div>} />
        </Routes>
      </MemoryRouter>
    )
    // TagInput の getTags 呼び出し完了を待つ
    await waitFor(() => {
      expect(apiClient.getTags).toHaveBeenCalled()
    })
    return result
  }

  describe('Create mode rendering', () => {
    it('should render page title as "新規作成"', async () => {
      await renderCreateMode()
      expect(screen.getByText('新規作成')).toBeInTheDocument()
    })

    it('should render empty form fields', async () => {
      await renderCreateMode()
      expect(screen.getByLabelText(/タイトル/)).toHaveValue('')
      expect(screen.getByLabelText(/スラッグ/)).toHaveValue('')
      // TagInputコンポーネントはlabel関連付けが異なるため、ラベル表示のみ確認
      expect(screen.getByText('タグ')).toBeInTheDocument()
      // MarkdownEditor コンポーネントはlabel関連付けが異なるため、ラベル表示のみ確認
      expect(screen.getByText('本文（Markdown）')).toBeInTheDocument()
    })

    it('should render save button', async () => {
      await renderCreateMode()
      expect(screen.getByRole('button', { name: '保存' })).toBeInTheDocument()
    })

    it('should not render publish/unpublish/delete buttons', async () => {
      await renderCreateMode()
      expect(screen.queryByRole('button', { name: '公開する' })).not.toBeInTheDocument()
      expect(screen.queryByRole('button', { name: '非公開にする' })).not.toBeInTheDocument()
      expect(screen.queryByRole('button', { name: '削除' })).not.toBeInTheDocument()
    })

    it('should render back link', async () => {
      await renderCreateMode()
      expect(screen.getByText('← 記事一覧に戻る')).toBeInTheDocument()
    })

    it('should have required attributes on title and slug fields', async () => {
      await renderCreateMode()
      expect(screen.getByLabelText(/タイトル/)).toBeRequired()
      expect(screen.getByLabelText(/スラッグ/)).toBeRequired()
    })
  })

  describe('Edit mode rendering', () => {
    it('should show loading state initially', async () => {
      vi.mocked(apiClient.getPost).mockReturnValue(new Promise(() => {}))
      await renderEditMode(1)
      expect(screen.getByText('読み込み中...')).toBeInTheDocument()
    })

    it('should render page title as "記事編集"', async () => {
      vi.mocked(apiClient.getPost).mockResolvedValue(mockDraftPost)
      await renderEditMode(1)

      await waitFor(() => {
        expect(screen.getByText('記事編集')).toBeInTheDocument()
      })
    })

    it('should load and display post data', async () => {
      vi.mocked(apiClient.getPost).mockResolvedValue(mockDraftPost)
      await renderEditMode(1)

      await waitFor(() => {
        expect(screen.getByLabelText(/タイトル/)).toHaveValue('Draft Post')
      })

      expect(screen.getByLabelText(/スラッグ/)).toHaveValue('draft-post')
      // TagInputコンポーネントはタグをチップで表示
      expect(screen.getByText('Go')).toBeInTheDocument()
      expect(screen.getByText('Testing')).toBeInTheDocument()
      // MarkdownEditorのtextareaは直接取得できないため、APIコールを確認
      expect(apiClient.getPost).toHaveBeenCalledWith(1)
    })

    it('should show error message when post load fails', async () => {
      vi.mocked(apiClient.getPost).mockRejectedValue(new Error('記事が見つかりません'))
      await renderEditMode(1)

      await waitFor(() => {
        expect(screen.getByText('記事が見つかりません')).toBeInTheDocument()
      })
    })

    it('should render publish button for draft post', async () => {
      vi.mocked(apiClient.getPost).mockResolvedValue(mockDraftPost)
      await renderEditMode(1)

      await waitFor(() => {
        expect(screen.getByRole('button', { name: '公開する' })).toBeInTheDocument()
      })

      expect(screen.queryByRole('button', { name: '非公開にする' })).not.toBeInTheDocument()
    })

    it('should render unpublish button for published post', async () => {
      vi.mocked(apiClient.getPost).mockResolvedValue(mockPublishedPost)
      await renderEditMode(2)

      await waitFor(() => {
        expect(screen.getByRole('button', { name: '非公開にする' })).toBeInTheDocument()
      })

      expect(screen.queryByRole('button', { name: '公開する' })).not.toBeInTheDocument()
    })

    it('should render delete button', async () => {
      vi.mocked(apiClient.getPost).mockResolvedValue(mockDraftPost)
      await renderEditMode(1)

      await waitFor(() => {
        expect(screen.getByRole('button', { name: '削除' })).toBeInTheDocument()
      })
    })

    it('should render status badge', async () => {
      vi.mocked(apiClient.getPost).mockResolvedValue(mockDraftPost)
      await renderEditMode(1)

      await waitFor(() => {
        expect(screen.getByText('下書き')).toBeInTheDocument()
      })
    })
  })

  describe('Post metadata display', () => {
    it('should display created_at and updated_at for draft post', async () => {
      vi.mocked(apiClient.getPost).mockResolvedValue(mockDraftPost)
      await renderEditMode(1)

      await waitFor(() => {
        expect(screen.getByText('作成日:')).toBeInTheDocument()
      })

      expect(screen.getByText('更新日:')).toBeInTheDocument()
      // published_at が null の場合は公開日を表示しない
      expect(screen.queryByText('公開日:')).not.toBeInTheDocument()
    })

    it('should display published_at for published post', async () => {
      vi.mocked(apiClient.getPost).mockResolvedValue(mockPublishedPost)
      await renderEditMode(2)

      await waitFor(() => {
        expect(screen.getByText('公開日:')).toBeInTheDocument()
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
        expect(screen.getByText('作成日:')).toBeInTheDocument()
      })

      // ISO 8601形式（yyyy-MM-dd HH:mm:ss）で表示されることを確認
      const datePattern = /\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}/
      const metadataSection = screen.getByText('作成日:').closest('div')?.parentElement
      expect(metadataSection?.textContent).toMatch(datePattern)
    })

    it('should not display metadata in create mode', async () => {
      await renderCreateMode()

      expect(screen.queryByText('作成日:')).not.toBeInTheDocument()
      expect(screen.queryByText('更新日:')).not.toBeInTheDocument()
      expect(screen.queryByText('公開日:')).not.toBeInTheDocument()
    })
  })

  describe('Form input', () => {
    it('should allow entering title', async () => {
      const user = userEvent.setup()
      await renderCreateMode()

      const titleInput = screen.getByLabelText(/タイトル/)
      await user.type(titleInput, 'New Post Title')

      expect(titleInput).toHaveValue('New Post Title')
    })

    it('should allow entering slug', async () => {
      const user = userEvent.setup()
      await renderCreateMode()

      const slugInput = screen.getByLabelText(/スラッグ/)
      await user.type(slugInput, 'new-post-slug')

      expect(slugInput).toHaveValue('new-post-slug')
    })

    it('should allow entering tags', async () => {
      const user = userEvent.setup()
      await renderCreateMode()

      // TagInputコンポーネントのinputを取得
      const tagsInput = screen.getByLabelText('タグ入力')
      // タグを入力してカンマで区切ると追加される
      await user.type(tagsInput, 'NewTag,')

      // 追加されたタグがチップとして表示される
      await waitFor(() => {
        expect(screen.getByText('NewTag')).toBeInTheDocument()
      })
    })

    it('should render markdown editor', async () => {
      await renderCreateMode()
      // MarkdownEditorコンポーネントがレンダリングされていることを確認
      expect(screen.getByText('本文（Markdown）')).toBeInTheDocument()
      // MDEditorはツールバーを表示する
      expect(screen.getByRole('button', { name: /bold/i })).toBeInTheDocument()
    })
  })

  describe('Create post', () => {
    it('should show validation error when both are whitespace', async () => {
      const user = userEvent.setup()
      await renderCreateMode()

      await user.type(screen.getByLabelText(/タイトル/), '   ')
      await user.type(screen.getByLabelText(/スラッグ/), '   ')
      await user.click(screen.getByRole('button', { name: '保存' }))

      await waitFor(() => {
        expect(screen.getByText('タイトルとスラッグは必須です')).toBeInTheDocument()
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

      await user.type(screen.getByLabelText(/タイトル/), 'New Post')
      await user.type(screen.getByLabelText(/スラッグ/), 'new-post')
      // TagInputでタグを追加（カンマで区切って追加）
      await user.type(screen.getByLabelText('タグ入力'), 'React,')
      // MarkdownEditorはlabel関連付けが異なるため、content入力はスキップ
      await user.click(screen.getByRole('button', { name: '保存' }))

      await waitFor(() => {
        expect(apiClient.createPost).toHaveBeenCalledWith({
          title: 'New Post',
          slug: 'new-post',
          content: '',
          tags: 'React',
        })
      })

      expect(mockAlert).toHaveBeenCalledWith('記事を作成しました')
      expect(mockNavigate).toHaveBeenCalledWith('/posts/999/edit')
    })

    it('should show error message when create fails', async () => {
      const user = userEvent.setup()
      vi.mocked(apiClient.createPost).mockRejectedValue(
        new Error('スラッグが既に存在します')
      )

      await renderCreateMode()

      await user.type(screen.getByLabelText(/タイトル/), 'New Post')
      await user.type(screen.getByLabelText(/スラッグ/), 'duplicate-slug')
      await user.click(screen.getByRole('button', { name: '保存' }))

      await waitFor(() => {
        expect(screen.getByText('スラッグが既に存在します')).toBeInTheDocument()
      })

      expect(mockAlert).not.toHaveBeenCalled()
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

      const titleInput = screen.getByLabelText(/タイトル/)
      const slugInput = screen.getByLabelText(/スラッグ/)
      const tagsInput = screen.getByLabelText('タグ入力')
      const saveButton = screen.getByRole('button', { name: '保存' })

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
        expect(screen.getByRole('button', { name: '保存中...' })).toBeInTheDocument()
      })

      resolveCreate!(mockDraftPost)

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
        expect(screen.getByLabelText(/タイトル/)).toHaveValue('Draft Post')
      })

      const titleInput = screen.getByLabelText(/タイトル/)
      await user.clear(titleInput)
      await user.type(titleInput, 'Updated Title')
      await user.click(screen.getByRole('button', { name: '保存' }))

      await waitFor(() => {
        expect(apiClient.updatePost).toHaveBeenCalledWith(1, {
          title: 'Updated Title',
          slug: 'draft-post',
          content: 'Draft content',
          tags: 'Go, Testing',
        })
      })

      expect(mockAlert).toHaveBeenCalledWith('記事を更新しました')
      expect(mockNavigate).not.toHaveBeenCalled()
    })

    it('should show error message when update fails', async () => {
      const user = userEvent.setup()
      vi.mocked(apiClient.getPost).mockResolvedValue(mockDraftPost)
      vi.mocked(apiClient.updatePost).mockRejectedValue(
        new Error('更新に失敗しました')
      )

      await renderEditMode(1)

      await waitFor(() => {
        expect(screen.getByLabelText(/タイトル/)).toHaveValue('Draft Post')
      })

      await user.click(screen.getByRole('button', { name: '保存' }))

      await waitFor(() => {
        expect(screen.getByText('更新に失敗しました')).toBeInTheDocument()
      })

      expect(mockAlert).not.toHaveBeenCalled()
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
        expect(screen.getByRole('button', { name: '公開する' })).toBeInTheDocument()
      })

      await user.click(screen.getByRole('button', { name: '公開する' }))

      await waitFor(() => {
        expect(apiClient.publishPost).toHaveBeenCalledWith(1)
      })

      expect(mockAlert).toHaveBeenCalledWith('記事を公開しました')
    })

    it('should show error message when publish fails', async () => {
      const user = userEvent.setup()
      vi.mocked(apiClient.getPost).mockResolvedValue(mockDraftPost)
      vi.mocked(apiClient.publishPost).mockRejectedValue(
        new Error('公開に失敗しました')
      )

      await renderEditMode(1)

      await waitFor(() => {
        expect(screen.getByRole('button', { name: '公開する' })).toBeInTheDocument()
      })

      await user.click(screen.getByRole('button', { name: '公開する' }))

      await waitFor(() => {
        expect(screen.getByText('公開に失敗しました')).toBeInTheDocument()
      })

      expect(mockAlert).not.toHaveBeenCalled()
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
        expect(screen.getByText('下書き')).toBeInTheDocument()
      })

      await user.click(screen.getByRole('button', { name: '公開する' }))

      await waitFor(() => {
        expect(screen.getByText('公開済み')).toBeInTheDocument()
      })

      expect(screen.queryByText('下書き')).not.toBeInTheDocument()
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
        expect(screen.getByRole('button', { name: '非公開にする' })).toBeInTheDocument()
      })

      await user.click(screen.getByRole('button', { name: '非公開にする' }))

      await waitFor(() => {
        expect(apiClient.unpublishPost).toHaveBeenCalledWith(2)
      })

      expect(mockAlert).toHaveBeenCalledWith('記事を非公開にしました')
    })

    it('should show error message when unpublish fails', async () => {
      const user = userEvent.setup()
      vi.mocked(apiClient.getPost).mockResolvedValue(mockPublishedPost)
      vi.mocked(apiClient.unpublishPost).mockRejectedValue(
        new Error('非公開化に失敗しました')
      )

      await renderEditMode(2)

      await waitFor(() => {
        expect(screen.getByRole('button', { name: '非公開にする' })).toBeInTheDocument()
      })

      await user.click(screen.getByRole('button', { name: '非公開にする' }))

      await waitFor(() => {
        expect(screen.getByText('非公開化に失敗しました')).toBeInTheDocument()
      })

      expect(mockAlert).not.toHaveBeenCalled()
    })
  })

  describe('Delete post', () => {
    it('should not delete when confirm is cancelled', async () => {
      const user = userEvent.setup()
      vi.mocked(apiClient.getPost).mockResolvedValue(mockDraftPost)
      mockConfirm.mockReturnValue(false)

      await renderEditMode(1)

      await waitFor(() => {
        expect(screen.getByRole('button', { name: '削除' })).toBeInTheDocument()
      })

      await user.click(screen.getByRole('button', { name: '削除' }))

      expect(mockConfirm).toHaveBeenCalledWith(
        '本当に削除しますか？この操作は取り消せません。'
      )
      expect(apiClient.deletePost).not.toHaveBeenCalled()
    })

    it('should delete post when confirm is accepted', async () => {
      const user = userEvent.setup()
      vi.mocked(apiClient.getPost).mockResolvedValue(mockDraftPost)
      vi.mocked(apiClient.deletePost).mockResolvedValue(undefined)
      mockConfirm.mockReturnValue(true)

      await renderEditMode(1)

      await waitFor(() => {
        expect(screen.getByRole('button', { name: '削除' })).toBeInTheDocument()
      })

      await user.click(screen.getByRole('button', { name: '削除' }))

      await waitFor(() => {
        expect(apiClient.deletePost).toHaveBeenCalledWith(1)
      })

      expect(mockAlert).toHaveBeenCalledWith('記事を削除しました')
      expect(mockNavigate).toHaveBeenCalledWith('/posts')
    })

    it('should show error message when delete fails', async () => {
      const user = userEvent.setup()
      vi.mocked(apiClient.getPost).mockResolvedValue(mockDraftPost)
      vi.mocked(apiClient.deletePost).mockRejectedValue(
        new Error('削除に失敗しました')
      )
      mockConfirm.mockReturnValue(true)

      await renderEditMode(1)

      await waitFor(() => {
        expect(screen.getByRole('button', { name: '削除' })).toBeInTheDocument()
      })

      await user.click(screen.getByRole('button', { name: '削除' }))

      await waitFor(() => {
        expect(screen.getByText('削除に失敗しました')).toBeInTheDocument()
      })

      expect(mockAlert).not.toHaveBeenCalled()
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

      await user.type(screen.getByLabelText(/タイトル/), 'Test')
      await user.type(screen.getByLabelText(/スラッグ/), 'test')
      await user.click(screen.getByRole('button', { name: '保存' }))

      await waitFor(() => {
        expect(screen.getByText('First error')).toBeInTheDocument()
      })

      await user.click(screen.getByRole('button', { name: '保存' }))

      await waitFor(() => {
        expect(screen.queryByText('First error')).not.toBeInTheDocument()
      })
    })

    it('should show default error message for non-Error exceptions', async () => {
      const user = userEvent.setup()
      vi.mocked(apiClient.createPost).mockRejectedValue('Unknown error')

      await renderCreateMode()

      await user.type(screen.getByLabelText(/タイトル/), 'Test')
      await user.type(screen.getByLabelText(/スラッグ/), 'test')
      await user.click(screen.getByRole('button', { name: '保存' }))

      await waitFor(() => {
        expect(screen.getByText('保存に失敗しました')).toBeInTheDocument()
      })
    })
  })
})
