import { http, HttpResponse } from 'msw';

const API_BASE = '/api/v1';

// Type definitions
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
  is_pinned: boolean;
  created_at: string;
  updated_at: string;
  published_at: string | null;
  view_count: number;
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

// Mock data for testing
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
    is_pinned: false,
    created_at: '2024-01-01T00:00:00Z',
    updated_at: '2024-01-01T00:00:00Z',
    published_at: '2024-01-01T10:00:00Z',
    view_count: 42,
  },
  {
    id: 2,
    title: 'Test Post 2',
    slug: 'test-post-2',
    content: 'Draft content that has not been published yet.',
    status: 'draft',
    tags: 'Go',
    is_pinned: false,
    created_at: '2024-01-02T00:00:00Z',
    updated_at: '2024-01-02T00:00:00Z',
    published_at: null,
    view_count: 0,
  },
  {
    id: 3,
    title: 'Another Published Post',
    slug: 'another-published-post',
    content: 'This is another published post with different tags.',
    status: 'published',
    tags: 'Web, Tutorial',
    is_pinned: false,
    created_at: '2024-01-03T00:00:00Z',
    updated_at: '2024-01-03T00:00:00Z',
    published_at: '2024-01-03T12:00:00Z',
    view_count: 157,
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
      is_pinned: false,
      status: 'draft',
      created_at: new Date().toISOString(),
      updated_at: new Date().toISOString(),
      published_at: null,
      view_count: 0,
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

  http.post(`${API_BASE}/posts/:id/pin`, ({ params }) => {
    const postIndex = mockPosts.findIndex(p => p.id === Number(params.id));

    if (postIndex === -1) {
      return HttpResponse.json(
        { error: 'Post not found' },
        { status: 404 }
      );
    }

    mockPosts[postIndex] = {
      ...mockPosts[postIndex]!,
      is_pinned: true,
      updated_at: new Date().toISOString(),
    };

    return HttpResponse.json(mockPosts[postIndex]);
  }),

  http.post(`${API_BASE}/posts/:id/unpin`, ({ params }) => {
    const postIndex = mockPosts.findIndex(p => p.id === Number(params.id));

    if (postIndex === -1) {
      return HttpResponse.json(
        { error: 'Post not found' },
        { status: 404 }
      );
    }

    mockPosts[postIndex] = {
      ...mockPosts[postIndex]!,
      is_pinned: false,
      updated_at: new Date().toISOString(),
    };

    return HttpResponse.json(mockPosts[postIndex]);
  }),

  // Markdown preview endpoint
  http.post(`${API_BASE}/markdown/preview`, async ({ request }) => {
    const { content } = await request.json() as { content: string };

    // Return empty string for empty content
    if (!content || content.trim() === '') {
      return HttpResponse.json({ html: '' });
    }

    // Simple Markdown to HTML conversion (for testing)
    // Calculate line numbers and add data-line attributes
    const lines = content.split('\n');
    const htmlParts: string[] = [];
    let lineNum = 0;

    for (let i = 0; i < lines.length; i++) {
      const line = lines[i] ?? '';

      // Headings
      const h1Match = line.match(/^# (.+)$/);
      if (h1Match) {
        htmlParts.push(`<h1 data-line="${lineNum}">${h1Match[1]}</h1>`);
        lineNum++;
        continue;
      }

      const h2Match = line.match(/^## (.+)$/);
      if (h2Match) {
        htmlParts.push(`<h2 data-line="${lineNum}">${h2Match[1]}</h2>`);
        lineNum++;
        continue;
      }

      const h3Match = line.match(/^### (.+)$/);
      if (h3Match) {
        htmlParts.push(`<h3 data-line="${lineNum}">${h3Match[1]}</h3>`);
        lineNum++;
        continue;
      }

      // Empty lines are paragraph separators
      if (line.trim() === '') {
        lineNum++;
        continue;
      }

      // Everything else is a paragraph
      htmlParts.push(`<p data-line="${lineNum}">${line}</p>`);
      lineNum++;
    }

    return HttpResponse.json({ html: htmlParts.join('') });
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

  // Reaction types endpoints
  http.get(`${API_BASE}/reaction-types`, () =>
    HttpResponse.json([
      { id: 1, emoji: '👍', label: 'いいね', sort_order: 10, is_active: true, is_seed: true },
    ]),
  ),

  http.post(`${API_BASE}/reaction-types`, async ({ request }) => {
    const body = (await request.json()) as { emoji: string; label: string; sort_order: number };
    return HttpResponse.json(
      { id: 7, emoji: body.emoji, label: body.label, sort_order: body.sort_order, is_active: true, is_seed: false },
      { status: 201 },
    );
  }),

  http.put(`${API_BASE}/reaction-types/:id`, async ({ request, params }) => {
    const body = (await request.json()) as { emoji: string; label: string; sort_order: number; is_active: boolean };
    return HttpResponse.json({ id: Number(params.id), is_seed: false, ...body });
  }),

  http.delete(`${API_BASE}/reaction-types/:id`, () => new HttpResponse(null, { status: 204 })),

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
