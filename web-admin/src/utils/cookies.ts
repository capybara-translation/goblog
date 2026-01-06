/**
 * Get a cookie value by name
 */
export function getCookie(name: string): string | null {
  const value = `; ${document.cookie}`;
  const parts = value.split(`; ${name}=`);

  if (parts.length === 2) {
    const lastPart = parts.pop();
    if (lastPart) {
      return lastPart.split(';').shift() || null;
    }
  }

  return null;
}

/**
 * Get CSRF token from cookie
 */
export function getCsrfToken(): string | null {
  return getCookie('csrf_token');
}
