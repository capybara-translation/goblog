# goblog Admin Panel (web-admin)

Single Page Application (SPA) for goblog administration.

## Tech Stack

| Category | Technology |
|----------|------------|
| Framework | React 19 |
| Language | TypeScript |
| Build Tool | Vite |
| Styling | Tailwind CSS |
| Routing | React Router |
| Testing | Vitest + Testing Library + MSW |
| Markdown Editor | @uiw/react-md-editor |
| Date Handling | date-fns |

## Development Commands

```bash
# Install dependencies
npm install

# Start development server (http://localhost:5173)
npm run dev

# Build for production
npm run build

# Preview build
npm run preview

# Lint
npm run lint

# Run tests
npm test

# Run tests (UI mode)
npm run test:ui

# Run tests (coverage)
npm run test:coverage
```

**Note:** When using the development server, API requests are proxied to the backend (default: `http://localhost:8080`). See the `proxy` configuration in `vite.config.ts`.

## Directory Structure

```
/src
  /api
    client.ts          # API client (fetch wrapper)
  /components
    Header.tsx         # Header component
    Layout.tsx         # Layout component
    MarkdownEditor.tsx # Markdown editor
    Modal.tsx          # Modal dialog
    PrivateRoute.tsx   # Authentication required route
    StatusBadge.tsx    # Published/Draft badge
    TagInput.tsx       # Tag input component
    TagList.tsx        # Tag list component
  /hooks
    useAuth.tsx        # Authentication hook
    useModal.tsx       # Modal management hook
  /mocks
    handlers.ts        # MSW mock handlers
    server.ts          # MSW server configuration
  /pages
    Login.tsx          # Login page
    PostEdit.tsx       # Post edit page
    PostList.tsx       # Post list page
  /utils
    date.ts            # Date formatting functions
  App.tsx              # Root component (routing definition)
  main.tsx             # Entry point
  index.css            # Global styles
```

## Features

### Authentication
- Login/Logout
- Session-based authentication (HttpOnly Cookie)
- CSRF protection (X-CSRF-Token header)

### Post Management
- Post list (filter by status and tags)
- Create/Edit posts (Markdown editor)
- Publish/Unpublish posts
- Pin/Unpin posts
- Delete posts
- Image upload (drag & drop supported)

### Keyboard Shortcuts
- `Ctrl/Cmd + S` - Save post

## Testing

### Testing Tools
- **Vitest** - Test runner
- **Testing Library** - UI testing
- **MSW (Mock Service Worker)** - API mocking

### Running Tests

```bash
# Run all tests
npm test

# Watch mode
npm test -- --watch

# Run specific test file
npm test -- src/components/Header.test.tsx

# UI mode (view test results in browser)
npm run test:ui

# Coverage report
npm run test:coverage
```

### Test File Placement
Test files are placed in the same directory as the files they test:
- `Header.tsx` → `Header.test.tsx`
- `useAuth.tsx` → `useAuth.test.tsx`

## Environment Variables

Configurable via `.env` file or environment variables:

| Variable | Description | Default |
|----------|-------------|---------|
| `VITE_BLOG_TITLE` | Blog title displayed in admin panel | `goblog` |

## Production Build

```bash
npm run build
```

Build output is generated in the `dist/` directory. It is embedded in the Go binary and served at the `/admin` path.

## Development Notes

1. **API Proxy**: During development, API requests are proxied to the backend via `vite.config.ts`. Start the backend (`make run`) first.

2. **Authentication**: For development, create a test user with `make seed` and login with `admin` / `password`.

3. **Hot Reload**: Vite hot reload is enabled. Changes are automatically reflected when you save files.
