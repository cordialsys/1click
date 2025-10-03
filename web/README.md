# Treasury Panel Web UI

This is a simple NextJS-based web interface for the Treasury Panel server that provides a user-friendly way to activate and configure a treasury node.

## Features

- Step-by-step activation wizard
- API key configuration
- Binary downloads
- Network setup via Netbird
- Backup key configuration
- OTEL (observability) setup
- Real-time status updates

## Building the UI

To build the static website:

```bash
./build-web.sh
```

This will:
1. Install Node.js dependencies (if needed)
2. Build the NextJS application as static files
3. Output files to `web/out/` directory

## Development

For development with hot reload:

```bash
cd web
pnpm install
pnpm run dev
```

The development server will run on http://localhost:3000

### API Host Configuration

By default, the web UI makes API calls to the same host it's served from. For development, you can override this:

1. Copy the example environment file:
   ```bash
   cp .env.local.example .env.local
   ```

2. Set the API host in `.env.local`:
   ```bash
   NEXT_PUBLIC_API_HOST=http://localhost:8080
   ```

This allows you to run the Next.js dev server on port 3000 while connecting to a Go server running on port 8080.

## Integration

The Go server automatically serves static files from `web/out/` at the root path `/`. The web interface makes API calls to the existing `/v1/activate/*` endpoints.

When you run the panel server, the web UI will be available at the server's address (e.g., http://localhost:8080).

## API Endpoints Used

- `POST /v1/activate/api-key` - Configure API key
- `POST /v1/activate/binaries` - Download required binaries
- `POST /v1/activate/network` - Setup network connection
- `POST /v1/activate/backup` - Configure backup keys
- `POST /v1/activate/otel` - Configure observability