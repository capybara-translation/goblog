import { useState, useEffect, useCallback, FormEvent } from 'react';
import { useParams, useNavigate, Link } from 'react-router-dom';
import { formatInTimeZone } from 'date-fns-tz';
import { apiClient, Post } from '../api/client';
import { StatusBadge } from '../components/StatusBadge';
import { MarkdownEditor } from '../components/MarkdownEditor';
import { TagInput } from '../components/TagInput';
import { useModal } from '../hooks/useModal';

const BLOG_TIMEZONE = import.meta.env.VITE_BLOG_TIMEZONE || 'UTC';

/**
 * Format date in ISO 8601 format
 * Example: "2024-01-15 10:30:05"
 */
function formatDateTime(dateString: string): string {
  return formatInTimeZone(new Date(dateString), BLOG_TIMEZONE, 'yyyy-MM-dd HH:mm:ss');
}

export function PostEdit() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const { showAlert, confirm } = useModal();
  const isEditMode = Boolean(id);

  const [post, setPost] = useState<Post | null>(null);
  const [defaultTitle, defaultSlug] = (() => {
    if (!isEditMode) {
      const now = new Date();
      return [
        formatInTimeZone(now, BLOG_TIMEZONE, 'yyyy.MM.dd'),
        formatInTimeZone(now, BLOG_TIMEZONE, 'yyyy-MM-dd'),
      ];
    }
    return ['', ''];
  })();
  const [title, setTitle] = useState(defaultTitle);
  const [slug, setSlug] = useState(defaultSlug);
  const [content, setContent] = useState('');
  const [tags, setTags] = useState('');
  const [isPinned, setIsPinned] = useState(false);
  const [error, setError] = useState('');
  const [isLoading, setIsLoading] = useState(false);
  const [isSaving, setIsSaving] = useState(false);

  useEffect(() => {
    if (isEditMode && id) {
      loadPost(Number(id));
    }
  }, [id, isEditMode]);

  // Save handler (shared between form submit and shortcut key)
  const performSave = useCallback(async () => {
    if (isSaving) return;

    setError('');

    if (!title.trim() || !slug.trim()) {
      setError('Title and slug are required');
      return;
    }

    try {
      setIsSaving(true);

      if (isEditMode && id) {
        const updated = await apiClient.updatePost(Number(id), {
          title,
          slug,
          content,
          tags,
          is_pinned: isPinned,
        });
        setPost(updated);
        await showAlert('Post updated');
      } else {
        const created = await apiClient.createPost({
          title,
          slug,
          content,
          tags,
          is_pinned: isPinned,
        });
        await showAlert('Post created');
        navigate(`/posts/${created.id}/edit`);
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to save');
    } finally {
      setIsSaving(false);
    }
  }, [title, slug, content, tags, isPinned, isSaving, isEditMode, id, navigate, showAlert]);

  const loadPost = async (postId: number) => {
    try {
      setIsLoading(true);
      setError('');
      const data = await apiClient.getPost(postId);
      setPost(data);
      setTitle(data.title);
      setSlug(data.slug);
      setContent(data.content);
      setTags(data.tags);
      setIsPinned(data.is_pinned);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load post');
    } finally {
      setIsLoading(false);
    }
  };

  const handleSave = (e: FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    performSave();
  };

  const handlePublish = async () => {
    if (!post || !id) return;

    try {
      setError('');
      const updated = await apiClient.publishPost(Number(id));
      setPost(updated);
      await showAlert('Post published');
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to publish');
    }
  };

  const handleUnpublish = async () => {
    if (!post || !id) return;

    try {
      setError('');
      const updated = await apiClient.unpublishPost(Number(id));
      setPost(updated);
      await showAlert('Post unpublished');
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to unpublish');
    }
  };

  const handleDelete = async () => {
    if (!post || !id) return;

    const confirmed = await confirm('Are you sure you want to delete this post? This action cannot be undone.', {
      title: 'Delete Post',
      confirmText: 'Delete',
      cancelText: 'Cancel',
      danger: true,
    });

    if (!confirmed) return;

    try {
      setError('');
      await apiClient.deletePost(Number(id));
      await showAlert('Post deleted');
      navigate('/posts');
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to delete');
    }
  };

  if (isLoading) {
    return (
      <div className="flex justify-center items-center min-h-64">
        <div className="text-primary-600">Loading...</div>
      </div>
    );
  }

  return (
    <div className="max-w-4xl mx-auto space-y-6">
      {/* Header */}
      <div className="flex justify-between items-center">
        <div>
          <Link
            to="/posts"
            className="text-sm text-primary-600 hover:text-primary-800 mb-2 inline-block"
          >
            ← Back to posts
          </Link>
          <h1 className="text-2xl font-sans font-bold text-primary-900">
            {isEditMode ? 'Edit Post' : 'New Post'}
          </h1>
        </div>
        {post && (
          <div className="flex items-center gap-4">
            {post.status === 'published' ? (
              <a
                href={`/posts/${post.slug}`}
                target="_blank"
                rel="noopener noreferrer"
                title="Open public page in a new tab"
                className="text-sm text-primary-600 hover:text-primary-800 underline"
              >
                View ↗
              </a>
            ) : (
              <span
                aria-disabled="true"
                title="Draft posts aren't publicly visible. Publish first to view."
                className="text-sm text-primary-400 cursor-not-allowed select-none"
              >
                View ↗
              </span>
            )}
            <StatusBadge status={post.status} />
            {post.status === 'draft' ? (
              <button
                onClick={handlePublish}
                className="bg-green-600 text-white px-4 py-2 rounded-md hover:bg-green-700 transition-colors text-sm font-medium"
              >
                Publish
              </button>
            ) : (
              <button
                onClick={handleUnpublish}
                className="bg-yellow-600 text-white px-4 py-2 rounded-md hover:bg-yellow-700 transition-colors text-sm font-medium"
              >
                Unpublish
              </button>
            )}
            <button
              onClick={handleDelete}
              className="bg-red-600 text-white px-4 py-2 rounded-md hover:bg-red-700 transition-colors text-sm font-medium"
            >
              Delete
            </button>
          </div>
        )}
      </div>

      {/* Post metadata */}
      {post && (
        <div className="bg-primary-50 border border-primary-200 rounded-lg px-4 py-3">
          <div className="flex flex-wrap gap-x-6 gap-y-1 text-sm text-primary-600">
            <div>
              <span className="font-medium">Created:</span>{' '}
              {formatDateTime(post.created_at)}
            </div>
            <div>
              <span className="font-medium">Updated:</span>{' '}
              {formatDateTime(post.updated_at)}
            </div>
            {post.published_at && (
              <div>
                <span className="font-medium">Published:</span>{' '}
                {formatDateTime(post.published_at)}
              </div>
            )}
            <div>
              <span className="font-medium">Views:</span>{' '}
              {post.view_count.toLocaleString()}
            </div>
          </div>
        </div>
      )}

      {/* Error message */}
      {error && (
        <div className="bg-red-50 border border-red-200 text-red-700 px-4 py-3 rounded">
          {error}
        </div>
      )}

      {/* Form */}
      <form onSubmit={handleSave} className="bg-white rounded-lg shadow-sm p-6 space-y-6">
        <div>
          <label
            htmlFor="title"
            className="block text-sm font-medium text-primary-700 mb-1"
          >
            Title <span className="text-red-600">*</span>
          </label>
          <input
            id="title"
            type="text"
            value={title}
            onChange={(e) => setTitle(e.target.value)}
            required
            autoCapitalize="none"
            className="w-full px-3 py-2 border border-primary-300 rounded-md focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
            disabled={isSaving}
          />
        </div>

        <div>
          <label
            htmlFor="slug"
            className="block text-sm font-medium text-primary-700 mb-1"
          >
            Slug <span className="text-red-600">*</span>
          </label>
          <input
            id="slug"
            type="text"
            value={slug}
            onChange={(e) => setSlug(e.target.value)}
            required
            autoCapitalize="none"
            className="w-full px-3 py-2 border border-primary-300 rounded-md focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
            disabled={isSaving}
            placeholder="my-first-post"
          />
          <p className="mt-1 text-sm text-primary-500">
            Used in URL (e.g., /posts/my-first-post)
          </p>
        </div>

        <div>
          <label className="block text-sm font-medium text-primary-700 mb-1">
            Tags
          </label>
          <TagInput
            value={tags}
            onChange={setTags}
            disabled={isSaving}
            placeholder="Enter tags..."
          />
        </div>

        <div className="flex items-center gap-2">
          <input
            type="checkbox"
            id="is_pinned"
            checked={isPinned}
            onChange={(e) => setIsPinned(e.target.checked)}
            disabled={isSaving}
            className="h-4 w-4 text-primary-600 focus:ring-primary-500 border-primary-300 rounded"
          />
          <label
            htmlFor="is_pinned"
            className="text-sm text-primary-700"
            title="Pinned posts appear in the header"
          >
            Pinned
          </label>
        </div>

        <div>
          <label
            htmlFor="content"
            className="block text-sm font-medium text-primary-700 mb-1"
          >
            Content (Markdown)
          </label>
          <MarkdownEditor
            value={content}
            onChange={setContent}
            disabled={isSaving}
            onSave={performSave}
          />
        </div>

        <div className="flex justify-between items-center">
          <span className="text-xs text-primary-400">
            Icons by <a href="https://icons8.jp/" target="_blank" rel="noopener noreferrer" className="underline hover:text-primary-600">Icons8</a>
          </span>
          <button
            type="submit"
            disabled={isSaving}
            className="bg-primary-900 text-white px-6 py-2 rounded-md hover:bg-primary-800 focus:outline-none focus:ring-2 focus:ring-primary-500 focus:ring-offset-2 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
          >
            {isSaving ? 'Saving...' : 'Save'}
          </button>
        </div>
      </form>
    </div>
  );
}
