import { Link, useNavigate } from 'react-router-dom';
import { useAuth } from '../hooks/useAuth';

const BLOG_TITLE = import.meta.env.VITE_BLOG_TITLE || 'goblog';

export function Header() {
  const { user, logout } = useAuth();
  const navigate = useNavigate();

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
        <div className="flex justify-between items-center h-16">
          {/* Left side - App title and navigation */}
          <div className="flex items-center gap-8">
            <Link
              to="/posts"
              className="text-xl font-sans font-bold text-primary-900 hover:text-primary-700 transition-colors"
            >
              {BLOG_TITLE} 管理画面
            </Link>
            <nav className="flex gap-4">
              <Link
                to="/posts"
                className="text-sm font-medium text-primary-700 hover:text-primary-900 transition-colors"
              >
                記事一覧
              </Link>
              <Link
                to="/posts/new"
                className="text-sm font-medium text-primary-700 hover:text-primary-900 transition-colors"
              >
                新規作成
              </Link>
            </nav>
          </div>

          {/* Right side - User info and logout */}
          <div className="flex items-center gap-4">
            <span className="text-sm text-primary-600">{user?.username}</span>
            <button
              onClick={handleLogout}
              className="text-sm font-medium text-primary-700 hover:text-primary-900 transition-colors px-3 py-1.5 rounded hover:bg-primary-50"
            >
              ログアウト
            </button>
          </div>
        </div>
      </div>
    </header>
  );
}
