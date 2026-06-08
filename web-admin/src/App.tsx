import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import { AuthProvider } from './hooks/useAuth';
import { ModalProvider } from './hooks/useModal';
import { PrivateRoute } from './components/PrivateRoute';
import { Layout } from './components/Layout';
import { Login } from './pages/Login';
import { PostList } from './pages/PostList';
import { PostEdit } from './pages/PostEdit';
import { ReactionTypeList } from './pages/ReactionTypeList';

function App() {
  return (
    <BrowserRouter basename="/admin">
      <ModalProvider>
        <AuthProvider>
          <Routes>
          {/* Public routes */}
          <Route path="/login" element={<Login />} />

          {/* Protected routes */}
          <Route
            path="/"
            element={
              <PrivateRoute>
                <Layout />
              </PrivateRoute>
            }
          >
            <Route index element={<Navigate to="/posts" replace />} />
            <Route path="posts" element={<PostList />} />
            <Route path="posts/new" element={<PostEdit />} />
            <Route path="posts/:id/edit" element={<PostEdit />} />
            <Route path="reactions" element={<ReactionTypeList />} />
          </Route>

          {/* 404 */}
          <Route path="*" element={<Navigate to="/posts" replace />} />
          </Routes>
        </AuthProvider>
      </ModalProvider>
    </BrowserRouter>
  );
}

export default App;
