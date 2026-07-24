import { getCsrfToken } from '../utils/cookies';

const API_BASE = '/api/v1';

// Type definitions
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
  health_date: string | null;
  view_count: number;
  // Admin analytics: per-type reaction counts (all types incl. inactive).
  // Present on GET list/detail responses; omitted by mutation responses.
  reactions?: AdminReactionCount[];
}

export interface Tag {
  name: string;
  count: number;
}

export interface LoginRequest {
  username: string;
  password: string;
  remember_me?: boolean;
}

export interface PostCreateRequest {
  title: string;
  slug: string;
  content: string;
  tags?: string;
  is_pinned?: boolean;
  health_date?: string | null;
}

export interface PostUpdateRequest {
  title: string;
  slug: string;
  content: string;
  tags?: string;
  is_pinned?: boolean;
  health_date?: string | null;
}

export interface ApiError {
  error: string;
}

export interface ImageUploadResponse {
  url: string;
  filename: string;
}

export interface ReactionType {
  id: number;
  emoji: string;
  label: string;
  sort_order: number;
  is_active: boolean;
  is_seed: boolean;
}

export interface Device {
  kind: 'remember' | 'session';
  id: string;
  device: string;
  browser: string;
  os: string;
  ip: string;
  last_used_at: string;
  created_at: string;
  is_current: boolean;
  is_ephemeral: boolean;
}

export interface HealthPlanetStatus {
  enabled: boolean;
  authorized?: boolean;
  token_expires_at?: string | null;
  last_refreshed_at?: string | null;
}

export interface AdminReactionCount {
  id: number;
  emoji: string;
  label: string;
  count: number;
  is_active: boolean;
}

export interface ReactionTypeCreateRequest {
  emoji: string;
  label: string;
  sort_order: number;
}

export interface ReactionTypeUpdateRequest {
  emoji: string;
  label: string;
  sort_order: number;
  is_active: boolean;
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

  // Reaction type master endpoints
  async getReactionTypes(): Promise<ReactionType[]> {
    return this.request<ReactionType[]>('/reaction-types');
  }

  async createReactionType(req: ReactionTypeCreateRequest): Promise<ReactionType> {
    return this.request<ReactionType>('/reaction-types', {
      method: 'POST',
      body: JSON.stringify(req),
    });
  }

  async updateReactionType(id: number, req: ReactionTypeUpdateRequest): Promise<ReactionType> {
    return this.request<ReactionType>(`/reaction-types/${id}`, {
      method: 'PUT',
      body: JSON.stringify(req),
    });
  }

  async deleteReactionType(id: number): Promise<void> {
    return this.request<void>(`/reaction-types/${id}`, { method: 'DELETE' });
  }

  // Device management endpoints
  async getDevices(): Promise<Device[]> {
    return this.request<Device[]>('/devices');
  }

  async revokeDevice(kind: 'remember' | 'session', id: string): Promise<void> {
    return this.request<void>(`/devices/${kind}/${encodeURIComponent(id)}`, {
      method: 'DELETE',
    });
  }

  async logoutOtherDevices(): Promise<void> {
    return this.request<void>('/devices/logout-others', { method: 'POST' });
  }

  // Health Planet integration endpoints
  async getHealthPlanetStatus(): Promise<HealthPlanetStatus> {
    return this.request<HealthPlanetStatus>('/healthplanet/status');
  }

  async getHealthPlanetAuthURL(): Promise<{ url: string }> {
    return this.request<{ url: string }>('/healthplanet/auth-url');
  }

  async exchangeHealthPlanetCode(code: string): Promise<void> {
    return this.request<void>('/healthplanet/exchange', {
      method: 'POST',
      body: JSON.stringify({ code }),
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
  // Each block element gets a data-line attribute for synchronized preview scrolling
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

    // Don't set Content-Type (browser auto-sets multipart/form-data boundary)
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
