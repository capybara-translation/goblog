# goblog
goblog is a simple blog system written in Go.

## Development Environment Setup

### 1. Clone the Repository

```bash
git clone https://github.com/capybara-translation/goblog.git
cd goblog
```

### 2. Install Dependencies

```bash
make deps
# or
go mod download
```

### 3. Configure Environment Variables (Optional)

In the development environment, you can manage environment variables with a `.env` file:

```bash
# Copy .env.example to create .env
cp .env.example .env

# Edit .env file to customize settings
# e.g., change port number or blog title
```

**Note:** The `.env` file is included in `.gitignore` and will not be committed to Git.

### 4. Database Setup and Test Data

```bash
# Reset database and seed test data
make reset

# Or run individually
make clean  # Delete database
make seed   # Seed test data
```

This creates test users and test posts for verification.

**Test User:**
- Username: `admin`
- Password: `password`

### 5. Starting the Server

#### Development Environment

**Method 1: Using .env file (Recommended)**

```bash
# Create and configure .env file
cp .env.example .env
# Edit .env file to customize settings

# Start server (automatically loads from .env)
make run
# or
go run cmd/goblog/main.go
```

**Method 2: Specify Environment Variables Directly**

```bash
# Start with environment variables
PORT=8000 BLOG_TITLE="Dev Blog" go run cmd/goblog/main.go
```

Access http://localhost:8080 in your browser to verify.

#### Production Environment (Using Environment Variables)

```bash
# Specify settings with environment variables
SECURE_COOKIE=true PASSWORD_POLICY=STRONG PORT=3000 BLOG_TITLE="My Awesome Blog" go run cmd/goblog/main.go

# Or export environment variables
export SECURE_COOKIE=true
export PASSWORD_POLICY=STRONG
export PORT=3000
export DATABASE_PATH=/var/lib/goblog/production.db
export BLOG_TITLE="My Awesome Blog"
go run cmd/goblog/main.go
```

**Available Environment Variables:**

| Variable | Description | Default |
|----------|-------------|---------|
| `PORT` | Server port number | `8080` |
| `SECURE_COOKIE` | Cookie Secure flag (set to `true` for HTTPS) | `false` |
| `PASSWORD_POLICY` | Password policy (`NONE` or `STRONG`) | `NONE` |
| `DATABASE_PATH` | Database file path | `data/goblog.db` |
| `BLOG_TITLE` | Blog title (displayed in header and page titles) | `goblog` |
| `BASE_URL` | Site base URL (used for sitemap, etc.) | `http://localhost:{PORT}` |
| `UPLOAD_DIR` | Upload file storage directory | `data/uploads` |
| `MAX_UPLOAD_SIZE` | Maximum upload file size (bytes) | `5242880` (5MB) |
| `TZ` | Timezone (e.g., `Asia/Tokyo`, `UTC`, `America/New_York`)<br>Used for date display | System timezone |

**About Password Policy:**

- `NONE`: No restrictions (for development/testing)
- `STRONG`: Strict policy (for production)
  - Minimum 15 characters
  - At least 1 uppercase letter
  - At least 1 lowercase letter
  - At least 1 number
  - At least 1 symbol

*Case-insensitive (`none`/`NONE`/`None`, `strong`/`STRONG`/`Strong` are all valid)

**About Timezone:**

Dates are displayed in ISO 8601 format (YYYY-MM-DD) with timezone abbreviation:
- e.g., `2024-12-26 (JST)`, `2024-12-25 (UTC)`, `2024-12-26 (EST)`

Set the TZ environment variable to display dates in the blog author's timezone:

```bash
# Start with Asia/Tokyo timezone
TZ=Asia/Tokyo make run

# Or add to .env file
echo "TZ=Asia/Tokyo" >> .env
```

**Notes:**
- For production (HTTPS-enabled server), always set `SECURE_COOKIE=true`. This ensures cookies are only sent over HTTPS connections.
- Setting `PASSWORD_POLICY=STRONG` is highly recommended for production.
- **For production, it is recommended to set system environment variables directly instead of using a `.env` file.**

### 6. Running Tests

```bash
# Run all tests
make test

# With verbose output
make test-v

# Check coverage
make test-cover
```

## Production Deployment

In production, use systemd for service management and nginx for reverse proxy.

### 1. Server Preparation

```bash
# Install required packages
sudo apt update
sudo apt install -y nginx certbot python3-certbot-nginx

# Set timezone (for Japan)
sudo timedatectl set-timezone Asia/Tokyo

# Verify settings
timedatectl

# Create user and directories for goblog
sudo useradd -r -s /bin/false goblog
sudo mkdir -p /opt/goblog/bin
sudo mkdir -p /var/lib/goblog/uploads
sudo chown -R goblog:goblog /var/lib/goblog
```

