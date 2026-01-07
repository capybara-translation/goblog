import { useState, useEffect, useMemo } from 'react';
import { Link } from 'react-router-dom';
import { apiClient, Post } from '../api/client';
import { StatusBadge } from '../components/StatusBadge';
import { TagList } from '../components/TagList';

const POSTS_PER_PAGE = 20;
const BLOG_TIMEZONE = import.meta.env.VITE_BLOG_TIMEZONE || 'UTC';

export function PostList() {
  const [posts, setPosts] = useState<Post[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState('');
  const [statusFilter, setStatusFilter] = useState<'all' | 'draft' | 'published'>('all');
  const [searchQuery, setSearchQuery] = useState('');
  const [currentPage, setCurrentPage] = useState(1);

  useEffect(() => {
    loadPosts();
  }, []);

  const loadPosts = async () => {
    try {
      setIsLoading(true);
      setError('');
      const data = await apiClient.getPosts({ status: undefined });
      setPosts(data);
    } catch (err) {
      setError(err instanceof Error ? err.message : '記事の取得に失敗しました');
    } finally {
      setIsLoading(false);
    }
  };

  // Filter and search posts
  const filteredPosts = useMemo(() => {
    let filtered = [...posts];

    // Filter by status
    if (statusFilter !== 'all') {
      filtered = filtered.filter((post) => post.status === statusFilter);
    }

    // Search by title
    if (searchQuery.trim()) {
      const query = searchQuery.toLowerCase();
      filtered = filtered.filter((post) =>
        post.title.toLowerCase().includes(query)
      );
    }

    return filtered;
  }, [posts, statusFilter, searchQuery]);

  // Paginate posts
  const paginatedPosts = useMemo(() => {
    const startIndex = (currentPage - 1) * POSTS_PER_PAGE;
    const endIndex = startIndex + POSTS_PER_PAGE;
    return filteredPosts.slice(startIndex, endIndex);
  }, [filteredPosts, currentPage]);

  const totalPages = Math.ceil(filteredPosts.length / POSTS_PER_PAGE);

  // Reset to page 1 when filters change
  useEffect(() => {
    setCurrentPage(1);
  }, [statusFilter, searchQuery]);

  const formatDate = (dateString: string | null) => {
    if (!dateString) return '-';
    const date = new Date(dateString);

    // 日付フォーマット
    const formatted = date.toLocaleDateString(undefined, {
      year: 'numeric',
      month: '2-digit',
      day: '2-digit',
      timeZone: BLOG_TIMEZONE,
    });

    // タイムゾーン略称を取得（JST, EST など）
    const timeZoneName = new Intl.DateTimeFormat(undefined, {
      timeZone: BLOG_TIMEZONE,
      timeZoneName: 'short',
    }).formatToParts(date).find(part => part.type === 'timeZoneName')?.value;

    return timeZoneName ? `${formatted} (${timeZoneName})` : formatted;
  };

  if (isLoading) {
    return (
      <div className="flex justify-center items-center min-h-64">
        <div className="text-primary-600">読み込み中...</div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="bg-red-50 border border-red-200 text-red-700 px-4 py-3 rounded">
        {error}
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex justify-between items-center">
        <h1 className="text-2xl font-sans font-bold text-primary-900">
          記事一覧
        </h1>
        <Link
          to="/posts/new"
          className="bg-primary-900 text-white px-4 py-2 rounded-md hover:bg-primary-800 transition-colors text-sm font-medium"
        >
          新規作成
        </Link>
      </div>

      {/* Filters */}
      <div className="bg-white rounded-lg shadow-sm p-4 space-y-4">
        <div className="flex flex-col sm:flex-row gap-4">
          {/* Status filter */}
          <div className="flex-1">
            <label
              htmlFor="status-filter"
              className="block text-sm font-medium text-primary-700 mb-1"
            >
              ステータス
            </label>
            <select
              id="status-filter"
              value={statusFilter}
              onChange={(e) =>
                setStatusFilter(e.target.value as 'all' | 'draft' | 'published')
              }
              className="w-full px-3 py-2 border border-primary-300 rounded-md focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
            >
              <option value="all">全て</option>
              <option value="draft">下書き</option>
              <option value="published">公開済み</option>
            </select>
          </div>

          {/* Search */}
          <div className="flex-1">
            <label
              htmlFor="search"
              className="block text-sm font-medium text-primary-700 mb-1"
            >
              タイトル検索
            </label>
            <input
              id="search"
              type="text"
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              placeholder="記事のタイトルで検索..."
              className="w-full px-3 py-2 border border-primary-300 rounded-md focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
            />
          </div>
        </div>

        <div className="text-sm text-primary-600">
          {filteredPosts.length}件の記事
        </div>
      </div>

      {/* Posts table */}
      {paginatedPosts.length === 0 ? (
        <div className="bg-white rounded-lg shadow-sm p-8 text-center text-primary-500">
          記事が見つかりませんでした
        </div>
      ) : (
        <div className="bg-white rounded-lg shadow-sm overflow-hidden">
          <table className="min-w-full divide-y divide-primary-200">
            <thead className="bg-primary-50">
              <tr>
                <th className="px-6 py-3 text-left text-xs font-medium text-primary-700 uppercase tracking-wider">
                  タイトル
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-primary-700 uppercase tracking-wider">
                  ステータス
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-primary-700 uppercase tracking-wider">
                  タグ
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-primary-700 uppercase tracking-wider">
                  更新日
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-primary-700 uppercase tracking-wider">
                  公開日
                </th>
              </tr>
            </thead>
            <tbody className="bg-white divide-y divide-primary-200">
              {paginatedPosts.map((post) => (
                <tr
                  key={post.id}
                  className="hover:bg-primary-50 transition-colors"
                >
                  <td className="px-6 py-4">
                    <Link
                      to={`/posts/${post.id}/edit`}
                      className="text-primary-900 hover:text-primary-700 font-medium"
                    >
                      {post.title}
                    </Link>
                  </td>
                  <td className="px-6 py-4">
                    <StatusBadge status={post.status} />
                  </td>
                  <td className="px-6 py-4">
                    <TagList tags={post.tags} />
                  </td>
                  <td className="px-6 py-4 text-sm text-primary-600">
                    {formatDate(post.updated_at)}
                  </td>
                  <td className="px-6 py-4 text-sm text-primary-600">
                    {formatDate(post.published_at)}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {/* Pagination */}
      {totalPages > 1 && (
        <div className="flex justify-center gap-2">
          <button
            onClick={() => setCurrentPage((p) => Math.max(1, p - 1))}
            disabled={currentPage === 1}
            className="px-4 py-2 border border-primary-300 rounded-md text-sm font-medium text-primary-700 hover:bg-primary-50 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
          >
            前へ
          </button>
          <span className="px-4 py-2 text-sm text-primary-600">
            {currentPage} / {totalPages}
          </span>
          <button
            onClick={() => setCurrentPage((p) => Math.min(totalPages, p + 1))}
            disabled={currentPage === totalPages}
            className="px-4 py-2 border border-primary-300 rounded-md text-sm font-medium text-primary-700 hover:bg-primary-50 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
          >
            次へ
          </button>
        </div>
      )}
    </div>
  );
}
