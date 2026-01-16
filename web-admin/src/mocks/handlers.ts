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
    const query = url.searchParams.get('q');
    const limitStr = url.searchParams.get('limit');
    const offsetStr = url.searchParams.get('offset');

    let filtered = [...mockPosts];

    // Filter by search query (title or content)
    if (query) {
      const lowerQuery = query.toLowerCase();
      filtered = filtered.filter(p =>
        p.title.toLowerCase().includes(lowerQuery) ||
        p.content.toLowerCase().includes(lowerQuery)
      );
    }

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

  // Markdown preview endpoint
  http.post(`${API_BASE}/markdown/preview`, async ({ request }) => {
    const { content } = await request.json() as { content: string };

    // 空のコンテンツは空文字を返す
    if (!content || content.trim() === '') {
      return HttpResponse.json({ html: '' });
    }

    // 簡易的なMarkdown→HTML変換（テスト用）
    let html = content;

    // 見出し
    html = html.replace(/^# (.+)$/gm, '<h1>$1</h1>');
    html = html.replace(/^## (.+)$/gm, '<h2>$1</h2>');
    html = html.replace(/^### (.+)$/gm, '<h3>$1</h3>');

    // 太字
    html = html.replace(/\*\*(.+?)\*\*/g, '<strong>$1</strong>');

    // 斜体
    html = html.replace(/\*(.+?)\*/g, '<em>$1</em>');

    // コードブロック
    html = html.replace(/```(\w+)?\n([\s\S]*?)```/g, '<pre><code>$2</code></pre>');

    // 段落
    html = html.split('\n\n').map(p => {
      if (!p.startsWith('<')) {
        return `<p>${p}</p>`;
      }
      return p;
    }).join('\n');

    return HttpResponse.json({ html });
  }),

  // Image upload endpoint
  http.post(`${API_BASE}/images`, async ({ request }) => {
    const formData = await request.formData();
    const file = formData.get('image') as File | null;

    if (!file) {
      return HttpResponse.json(
        { error: 'No image file provided' },
        { status: 400 }
      );
    }

    // Check file type
    const allowedTypes = ['image/jpeg', 'image/png', 'image/gif', 'image/webp'];
    if (!allowedTypes.includes(file.type)) {
      return HttpResponse.json(
        { error: 'Invalid file type. Allowed types: JPEG, PNG, GIF, WebP' },
        { status: 400 }
      );
    }

    // Check file size (5MB)
    const maxSize = 5 * 1024 * 1024;
    if (file.size > maxSize) {
      return HttpResponse.json(
        { error: `File too large. Maximum size is ${maxSize} bytes` },
        { status: 400 }
      );
    }

    // Generate mock response
    const extension = file.type.split('/')[1] === 'jpeg' ? 'jpg' : file.type.split('/')[1];
    const mockFilename = `mock-uuid-${Date.now()}.${extension}`;

    return HttpResponse.json(
      {
        url: `/uploads/${mockFilename}`,
        filename: file.name,
      },
      { status: 201 }
    );
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
