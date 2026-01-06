import { describe, it, expect, beforeEach } from 'vitest'
import { getCookie, getCsrfToken } from './cookies'

describe('Cookie utilities', () => {
  // Helper function to set mock document.cookie
  const setMockCookie = (cookieString: string) => {
    Object.defineProperty(document, 'cookie', {
      writable: true,
      value: cookieString,
    })
  }

  beforeEach(() => {
    // Reset document.cookie before each test
    setMockCookie('')
  })

  describe('getCookie', () => {
    it('should return cookie value when cookie exists', () => {
      setMockCookie('session_id=abc123')
      expect(getCookie('session_id')).toBe('abc123')
    })

    it('should return null when cookie does not exist', () => {
      setMockCookie('session_id=abc123')
      expect(getCookie('other_cookie')).toBeNull()
    })

    it('should return null when document.cookie is empty', () => {
      setMockCookie('')
      expect(getCookie('session_id')).toBeNull()
    })

    it('should handle multiple cookies correctly', () => {
      setMockCookie('session_id=abc123; csrf_token=xyz789; user=testuser')
      expect(getCookie('session_id')).toBe('abc123')
      expect(getCookie('csrf_token')).toBe('xyz789')
      expect(getCookie('user')).toBe('testuser')
    })

    it('should handle cookies with similar names', () => {
      setMockCookie('token=value1; csrf_token=value2')
      expect(getCookie('token')).toBe('value1')
      expect(getCookie('csrf_token')).toBe('value2')
    })

    it('should return null when cookie name appears multiple times (malformed)', () => {
      // This is an edge case - normally browsers don't allow duplicate cookie names
      // The function correctly returns null for malformed cookie strings
      setMockCookie('key=first; key=second')
      expect(getCookie('key')).toBeNull()
    })

    it('should handle cookie values with special characters', () => {
      setMockCookie('encoded=Hello%20World')
      expect(getCookie('encoded')).toBe('Hello%20World')
    })

    it('should handle cookie values with equals signs', () => {
      setMockCookie('data=key=value')
      expect(getCookie('data')).toBe('key=value')
    })

    it('should trim values correctly when semicolons are present', () => {
      setMockCookie('session_id=abc123; csrf_token=xyz789')
      expect(getCookie('csrf_token')).toBe('xyz789')
    })
  })

  describe('getCsrfToken', () => {
    it('should return csrf_token value when it exists', () => {
      setMockCookie('csrf_token=test_token_123')
      expect(getCsrfToken()).toBe('test_token_123')
    })

    it('should return null when csrf_token does not exist', () => {
      setMockCookie('session_id=abc123')
      expect(getCsrfToken()).toBeNull()
    })

    it('should return null when document.cookie is empty', () => {
      setMockCookie('')
      expect(getCsrfToken()).toBeNull()
    })

    it('should return csrf_token value from multiple cookies', () => {
      setMockCookie('session_id=abc123; csrf_token=xyz789; user=testuser')
      expect(getCsrfToken()).toBe('xyz789')
    })
  })
})
