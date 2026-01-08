import { http, HttpResponse } from 'msw';

const API_BASE = '/api/v1';

// 型定義
interface User {
  id: number;
  username: string;
  created_at: string;
  updated_at: string;
}

interface Post {
  id: number;
  title: string;
  slug: string;
  content: string;
  status: 'draft' | 'published';
  tags: string;
  created_at: string;
  updated_at: string;
  published_at: string | null;
}

interface Tag {
  name: string;
  count: number;
}

interface LoginRequest {
  username: string;
  password: string;
}

interface PostCreateRequest {
  title: string;
  slug: string;
  content: string;
  tags?: string;
}

interface PostUpdateRequest {
  title: string;
  slug: string;
  content: string;
  tags?: string;
}

// テスト用のモックデータ
const mockUser: User = {
  id: 1,
  username: 'testuser',
  created_at: '2024-01-01T00:00:00Z',
  updated_at: '2024-01-01T00:00:00Z',
};

let mockPosts: Post[] = [
  {
    id: 1,
    title: 'Test Post 1',
    slug: 'test-post-1',
    content: 'This is test content for the first post.\n\nIt has multiple paragraphs.',
    status: 'published',
    tags: 'Go, Testing',
    created_at: '2024-01-01T00:00:00Z',
    updated_at: '2024-01-01T00:00:00Z',
    published_at: '2024-01-01T10:00:00Z',
  },
  {
    id: 2,
    title: 'Test Post 2',
    slug: 'test-post-2',
    content: 'Draft content that has not been published yet.',
    status: 'draft',
    tags: 'Go',
    created_at: '2024-01-02T00:00:00Z',
    updated_at: '2024-01-02T00:00:00Z',
    published_at: null,
  },
  {
    id: 3,
    title: 'Another Published Post',
    slug: 'another-published-post',
    content: 'This is another published post with different tags.',
    status: 'published',
    tags: 'Web, Tutorial',
    created_at: '2024-01-03T00:00:00Z',
    updated_at: '2024-01-03T00:00:00Z',
    published_at: '2024-01-03T12:00:00Z',
  },
];

