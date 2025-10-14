const http = require('http');
const url = require('url');

const PORT = process.env.PORT || 3000;
const SERVER_NAME = process.env.SERVER_NAME || 'backend2';

// Simple request counter for demonstration
let requestCount = 0;
const startTime = Date.now();

const server = http.createServer((req, res) => {
    requestCount++;
    const parsedUrl = url.parse(req.url, true);
    const path = parsedUrl.pathname;

    // Enable CORS
    res.setHeader('Access-Control-Allow-Origin', '*');
    res.setHeader('Access-Control-Allow-Methods', 'GET, POST, PUT, DELETE, OPTIONS');
    res.setHeader('Access-Control-Allow-Headers', 'Content-Type, Authorization');

    if (req.method === 'OPTIONS') {
        res.writeHead(200);
        res.end();
        return;
    }

    console.log(`[${new Date().toISOString()}] ${req.method} ${req.url} - Request #${requestCount}`);

    // Health check endpoint
    if (path === '/health') {
        res.writeHead(200, { 'Content-Type': 'application/json' });
        res.end(JSON.stringify({
            status: 'healthy',
            server: SERVER_NAME,
            uptime: Date.now() - startTime,
            requestCount: requestCount,
            timestamp: new Date().toISOString()
        }));
        return;
    }

    // Metrics endpoint (basic Prometheus format)
    if (path === '/metrics') {
        const uptime = (Date.now() - startTime) / 1000;
        const metrics = [
            `# HELP backend_requests_total Total number of requests`,
            `# TYPE backend_requests_total counter`,
            `backend_requests_total{server="${SERVER_NAME}"} ${requestCount}`,
            ``,
            `# HELP backend_uptime_seconds Server uptime in seconds`,
            `# TYPE backend_uptime_seconds gauge`,
            `backend_uptime_seconds{server="${SERVER_NAME}"} ${uptime}`,
            ``,
            `# HELP backend_info Server information`,
            `# TYPE backend_info gauge`,
            `backend_info{server="${SERVER_NAME}",version="1.0.0",port="${PORT}"} 1`,
            ``
        ].join('\n');

        res.writeHead(200, { 'Content-Type': 'text/plain' });
        res.end(metrics);
        return;
    }

    // API endpoint
    if (path === '/api/status') {
        res.writeHead(200, { 'Content-Type': 'application/json' });
        res.end(JSON.stringify({
            server: SERVER_NAME,
            message: 'API is working',
            requestId: requestCount,
            timestamp: new Date().toISOString(),
            headers: req.headers
        }));
        return;
    }

    // Load test endpoint
    if (path === '/api/load') {
        const delay = Math.floor(Math.random() * 100) + 50; // 50-150ms delay
        setTimeout(() => {
            res.writeHead(200, { 'Content-Type': 'application/json' });
            res.end(JSON.stringify({
                server: SERVER_NAME,
                message: 'Load test response',
                delay: delay,
                requestId: requestCount,
                timestamp: new Date().toISOString()
            }));
        }, delay);
        return;
    }

    // Root endpoint
    if (path === '/') {
        res.writeHead(200, { 'Content-Type': 'text/html' });
        res.end(`
      <!DOCTYPE html>
      <html>
        <head>
          <title>${SERVER_NAME} - Backend Service</title>
          <style>
            body { font-family: Arial, sans-serif; margin: 40px; }
            .container { max-width: 600px; margin: 0 auto; }
            .info { background: #f0f8ff; padding: 20px; border-radius: 8px; margin: 20px 0; }
            .endpoints { background: #f5f5f5; padding: 20px; border-radius: 8px; margin: 20px 0; }
            pre { background: #333; color: #fff; padding: 10px; border-radius: 4px; overflow-x: auto; }
          </style>
        </head>
        <body>
          <div class="container">
            <h1>${SERVER_NAME} Backend Service</h1>
            <div class="info">
              <h3>Server Information</h3>
              <p><strong>Server:</strong> ${SERVER_NAME}</p>
              <p><strong>Port:</strong> ${PORT}</p>
              <p><strong>Uptime:</strong> ${Math.floor((Date.now() - startTime) / 1000)} seconds</p>
              <p><strong>Requests Served:</strong> ${requestCount}</p>
            </div>
            <div class="endpoints">
              <h3>Available Endpoints</h3>
              <ul>
                <li><a href="/health">/health</a> - Health check</li>
                <li><a href="/metrics">/metrics</a> - Prometheus metrics</li>
                <li><a href="/api/status">/api/status</a> - API status</li>
                <li><a href="/api/load">/api/load</a> - Load test endpoint</li>
              </ul>
            </div>
          </div>
        </body>
      </html>
    `);
        return;
    }

    // 404 for unknown paths
    res.writeHead(404, { 'Content-Type': 'application/json' });
    res.end(JSON.stringify({
        error: 'Not Found',
        server: SERVER_NAME,
        path: path,
        timestamp: new Date().toISOString()
    }));
});

server.listen(PORT, '0.0.0.0', () => {
    console.log(`${SERVER_NAME} backend server running on port ${PORT}`);
    console.log(`Health check: http://localhost:${PORT}/health`);
    console.log(`Metrics: http://localhost:${PORT}/metrics`);
    console.log(`API status: http://localhost:${PORT}/api/status`);
});

// Graceful shutdown
process.on('SIGTERM', () => {
    console.log('Received SIGTERM, shutting down gracefully...');
    server.close(() => {
        console.log('Server closed');
        process.exit(0);
    });
});

process.on('SIGINT', () => {
    console.log('Received SIGINT, shutting down gracefully...');
    server.close(() => {
        console.log('Server closed');
        process.exit(0);
    });
});
