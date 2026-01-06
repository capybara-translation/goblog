import { useState, useEffect, FormEvent } from 'react';
import { useParams, useNavigate, Link } from 'react-router-dom';
import { apiClient, Post } from '../api/client';
import { StatusBadge } from '../components/StatusBadge';

export function PostEdit() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const isEditMode = Boolean(id);

  const [post, setPost] = useState<Post | null>(null);
  const [title, setTitle] = useState('');
  const [slug, setSlug] = useState('');
  const [content, setContent] = useState('');
  const [tags, setTags] = useState('');
  const [error, setError] = useState('');
  const [isLoading, setIsLoading] = useState(false);
  const [isSaving, setIsSaving] = useState(false);

  useEffect(() => {
    if (isEditMode && id) {
      loadPost(Number(id));
    }
  }, [id, isEditMode]);

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
    } catch (err) {
      setError(err instanceof Error ? err.message : '記事の取得に失敗しました');
    } finally {
      setIsLoading(false);
    }
  };

  const handleSave = async (e: FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    setError('');

    if (!title.trim() || !slug.trim()) {
      setError('タイトルとスラッグは必須です');
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
        });
        setPost(updated);
        alert('記事を更新しました');
      } else {
        const created = await apiClient.createPost({
          title,
          slug,
          content,
          tags,
        });
        alert('記事を作成しました');
        navigate(`/posts/${created.id}/edit`);
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : '保存に失敗しました');
    } finally {
      setIsSaving(false);
    }
  };

  const handlePublish = async () => {
    if (!post || !id) return;

    try {
      setError('');
      const updated = await apiClient.publishPost(Number(id));
      setPost(updated);
      alert('記事を公開しました');
    } catch (err) {
      setError(err instanceof Error ? err.message : '公開に失敗しました');
    }
  };

  const handleUnpublish = async () => {
    if (!post || !id) return;

    try {
      setError('');
      const updated = await apiClient.unpublishPost(Number(id));
      setPost(updated);
      alert('記事を非公開にしました');
    } catch (err) {
      setError(err instanceof Error ? err.message : '非公開化に失敗しました');
    }
  };

  const handleDelete = async () => {
    if (!post || !id) return;

    if (!confirm('本当に削除しますか？この操作は取り消せません。')) {
      return;
    }

    try {
      setError('');
      await apiClient.deletePost(Number(id));
      alert('記事を削除しました');
      navigate('/posts');
    } catch (err) {
      setError(err instanceof Error ? err.message : '削除に失敗しました');
    }
  };

  if (isLoading) {
    return (
      <div className="flex justify-center items-center min-h-64">
        <div className="text-primary-600">読み込み中...</div>
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
            ← 記事一覧に戻る
          </Link>
          <h1 className="text-2xl font-sans font-bold text-primary-900">
            {isEditMode ? '記事編集' : '新規作成'}
          </h1>
        </div>
        {post && (
          <div className="flex items-center gap-4">
            <StatusBadge status={post.status} />
            {post.status === 'draft' ? (
              <button
                onClick={handlePublish}
                className="bg-green-600 text-white px-4 py-2 rounded-md hover:bg-green-700 transition-colors text-sm font-medium"
              >
                公開する
              </button>
            ) : (
              <button
                onClick={handleUnpublish}
                className="bg-yellow-600 text-white px-4 py-2 rounded-md hover:bg-yellow-700 transition-colors text-sm font-medium"
              >
                非公開にする
              </button>
            )}
            <button
              onClick={handleDelete}
              className="bg-red-600 text-white px-4 py-2 rounded-md hover:bg-red-700 transition-colors text-sm font-medium"
            >
              削除
            </button>
          </div>
        )}
      </div>

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
            タイトル <span className="text-red-600">*</span>
          </label>
          <input
            id="title"
            type="text"
            value={title}
            onChange={(e) => setTitle(e.target.value)}
            required
            className="w-full px-3 py-2 border border-primary-300 rounded-md focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
            disabled={isSaving}
          />
        </div>

        <div>
          <label
            htmlFor="slug"
            className="block text-sm font-medium text-primary-700 mb-1"
          >
            スラッグ <span className="text-red-600">*</span>
          </label>
          <input
            id="slug"
            type="text"
            value={slug}
            onChange={(e) => setSlug(e.target.value)}
            required
            className="w-full px-3 py-2 border border-primary-300 rounded-md focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
            disabled={isSaving}
            placeholder="my-first-post"
          />
          <p className="mt-1 text-sm text-primary-500">
            URLに使用されます（例: /posts/my-first-post）
          </p>
        </div>

        <div>
          <label
            htmlFor="tags"
            className="block text-sm font-medium text-primary-700 mb-1"
          >
            タグ
          </label>
          <input
            id="tags"
            type="text"
            value={tags}
            onChange={(e) => setTags(e.target.value)}
            className="w-full px-3 py-2 border border-primary-300 rounded-md focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
            disabled={isSaving}
            placeholder="Go, Web, Tutorial"
          />
          <p className="mt-1 text-sm text-primary-500">
            カンマ区切りで複数指定できます
          </p>
        </div>

        <div>
          <label
            htmlFor="content"
            className="block text-sm font-medium text-primary-700 mb-1"
          >
            本文（Markdown）
          </label>
          <textarea
            id="content"
            value={content}
            onChange={(e) => setContent(e.target.value)}
            rows={20}
            className="w-full px-3 py-2 border border-primary-300 rounded-md focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-primary-500 font-mono text-sm"
            disabled={isSaving}
            placeholder="# 見出し

本文をここに書きます..."
          />
        </div>

        <div className="flex justify-end">
          <button
            type="submit"
            disabled={isSaving}
            className="bg-primary-900 text-white px-6 py-2 rounded-md hover:bg-primary-800 focus:outline-none focus:ring-2 focus:ring-primary-500 focus:ring-offset-2 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
          >
            {isSaving ? '保存中...' : '保存'}
          </button>
        </div>
      </form>
    </div>
  );
}
