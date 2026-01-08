import { describe, it, expect } from 'vitest'
import { apiClient } from './client'

describe('API Client', () => {
  describe('Auth endpoints', () => {
    describe('login', () => {
      it('should login successfully with valid credentials', async () => {
        const result = await apiClient.login({
          username: 'testuser',
          password: 'password',
        })

        expect(result).toHaveProperty('id', 1)
        expect(result).toHaveProperty('username', 'testuser')
      })

      it('should throw error with invalid credentials', async () => {
        await expect(
          apiClient.login({
            username: 'wronguser',
            password: 'wrongpass',
          })
        ).rejects.toThrow()
      })
    })

    describe('logout', () => {
      it('should logout successfully', async () => {
        await expect(apiClient.logout()).resolves.toBeUndefined()
      })
    })

    describe('me', () => {
      it('should fetch current user info', async () => {
        const user = await apiClient.me()

        expect(user).toHaveProperty('id', 1)
        expect(user).toHaveProperty('username', 'testuser')
      })
    })
  })

  describe('Posts endpoints', () => {
    describe('getPosts', () => {
      it('should fetch all posts with total count', async () => {
        const response = await apiClient.getPosts()

        expect(response).toHaveProperty('posts')
        expect(response).toHaveProperty('total')
        expect(response.posts).toBeInstanceOf(Array)
        expect(response.posts.length).toBeGreaterThan(0)
        expect(response.total).toBeGreaterThan(0)
      })

      it('should filter posts by status', async () => {
        const response = await apiClient.getPosts({ status: 'draft' })

        expect(response.posts).toBeInstanceOf(Array)
        response.posts.forEach((post) => {
          expect(post.status).toBe('draft')
        })
      })

      it('should filter posts by tag', async () => {
        const response = await apiClient.getPosts({ tag: 'Go' })

        expect(response.posts).toBeInstanceOf(Array)
        expect(response.posts.length).toBeGreaterThan(0)
      })

      it('should support pagination with limit and offset', async () => {
        const response = await apiClient.getPosts({ limit: 1, offset: 0 })

        expect(response.posts).toHaveLength(1)
        expect(response.total).toBeGreaterThan(1)
      })
    })

    describe('getPost', () => {
      it('should fetch a single post by id', async () => {
        const post = await apiClient.getPost(1)

        expect(post).toHaveProperty('id', 1)
        expect(post).toHaveProperty('title')
        expect(post).toHaveProperty('slug')
      })

      it('should throw error for non-existent post', async () => {
        await expect(apiClient.getPost(9999)).rejects.toThrow()
      })
    })

    describe('createPost', () => {
      it('should create a new post', async () => {
        const newPost = {
          title: 'New Test Post',
          slug: 'new-test-post',
          content: 'This is new content',
          tags: 'Test',
        }

        const post = await apiClient.createPost(newPost)

        expect(post).toHaveProperty('id')
        expect(post).toHaveProperty('title', newPost.title)
        expect(post).toHaveProperty('slug', newPost.slug)
        expect(post).toHaveProperty('status', 'draft')
      })
    })

    describe('updatePost', () => {
      it('should update an existing post', async () => {
        const updatedData = {
          title: 'Updated Title',
          slug: 'test-post-1',
          content: 'Updated content',
          tags: 'Go, Updated',
        }

        const post = await apiClient.updatePost(1, updatedData)

        expect(post).toHaveProperty('id', 1)
        expect(post).toHaveProperty('title', updatedData.title)
      })

      it('should throw error for non-existent post', async () => {
        await expect(
          apiClient.updatePost(9999, {
            title: 'Updated',
            slug: 'updated',
            content: 'Updated',
          })
        ).rejects.toThrow()
      })
    })

    describe('deletePost', () => {
      it('should delete a post', async () => {
        await expect(apiClient.deletePost(1)).resolves.toBeUndefined()
      })

      it('should throw error for non-existent post', async () => {
        await expect(apiClient.deletePost(9999)).rejects.toThrow()
      })
    })

    describe('publishPost', () => {
      it('should publish a draft post', async () => {
        const post = await apiClient.publishPost(2)

        expect(post).toHaveProperty('id', 2)
        expect(post).toHaveProperty('status', 'published')
        expect(post.published_at).not.toBeNull()
      })

      it('should throw error for non-existent post', async () => {
        await expect(apiClient.publishPost(9999)).rejects.toThrow()
      })
    })

    describe('unpublishPost', () => {
      it('should unpublish a published post', async () => {
        // ID 3 is initially published and won't be deleted by previous tests
        const post = await apiClient.unpublishPost(3)

        expect(post).toHaveProperty('id', 3)
        expect(post).toHaveProperty('status', 'draft')
        expect(post.published_at).toBeNull()
      })

      it('should throw error for non-existent post', async () => {
        await expect(apiClient.unpublishPost(9999)).rejects.toThrow()
      })
    })
  })

  describe('Tags endpoints', () => {
    describe('getTags', () => {
      it('should fetch all tags', async () => {
        const tags = await apiClient.getTags()

        expect(tags).toBeInstanceOf(Array)
        expect(tags.length).toBeGreaterThan(0)
        expect(tags[0]).toHaveProperty('name')
        expect(tags[0]).toHaveProperty('count')
      })

      it('should filter tags by status', async () => {
        const tags = await apiClient.getTags({ status: 'published' })

        expect(tags).toBeInstanceOf(Array)
      })
    })
  })
})
