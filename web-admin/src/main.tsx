import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import './index.css'
import App from './App'

const BLOG_TITLE = import.meta.env.VITE_BLOG_TITLE || 'goblog'

// Set document title
document.title = `${BLOG_TITLE} 管理画面`

const rootElement = document.getElementById('root')
if (!rootElement) {
  throw new Error('Root element not found')
}

createRoot(rootElement).render(
  <StrictMode>
    <App />
  </StrictMode>,
)