export const handlers = [
  // Auth endpoints
  http.post(`${API_BASE}/auth/login`, async ({ request }) => {
    const { username, password } = await request.json() as LoginRequest;

    if (username === 'testuser' && password === 'password') {
      return HttpResponse.json(mockUser);
    }

    return HttpResponse.json(
      { error: 'Invalid username or password' },
      { status: 401 }
    );
  }),

  http.post(`${API_BASE}/auth/logout`, () => {
    return new HttpResponse(null, { status: 204 });
  }),

  http.get(`${API_BASE}/auth/me`, () => {
    return HttpResponse.json(mockUser);
  }),

  // Posts endpoints
  http.get(`${API_BASE}/posts`, ({ request }) => {
    const url = new URL(request.url);
    const status = url.searchParams.get('status');
    const tag = url.searchParams.get('tag');
    const limitStr = url.searchParams.get('limit');
    const offsetStr = url.searchParams.get('offset');

    let filtered = [...mockPosts];

    // Filter by status
    if (status) {
      filtered = filtered.filter(p => p.status === status);
    }

    // Filter by tag
    if (tag) {
      filtered = filtered.filter(p =>
        p.tags && p.tags.split(',').map(t => t.trim()).includes(tag)
      );
    }

    const total = filtered.length;

    // Apply pagination
    const limit = limitStr ? parseInt(limitStr, 10) : 20;
    const offset = offsetStr ? parseInt(offsetStr, 10) : 0;
    const paginated = filtered.slice(offset, offset + limit);

    return HttpResponse.json({ posts: paginated, total });
  }),

  http.get(`${API_BASE}/posts/:id`, ({ params }) => {
    const post = mockPosts.find(p => p.id === Number(params.id));

    if (!post) {
      return HttpResponse.json(
        { error: 'Post not found' },
        { status: 404 }
      );
    }

    return HttpResponse.json(post);
  }),

  http.post(`${API_BASE}/posts`, async ({ request }) => {
    const body = await request.json() as PostCreateRequest;

    // Check if slug already exists
    const slugExists = mockPosts.some(p => p.slug === body.slug);
    if (slugExists) {
      return HttpResponse.json(
        { error: 'slug already exists' },
        { status: 500 }
      );
    }

    const newPost: Post = {
      id: Math.max(...mockPosts.map(p => p.id), 0) + 1,
      title: body.title,
      slug: body.slug,
      content: body.content,
      tags: body.tags || '',
      status: 'draft',
      created_at: new Date().toISOString(),
      updated_at: new Date().toISOString(),
      published_at: null,
    };

    mockPosts.push(newPost);
    return HttpResponse.json(newPost, { status: 201 });
  }),

  http.put(`${API_BASE}/posts/:id`, async ({ params, request }) => {
    const body = await request.json() as PostUpdateRequest;
    const postIndex = mockPosts.findIndex(p => p.id === Number(params.id));

    if (postIndex === -1) {
      return HttpResponse.json(
        { error: 'Post not found' },
        { status: 404 }
      );
    }

    // Check if slug already exists (excluding current post)
    const slugExists = mockPosts.some(
      p => p.slug === body.slug && p.id !== Number(params.id)
    );
    if (slugExists) {
      return HttpResponse.json(
        { error: 'slug already exists' },
        { status: 500 }
      );
    }

    mockPosts[postIndex] = {
      ...mockPosts[postIndex]!,
      title: body.title,
      slug: body.slug,
      content: body.content,
      tags: body.tags || '',
      updated_at: new Date().toISOString(),
    };

    return HttpResponse.json(mockPosts[postIndex]);
  }),

  http.delete(`${API_BASE}/posts/:id`, ({ params }) => {
    const postIndex = mockPosts.findIndex(p => p.id === Number(params.id));

    if (postIndex === -1) {
      return HttpResponse.json(
        { error: 'Post not found' },
        { status: 404 }
      );
    }

    mockPosts.splice(postIndex, 1);
    return new HttpResponse(null, { status: 204 });
  }),

  http.post(`${API_BASE}/posts/:id/publish`, ({ params }) => {
    const postIndex = mockPosts.findIndex(p => p.id === Number(params.id));

    if (postIndex === -1) {
      return HttpResponse.json(
        { error: 'Post not found' },
        { status: 404 }
      );
    }

    mockPosts[postIndex] = {
      ...mockPosts[postIndex]!,
      status: 'published',
      published_at: new Date().toISOString(),
      updated_at: new Date().toISOString(),
    };

    return HttpResponse.json(mockPosts[postIndex]);
  }),

  http.post(`${API_BASE}/posts/:id/unpublish`, ({ params }) => {
    const postIndex = mockPosts.findIndex(p => p.id === Number(params.id));

    if (postIndex === -1) {
      return HttpResponse.json(
        { error: 'Post not found' },
        { status: 404 }
      );
    }

    mockPosts[postIndex] = {
      ...mockPosts[postIndex]!,
      status: 'draft',
      published_at: null,
      updated_at: new Date().toISOString(),
    };

    return HttpResponse.json(mockPosts[postIndex]);
  }),

  // Tags endpoint
  http.get(`${API_BASE}/tags`, ({ request }) => {
    const url = new URL(request.url);
    const status = url.searchParams.get('status');

    let filteredPosts = [...mockPosts];
    if (status) {
      filteredPosts = filteredPosts.filter(p => p.status === status);
    }

    // Count tags
    const tagCounts: Record<string, number> = {};
    filteredPosts.forEach(post => {
      if (post.tags) {
        const tags = post.tags.split(',').map(t => t.trim()).filter(Boolean);
        tags.forEach(tag => {
          tagCounts[tag] = (tagCounts[tag] || 0) + 1;
        });
      }
    });

    // Convert to array and sort by count (descending), then by name (ascending)
    const tagsArray: Tag[] = Object.entries(tagCounts)
      .map(([name, count]) => ({ name, count }))
      .sort((a, b) => {
        if (b.count !== a.count) {
          return b.count - a.count;
        }
        return a.name.localeCompare(b.name);
      });

    return HttpResponse.json(tagsArray);
  }),
];
