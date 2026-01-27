import { getCsrfToken } from '../utils/cookies';

const API_BASE = '/api/v1';

// 型定義
export interface User {
  id: number;
  username: string;
  created_at: string;
  updated_at: string;
}

export interface Post {
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
}

export interface Tag {
  name: string;
  count: number;
}

export interface LoginRequest {
  username: string;
  password: string;
}

export interface PostCreateRequest {
  title: string;
  slug: string;
  content: string;
  tags?: string;
  is_pinned?: boolean;
}

export interface PostUpdateRequest {
  title: string;
  slug: string;
  content: string;
  tags?: string;
  is_pinned?: boolean;
}

export interface ApiError {
  error: string;
}

export interface ImageUploadResponse {
  url: string;
  filename: string;
}

export interface PostsResponse {
  posts: Post[];
  total: number;
}

/**
 * API client with CSRF token handling
 */
class ApiClient {
  private baseUrl: string;

  constructor(baseUrl: string = API_BASE) {
    this.baseUrl = baseUrl;
  }

  /**
   * Make an API request with automatic CSRF token handling
   */
  private async request<T>(
    endpoint: string,
    options: RequestInit = {}
  ): Promise<T> {
    const url = `${this.baseUrl}${endpoint}`;
    const headers: Record<string, string> = {
      'Content-Type': 'application/json',
    };

    // Add CSRF token for state-changing methods
    const method = options.method?.toUpperCase() || 'GET';
    if (['POST', 'PUT', 'DELETE', 'PATCH'].includes(method)) {
      const csrfToken = getCsrfToken();
      if (csrfToken) {
        headers['X-CSRF-Token'] = csrfToken;
      }
    }

    const response = await fetch(url, {
      ...options,
      headers,
      credentials: 'include', // Include cookies
    });

    // Handle 401 Unauthorized - redirect to login
    if (response.status === 401) {
      // Only redirect if not already on login page
      if (!window.location.pathname.includes('/login')) {
        window.location.href = '/admin/login';
      }
      throw new Error('Unauthorized');
    }

    // Handle empty responses (204 No Content)
    if (response.status === 204) {
      return undefined as T;
    }

    // Parse JSON response
    const data = await response.json();

    // Handle error responses
    if (!response.ok) {
      throw new Error((data as ApiError).error || `HTTP ${response.status}`);
    }

    return data as T;
  }

  // Auth endpoints
  async login(credentials: LoginRequest): Promise<User> {
    return this.request<User>('/auth/login', {
      method: 'POST',
      body: JSON.stringify(credentials),
    });
  }

  async logout(): Promise<void> {
    return this.request<void>('/auth/logout', {
      method: 'POST',
    });
  }

  async me(): Promise<User> {
    return this.request<User>('/auth/me');
  }

  // Posts endpoints
  async getPosts(params?: {
    status?: string;
    tag?: string;
    q?: string;
    limit?: number;
    offset?: number;
  }): Promise<PostsResponse> {
    const searchParams = new URLSearchParams();
    if (params?.status) searchParams.append('status', params.status);
    if (params?.tag) searchParams.append('tag', params.tag);
    if (params?.q) searchParams.append('q', params.q);
    if (params?.limit !== undefined) searchParams.append('limit', String(params.limit));
    if (params?.offset !== undefined) searchParams.append('offset', String(params.offset));

    const query = searchParams.toString();
    const endpoint = query ? `/posts?${query}` : '/posts';

    return this.request<PostsResponse>(endpoint);
  }

  async getPost(id: number): Promise<Post> {
    return this.request<Post>(`/posts/${id}`);
  }

  async createPost(post: PostCreateRequest): Promise<Post> {
    return this.request<Post>('/posts', {
      method: 'POST',
      body: JSON.stringify(post),
    });
  }

  async updatePost(id: number, post: PostUpdateRequest): Promise<Post> {
    return this.request<Post>(`/posts/${id}`, {
      method: 'PUT',
      body: JSON.stringify(post),
    });
  }

  async deletePost(id: number): Promise<void> {
    return this.request<void>(`/posts/${id}`, {
      method: 'DELETE',
    });
  }

  async publishPost(id: number): Promise<Post> {
    return this.request<Post>(`/posts/${id}/publish`, {
      method: 'POST',
    });
  }

  async unpublishPost(id: number): Promise<Post> {
    return this.request<Post>(`/posts/${id}/unpublish`, {
      method: 'POST',
    });
  }

  async pinPost(id: number): Promise<Post> {
    return this.request<Post>(`/posts/${id}/pin`, {
      method: 'POST',
    });
  }

  async unpinPost(id: number): Promise<Post> {
    return this.request<Post>(`/posts/${id}/unpin`, {
      method: 'POST',
    });
  }

  // Tags endpoint
  async getTags(params?: { status?: string }): Promise<Tag[]> {
    const searchParams = new URLSearchParams();
    if (params?.status) searchParams.append('status', params.status);

    const query = searchParams.toString();
    const endpoint = query ? `/tags?${query}` : '/tags';

    return this.request<Tag[]>(endpoint);
  }

  // Markdown preview endpoint
  // コンテンツ内にゼロ幅スペース（\u200B）が含まれている場合、
  // スクロール用マーカー <span id="cursor-line-marker"></span> に置換される
  async previewMarkdown(content: string): Promise<string> {
    const response = await this.request<{ html: string }>('/markdown/preview', {
      method: 'POST',
      body: JSON.stringify({ content }),
    });
    return response.html;
  }

  // Image upload endpoint
  async uploadImage(file: File): Promise<ImageUploadResponse> {
    const formData = new FormData();
    formData.append('image', file);

    const headers: Record<string, string> = {};

    // Add CSRF token
    const csrfToken = getCsrfToken();
    if (csrfToken) {
      headers['X-CSRF-Token'] = csrfToken;
    }

    // Content-Typeは設定しない（ブラウザがmultipart/form-dataのboundaryを自動設定）
    const response = await fetch(`${this.baseUrl}/images`, {
      method: 'POST',
      headers,
      body: formData,
      credentials: 'include',
    });

    // Handle 401 Unauthorized - redirect to login
    if (response.status === 401) {
      if (!window.location.pathname.includes('/login')) {
        window.location.href = '/admin/login';
      }
      throw new Error('Unauthorized');
    }

    // Parse JSON response
    const data = await response.json();

    // Handle error responses
    if (!response.ok) {
      throw new Error((data as ApiError).error || `HTTP ${response.status}`);
    }

    return data as ImageUploadResponse;
  }
}

// Export a singleton instance
export const apiClient = new ApiClient();