**Check available timezones:**
```bash
timedatectl list-timezones | grep -i tokyo
```

### 2. Build and Deploy Binaries

go-sqlite3 requires CGO, so build on the server.

```bash
# SSH to server
ssh user@your-server

# Install Go and build tools
sudo apt install -y golang-go build-essential git

# Install Node.js v24 (via NodeSource)
curl -fsSL https://deb.nodesource.com/setup_24.x | sudo -E bash -
sudo apt install -y nodejs

# Clone repository
git clone https://github.com/capybara-translation/goblog.git
cd goblog

# Build admin SPA (React)
cd web-admin
npm install
npm run build
cd ..

# Build binaries
go build -o bin/goblog cmd/goblog/main.go
go build -o bin/adduser cmd/adduser/main.go
go build -o bin/seed cmd/seed/main.go

# Deploy binaries
sudo mv bin/goblog bin/adduser bin/seed /opt/goblog/bin/
sudo chown root:root /opt/goblog/bin/goblog /opt/goblog/bin/adduser /opt/goblog/bin/seed
```

**Note**: Migration files, templates, and static files are embedded in the binary, so no separate copying is required.

### 3. Configure systemd Service

```bash
# Work from cloned repository (~/goblog)
cd ~/goblog

# Copy service file
sudo cp deploy/goblog.service /etc/systemd/system/

# Edit environment variables (change domain name and title)
sudo vim /etc/systemd/system/goblog.service

# Enable and start service
sudo systemctl daemon-reload
sudo systemctl enable goblog
sudo systemctl start goblog

# Check status
sudo systemctl status goblog

# Verify environment variables
sudo systemctl show goblog --property=Environment

# Check logs
sudo journalctl -u goblog -f
```

**Important Environment Variables:**

| Variable | Production Setting |
|----------|-------------------|
| `SECURE_COOKIE` | `true` (required) |
| `PASSWORD_POLICY` | `STRONG` (recommended) |
| `BASE_URL` | `https://your-domain.com` |
| `DATABASE_PATH` | `/var/lib/goblog/goblog.db` |
| `UPLOAD_DIR` | `/var/lib/goblog/uploads` |
| `BLOG_TITLE` | Your blog title |

### 4. Configure nginx

```bash
# Work from cloned repository (~/goblog)
cd ~/goblog

# Copy config file and change domain name
sudo cp deploy/nginx.conf /etc/nginx/sites-available/goblog
sudo vim /etc/nginx/sites-available/goblog  # Change your-domain.com to actual domain

# Enable site
sudo ln -s /etc/nginx/sites-available/goblog /etc/nginx/sites-enabled/

# Disable default site (optional)
sudo rm /etc/nginx/sites-enabled/default

# Test configuration
sudo nginx -t

# Reload nginx
sudo systemctl reload nginx
```

### 5. Obtain SSL Certificate (Let's Encrypt)

```bash
# Obtain SSL certificate with certbot (automatically adds SSL to nginx config)
sudo certbot --nginx -d your-domain.com

# Verify auto-renewal
sudo certbot renew --dry-run
```

**Note**: certbot automatically:
- Adds SSL certificate paths to nginx config
- Configures HTTP to HTTPS redirect

### 6. Create Admin User

```bash
# Create user on server (adduser deployed in step 2)
cd /opt/goblog
sudo -u goblog PASSWORD_POLICY=STRONG DATABASE_PATH=/var/lib/goblog/goblog.db ./bin/adduser
```

### 7. Verify Operation

```bash
# Health check
curl https://your-domain.com/api/v1/health

# Check sitemap
curl https://your-domain.com/sitemap.xml
```

### Troubleshooting

```bash
# Check goblog logs
sudo journalctl -u goblog -n 100

# Monitor goblog logs in real-time
sudo journalctl -u goblog -f

# Check nginx error logs
sudo tail -f /var/log/nginx/goblog_error.log
```

### Service Restart

| Change | Required Command |
|--------|-----------------|
| Unit file change | `sudo systemctl daemon-reload && sudo systemctl restart goblog` |
| Binary update | `sudo systemctl restart goblog` |
| Environment variable change (in Unit) | `sudo systemctl daemon-reload && sudo systemctl restart goblog` |
| nginx config change | `sudo nginx -t && sudo systemctl reload nginx` |

```bash
# After changing Unit file or environment variables
sudo systemctl daemon-reload
sudo systemctl restart goblog

# After updating binary only
sudo systemctl restart goblog

# After changing nginx config (test then reload)
sudo nginx -t && sudo systemctl reload nginx
```

