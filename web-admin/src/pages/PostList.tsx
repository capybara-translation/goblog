import { useState, useEffect, useCallback } from 'react';
import { Link, useSearchParams } from 'react-router-dom';
import { formatInTimeZone } from 'date-fns-tz';
import { apiClient, Post } from '../api/client';
import { StatusBadge } from '../components/StatusBadge';
import { TagList } from '../components/TagList';
import { ReactionTotalPopover, ReactionBreakdown } from '../components/ReactionSummary';

const POSTS_PER_PAGE = 20;
const BLOG_TIMEZONE = import.meta.env.VITE_BLOG_TIMEZONE || 'UTC';

type StatusFilter = 'all' | 'draft' | 'published';

function parseStatus(raw: string | null): StatusFilter {
  return raw === 'draft' || raw === 'published' ? raw : 'all';
}

function parsePage(raw: string | null): number {
  const n = raw == null ? NaN : parseInt(raw, 10);
  return Number.isFinite(n) && n > 0 ? n : 1;
}

export function PostList() {
  const [searchParams, setSearchParams] = useSearchParams();

  // URL-derived state (single source of truth for filters/page)
  const debouncedQuery = searchParams.get('q') ?? '';
  const statusFilter = parseStatus(searchParams.get('status'));
  const currentPage = parsePage(searchParams.get('page'));

  const [posts, setPosts] = useState<Post[]>([]);
  const [totalCount, setTotalCount] = useState(0);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState('');

  // Local input state for the controlled search box. Kept separate from the
  // URL-bound debouncedQuery so typing stays smooth and IME composition can
  // suspend the URL sync without freezing the input.
  const [searchInput, setSearchInput] = useState(debouncedQuery);
  const [isComposing, setIsComposing] = useState(false);

  // Mutate the URL while preserving unrelated params. Empty values are dropped
  // so the canonical URL omits defaults (no `?q=`, `?page=1`, `?status=all`).
  // User-initiated calls push onto history so the browser back button steps
  // through distinct filter states; programmatic corrections (e.g. clamping
  // an out-of-range page) pass `replace: true` to avoid back-button traps.
  const updateParams = useCallback(
    (patch: Record<string, string | null>, options?: { replace?: boolean }) => {
      setSearchParams(
        (prev) => {
          const next = new URLSearchParams(prev);
          for (const [key, value] of Object.entries(patch)) {
            if (value == null || value === '') {
              next.delete(key);
            } else {
              next.set(key, value);
            }
          }
          return next;
        },
        { replace: options?.replace },
      );
    },
    [setSearchParams],
  );

  // Canonicalize the URL: drop or normalize params that don't match their
  // canonical form (defaults, empty values, unparseable values). Uses replace
  // so back doesn't bounce to the non-canonical form. Settles in one rewrite
  // because the canonical state has nothing left to fix on the next pass.
  useEffect(() => {
    const next = new URLSearchParams(searchParams);
    let changed = false;

    if (next.has('q') && next.get('q') === '') {
      next.delete('q');
      changed = true;
    }

    if (next.has('status') && parseStatus(next.get('status')) === 'all') {
      next.delete('status');
      changed = true;
    }

    if (next.has('page')) {
      const raw = next.get('page')!;
      const parsed = parsePage(raw);
      if (parsed === 1) {
        next.delete('page');
        changed = true;
      } else if (raw !== String(parsed)) {
        next.set('page', String(parsed));
        changed = true;
      }
    }

    if (changed) {
      setSearchParams(next, { replace: true });
    }
  }, [searchParams, setSearchParams]);

  // Resync the local input when the URL changes externally (back/forward).
  useEffect(() => {
    setSearchInput(debouncedQuery);
  }, [debouncedQuery]);

  // Debounced search → URL. The debounce coalesces a burst of keystrokes
  // into one history entry; intentional pauses (>300ms) create separate
  // entries the user can step through with browser back.
  useEffect(() => {
    if (isComposing) return;
    if (searchInput === debouncedQuery) return;

    const timer = setTimeout(() => {
      updateParams({ q: searchInput, page: null });
    }, 300);
    return () => clearTimeout(timer);
  }, [searchInput, isComposing, debouncedQuery, updateParams]);

  // Fetch posts whenever the URL-derived filters change. A cancelled flag in
  // the cleanup discards stale responses so a slow fetch cannot overwrite the
  // result of a newer one (e.g. when back/forward triggers rapid changes).
  useEffect(() => {
    let cancelled = false;

    (async () => {
      try {
        setIsLoading(true);
        setError('');
        const offset = (currentPage - 1) * POSTS_PER_PAGE;
        const data = await apiClient.getPosts({
          status: statusFilter === 'all' ? undefined : statusFilter,
          q: debouncedQuery || undefined,
          limit: POSTS_PER_PAGE,
          offset,
        });
        if (cancelled) return;

        // If the requested page is past the end (URL was edited or stale)
        // normalize back to page 1. Use replace so back from the corrected
        // URL does not bounce the user straight back to the bad page (which
        // would normalize again, trapping the back button). Skip toggling
        // isLoading off so the re-running effect owns the loading state.
        if (data.posts.length === 0 && currentPage > 1) {
          updateParams({ page: null }, { replace: true });
          return;
        }

        setPosts(data.posts);
        setTotalCount(data.total);
        setIsLoading(false);
      } catch (err) {
        if (cancelled) return;
        setError(err instanceof Error ? err.message : 'Failed to load posts');
        setIsLoading(false);
      }
    })();

    return () => {
      cancelled = true;
    };
  }, [currentPage, statusFilter, debouncedQuery, updateParams]);

  // Explicit user actions push onto history.
  const handleStatusChange = (newStatus: StatusFilter) => {
    updateParams({
      status: newStatus === 'all' ? null : newStatus,
      page: null,
    });
  };

  const handleClearSearch = () => {
    setSearchInput('');
    updateParams({ q: null, page: null });
  };

  const handlePrevPage = () => {
    const prev = Math.max(1, currentPage - 1);
    updateParams({ page: prev === 1 ? null : String(prev) });
  };

  const handleNextPage = () => {
    updateParams({ page: String(currentPage + 1) });
  };

  // IME composition handlers
  const handleCompositionStart = () => {
    setIsComposing(true);
  };

  const handleCompositionEnd = () => {
    setIsComposing(false);
  };

  const totalPages = Math.ceil(totalCount / POSTS_PER_PAGE);

  const formatDate = (dateString: string | null) => {
    if (!dateString) return '-';
    return formatInTimeZone(new Date(dateString), BLOG_TIMEZONE, 'yyyy-MM-dd');
  };

  const formatDateDetail = (dateString: string | null) => {
    if (!dateString) return '';
    return formatInTimeZone(new Date(dateString), BLOG_TIMEZONE, 'yyyy-MM-dd HH:mm (zzz)');
  };

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex justify-between items-center">
        <h1 className="text-2xl font-sans font-bold text-primary-900">
          Posts
        </h1>
        <Link
          to="/posts/new"
          className="bg-primary-900 text-white px-4 py-2 rounded-md hover:bg-primary-800 transition-colors text-sm font-medium"
        >
          New Post
        </Link>
      </div>

      {/* Filters */}
      <div className="bg-white rounded-lg shadow-sm p-4 space-y-4">
        <div className="flex flex-col sm:flex-row gap-4">
          {/* Search input */}
          <div className="flex-1">
            <label
              htmlFor="search-input"
              className="block text-sm font-medium text-primary-700 mb-1"
            >
              Search
            </label>
            <div className="flex gap-2">
              <input
                id="search-input"
                type="text"
                value={searchInput}
                onChange={(e) => setSearchInput(e.target.value)}
                autoCapitalize="none"
                onCompositionStart={handleCompositionStart}
                onCompositionEnd={handleCompositionEnd}
                placeholder="Search by title or content..."
                className="flex-1 min-w-0 px-3 py-2 border border-primary-300 rounded-md focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
              />
              {searchInput && (
                <button
                  onClick={handleClearSearch}
                  className="px-3 py-2 text-primary-600 border border-primary-300 rounded-md hover:bg-primary-50 transition-colors"
                  title="Clear search"
                >
                  Clear
                </button>
              )}
            </div>
          </div>

          {/* Status filter */}
          <div className="w-full sm:w-48">
            <label
              htmlFor="status-filter"
              className="block text-sm font-medium text-primary-700 mb-1"
            >
              Status
            </label>
            <select
              id="status-filter"
              value={statusFilter}
              onChange={(e) => handleStatusChange(e.target.value as StatusFilter)}
              className="w-full px-3 py-2 border border-primary-300 rounded-md focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
            >
              <option value="all">All</option>
              <option value="draft">Draft</option>
              <option value="published">Published</option>
            </select>
          </div>
        </div>

        <div className="text-sm text-primary-600">
          {debouncedQuery ? (
            <>Search results for "{debouncedQuery}": {totalCount}</>
          ) : (
            <>{totalCount} posts</>
          )}
        </div>
      </div>

      {/* Posts table */}
      {error ? (
        <div className="bg-red-50 border border-red-200 text-red-700 px-4 py-3 rounded">
          {error}
        </div>
      ) : isLoading ? (
        <div className="flex justify-center items-center min-h-64">
          <div className="text-primary-600">Loading...</div>
        </div>
      ) : posts.length === 0 ? (
        <div className="bg-white rounded-lg shadow-sm p-8 text-center text-primary-500">
          {debouncedQuery
            ? `No posts found matching "${debouncedQuery}"`
            : 'No posts found'}
        </div>
      ) : (
        <div className="sm:bg-white sm:rounded-lg sm:shadow-sm sm:overflow-x-auto max-sm:overflow-x-hidden">
          <table className="min-w-full max-sm:block sm:divide-y sm:divide-primary-200">
            <thead className="bg-primary-50 max-sm:hidden">
              <tr>
                <th className="px-6 py-3 text-left text-xs font-medium text-primary-700 uppercase tracking-wider">
                  Title
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-primary-700 uppercase tracking-wider">
                  Status
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-primary-700 uppercase tracking-wider">
                  Pinned
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-primary-700 uppercase tracking-wider">
                  Tags
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-primary-700 uppercase tracking-wider">
                  Updated
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-primary-700 uppercase tracking-wider">
                  Published
                </th>
                <th className="px-6 py-3 text-right text-xs font-medium text-primary-700 uppercase tracking-wider">
                  Views
                </th>
                <th className="px-6 py-3 text-right text-xs font-medium text-primary-700 uppercase tracking-wider">
                  Reactions
                </th>
              </tr>
            </thead>
            <tbody className="bg-white sm:divide-y sm:divide-primary-200 max-sm:block max-sm:bg-transparent max-sm:space-y-3">
              {posts.map((post) => (
                <tr
                  key={post.id}
                  className="sm:hover:bg-primary-50 transition-colors max-sm:block max-sm:bg-white max-sm:rounded-lg max-sm:shadow-sm max-sm:p-4 max-sm:space-y-2"
                >
                  <td className="px-6 py-4 max-sm:p-0 max-sm:block">
                    <Link
                      to={`/posts/${post.id}/edit`}
                      className="text-primary-900 hover:text-primary-700 font-medium max-sm:text-base max-sm:font-semibold max-sm:break-words"
                    >
                      {post.title}
                    </Link>
                  </td>
                  <td className="px-6 py-4 max-sm:p-0 max-sm:block">
                    <StatusBadge status={post.status} />
                  </td>
                  <td
                    aria-label={post.is_pinned ? 'Pinned' : undefined}
                    className={
                      post.is_pinned
                        ? 'px-6 py-4 text-center max-sm:p-0 max-sm:block max-sm:text-left'
                        : 'px-6 py-4 text-center max-sm:hidden'
                    }
                  >
                    {post.is_pinned && (
                      <span title="Pinned posts appear in the header">
                        📌
                      </span>
                    )}
                  </td>
                  <td className="px-6 py-4 max-sm:p-0 max-sm:block">
                    <TagList tags={post.tags} />
                  </td>
                  <td
                    aria-label={`Updated: ${formatDate(post.updated_at)}`}
                    className="px-6 py-4 text-sm text-primary-600 max-sm:p-0 max-sm:block max-sm:before:content-['Updated_'] max-sm:before:uppercase max-sm:before:text-xs max-sm:before:text-primary-500 max-sm:before:font-medium max-sm:before:mr-2"
                    title={formatDateDetail(post.updated_at)}
                  >
                    {formatDate(post.updated_at)}
                  </td>
                  <td
                    aria-label={post.published_at ? `Published: ${formatDate(post.published_at)}` : 'Not published'}
                    className="px-6 py-4 text-sm text-primary-600 max-sm:p-0 max-sm:block max-sm:before:content-['Published_'] max-sm:before:uppercase max-sm:before:text-xs max-sm:before:text-primary-500 max-sm:before:font-medium max-sm:before:mr-2"
                    title={formatDateDetail(post.published_at)}
                  >
                    {formatDate(post.published_at)}
                  </td>
                  <td
                    aria-label={`Views: ${post.view_count.toLocaleString()}`}
                    className="px-6 py-4 text-sm text-primary-600 text-right max-sm:p-0 max-sm:block max-sm:text-left max-sm:before:content-['Views_'] max-sm:before:uppercase max-sm:before:text-xs max-sm:before:text-primary-500 max-sm:before:font-medium max-sm:before:mr-2"
                  >
                    {post.view_count.toLocaleString()}
                  </td>
                  <td className="px-6 py-4 text-sm text-primary-600 text-right max-sm:p-0 max-sm:block max-sm:text-left max-sm:before:content-['Reactions_'] max-sm:before:uppercase max-sm:before:text-xs max-sm:before:text-primary-500 max-sm:before:font-medium max-sm:before:mr-2">
                    <span className="max-sm:hidden">
                      <ReactionTotalPopover reactions={post.reactions} />
                    </span>
                    <span className="hidden max-sm:inline">
                      <ReactionBreakdown reactions={post.reactions} />
                    </span>
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
            onClick={handlePrevPage}
            disabled={currentPage === 1}
            className="px-4 py-2 border border-primary-300 rounded-md text-sm font-medium text-primary-700 hover:bg-primary-50 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
          >
            Prev
          </button>
          <span className="px-4 py-2 text-sm text-primary-600">
            {currentPage} / {totalPages}
          </span>
          <button
            onClick={handleNextPage}
            disabled={currentPage === totalPages}
            className="px-4 py-2 border border-primary-300 rounded-md text-sm font-medium text-primary-700 hover:bg-primary-50 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
          >
            Next
          </button>
        </div>
      )}
    </div>
  );
}
