import { useState, useEffect } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { useAuth } from '../hooks/useAuth';
import { apiClient } from '../api/client';

const BLOG_TITLE = import.meta.env.VITE_BLOG_TITLE || 'goblog';

export function Header() {
  const { user, logout } = useAuth();
  const navigate = useNavigate();
  const [hpEnabled, setHpEnabled] = useState(false);

  useEffect(() => {
    // Feature-flagged: only show the menu item when the server says the
    // integration is enabled. Errors just leave the link hidden.
    apiClient
      .getHealthPlanetStatus()
      .then((s) => setHpEnabled(s.enabled))
      .catch(() => {});
  }, []);

  const handleLogout = async () => {
    try {
      await logout();
      navigate('/login');
    } catch (error) {
      console.error('Logout failed:', error);
    }
  };

  return (
    <header className="bg-white border-b border-primary-200 shadow-sm">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <div className="flex flex-wrap items-center gap-x-4 sm:gap-x-8 gap-y-2 py-3 sm:py-0 sm:h-16 sm:flex-nowrap">
          <a
            href="/"
            target="_blank"
            rel="noopener noreferrer"
            className="text-xl font-sans font-bold text-primary-900 hover:text-primary-700 transition-colors order-1 truncate min-w-0"
          >
            {BLOG_TITLE}
          </a>

          <nav className="flex gap-4 basis-full sm:basis-auto order-3 sm:order-2">
            <Link
              to="/posts"
              className="text-sm font-medium text-primary-700 hover:text-primary-900 transition-colors"
            >
              Posts
            </Link>
            <Link
              to="/posts/new"
              className="text-sm font-medium text-primary-700 hover:text-primary-900 transition-colors"
            >
              New Post
            </Link>
            <Link
              to="/reactions"
              className="text-sm font-medium text-primary-700 hover:text-primary-900 transition-colors"
            >
              Reactions
            </Link>
            <Link
              to="/devices"
              className="text-sm font-medium text-primary-700 hover:text-primary-900 transition-colors"
            >
              Devices
            </Link>
            {hpEnabled && (
              <Link
                to="/healthplanet"
                className="text-sm font-medium text-primary-700 hover:text-primary-900 transition-colors"
              >
                Health Planet
              </Link>
            )}
          </nav>

          <div className="flex items-center gap-2 sm:gap-4 ml-auto order-2 sm:order-3 shrink-0">
            <span className="text-sm text-primary-600">{user?.username}</span>
            <button
              onClick={handleLogout}
              className="text-sm font-medium text-primary-700 hover:text-primary-900 transition-colors px-3 py-1.5 rounded hover:bg-primary-50"
            >
              Logout
            </button>
          </div>
        </div>
      </div>
    </header>
  );
}