## Available Make Commands

Common development commands are organized in the Makefile:

### Basic Commands

```bash
make help        # Show help
make run         # Start server
make stop        # Stop running server
make test        # Run tests
make test-v      # Run tests with verbose output
make test-cover  # Show test coverage
make clean       # Delete database and admin SPA build artifacts
make seed        # Seed test data
make reset       # Reset database and seed test data (also rebuilds admin SPA)
make build       # Build admin SPA and backend
make install     # Install binaries
make deps        # Download dependencies
```

### Admin SPA (React) Commands

```bash
make install-admin  # Install admin SPA npm dependencies
make build-admin    # Build admin SPA
make dev-admin      # Start admin SPA development server
make clean-admin    # Delete admin SPA build artifacts
```


## Directory Structure

```
/cmd/
  /adduser/main.go     # Admin user creation command
  /goblog/main.go      # Main application
  /seed/main.go        # Test data seeding command

/deploy/               # Deployment configuration
  goblog.service       # systemd Unit file
  nginx.conf           # nginx configuration

/internal/
  /auth/               # Authentication utilities
    session.go         # Session management
  /config/             # Configuration management
    config.go          # Load config from environment variables
  /db/                 # Database
    db.go              # DB connection and migrations
  /domain/             # Domain models
    post.go            # Post model
    user.go            # User model
  /http/               # HTTP layer
    router.go          # Routing configuration
    middleware.go      # Authentication and CSRF middleware
    handlers_admin.go  # Admin SPA serving
    handlers_api.go    # API handlers
    handlers_auth.go   # Authentication handlers
    handlers_image.go  # Image upload handlers
    handlers_public.go # Public page handlers
    handlers_sitemap.go # Sitemap handlers
  /markdown/           # Markdown processing
    markdown.go        # Markdown to HTML conversion
    dataline_extension.go # Line number extension
  /repo/               # Data access layer
    post_repo.go       # Post repository
    user_repo.go       # User repository
  /service/            # Business logic layer
    auth_service.go    # Authentication service
    post_service.go    # Post service
  /view/               # View related
    /static/           # Static files
      markdown.css     # Markdown styles
    /templates/        # HTML templates
      layout.html      # Common layout
      home.html        # Home page
      posts.html       # Post list
      post.html        # Post detail
      tags.html        # Tag list
      tag_posts.html   # Posts by tag
      notfound.html    # 404 page

/migrations/           # SQL migrations
  001_create_posts.sql # Posts table
  002_create_users.sql # Users table
  003_add_is_pinned.sql # Pin feature

/web-admin/            # Admin SPA (React)
  /src/
    /api/              # API client
    /components/       # Common components
    /hooks/            # Custom hooks
    /pages/            # Page components
    /mocks/            # Test mocks (MSW)
    /utils/            # Utilities
    App.tsx            # Root component
    main.tsx           # Entry point
```

## URL Design

### Public Pages

- `GET /` - Home page
- `GET /posts` - Post list (with pagination)
- `GET /posts/{slug}` - Post detail
- `GET /tags` - Tag list
- `GET /tags/{tag}` - Posts by tag (with pagination)
- `GET /sitemap.xml` - Sitemap

### Static Files

- `GET /static/*` - Static files like CSS
- `GET /uploads/*` - Uploaded images

### Admin SPA

- `GET /admin` - SPA entry point
- `GET /admin/*` - SPA fallback (client-side routing support)

### API (/api/v1)

**Public Endpoints:**
- `GET /api/v1/health` - Health check
- `POST /api/v1/auth/login` - Login

**Protected Endpoints (Authentication + CSRF Required):**
- `POST /api/v1/auth/logout` - Logout
- `GET /api/v1/auth/me` - Check login status
- `GET /api/v1/posts` - Get post list (`?status=draft|published&tag=tagname&limit=N&offset=N`)
- `POST /api/v1/posts` - Create post
- `GET /api/v1/posts/{id}` - Get post
- `PUT /api/v1/posts/{id}` - Update post
- `DELETE /api/v1/posts/{id}` - Delete post
- `POST /api/v1/posts/{id}/publish` - Publish post
- `POST /api/v1/posts/{id}/unpublish` - Unpublish post
- `POST /api/v1/posts/{id}/pin` - Pin post
- `POST /api/v1/posts/{id}/unpin` - Unpin post
- `GET /api/v1/tags` - Get tag list (`?status=draft|published`)
- `POST /api/v1/markdown/preview` - Markdown preview
- `POST /api/v1/images` - Image upload
