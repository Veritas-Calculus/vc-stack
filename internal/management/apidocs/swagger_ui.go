package apidocs

// swaggerUIHTML is the Swagger UI page served at /api/docs.
// Uses the official Swagger UI CDN (unpkg) for zero-dependency serving.
const swaggerUIHTML = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>VC Stack API Documentation</title>
  <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5.11.0/swagger-ui.css">
  <style>
    * { margin: 0; padding: 0; box-sizing: border-box; }
    body {
      font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
      background: #1a1a2e;
      color: #e0e0e0;
    }
    .topbar-wrapper {
      display: flex;
      align-items: center;
      padding: 12px 24px;
      background: linear-gradient(135deg, #0f0c29 0%, #302b63 50%, #24243e 100%);
      border-bottom: 1px solid rgba(255,255,255,0.1);
    }
    .topbar-wrapper h1 {
      font-size: 20px;
      font-weight: 600;
      color: #fff;
      letter-spacing: 0.5px;
    }
    .topbar-wrapper .version-badge {
      background: rgba(99, 102, 241, 0.3);
      border: 1px solid rgba(99, 102, 241, 0.5);
      color: #a5b4fc;
      padding: 2px 10px;
      border-radius: 12px;
      font-size: 12px;
      margin-left: 12px;
    }
    .topbar-wrapper .links {
      margin-left: auto;
      display: flex;
      gap: 16px;
    }
    .topbar-wrapper .links a {
      color: #94a3b8;
      text-decoration: none;
      font-size: 13px;
      transition: color 0.2s;
    }
    .topbar-wrapper .links a:hover { color: #e0e0e0; }
    #swagger-ui .topbar { display: none; }
    #swagger-ui .swagger-ui {
      background: #1a1a2e;
    }
    #swagger-ui .swagger-ui .info .title { color: #e0e0e0; }
    #swagger-ui .swagger-ui .info .description p { color: #94a3b8; }
    #swagger-ui .swagger-ui .scheme-container {
      background: #16213e;
      box-shadow: none;
    }
    #swagger-ui .swagger-ui .opblock-tag { color: #e0e0e0; border-color: rgba(255,255,255,0.1); }
    #swagger-ui .swagger-ui .opblock { border-color: rgba(255,255,255,0.1); }
    #swagger-ui .swagger-ui .opblock .opblock-summary { border-color: rgba(255,255,255,0.1); }
    #swagger-ui .swagger-ui .opblock .opblock-summary-description { color: #94a3b8; }
    #swagger-ui .swagger-ui .opblock .opblock-section-header {
      background: rgba(255,255,255,0.03);
    }
    #swagger-ui .swagger-ui .opblock .opblock-section-header h4 { color: #e0e0e0; }
    #swagger-ui .swagger-ui .model-box { background: #16213e; }
    #swagger-ui .swagger-ui table thead tr th { color: #94a3b8; border-color: rgba(255,255,255,0.1); }
    #swagger-ui .swagger-ui table tbody tr td { color: #e0e0e0; border-color: rgba(255,255,255,0.1); }
    #swagger-ui .swagger-ui .response-col_status { color: #e0e0e0; }
    #swagger-ui .swagger-ui .response-col_description { color: #94a3b8; }
    #swagger-ui .swagger-ui .btn.authorize { color: #6366f1; border-color: #6366f1; }
    #swagger-ui .swagger-ui .btn.authorize svg { fill: #6366f1; }
    #swagger-ui .swagger-ui select { background: #16213e; color: #e0e0e0; border-color: rgba(255,255,255,0.2); }
    #swagger-ui .swagger-ui input { background: #16213e; color: #e0e0e0; border-color: rgba(255,255,255,0.2); }
    #swagger-ui .swagger-ui textarea { background: #16213e; color: #e0e0e0; border-color: rgba(255,255,255,0.2); }
    #swagger-ui .swagger-ui .model { color: #e0e0e0; }
    #swagger-ui .swagger-ui .model-title { color: #e0e0e0; }
    #swagger-ui .swagger-ui section.models { border-color: rgba(255,255,255,0.1); }
    #swagger-ui .swagger-ui section.models h4 { color: #e0e0e0; }
  </style>
</head>
<body>
  <div class="topbar-wrapper">
    <h1>VC Stack</h1>
    <span class="version-badge">API v1.0.0</span>
    <div class="links">
      <a href="/api/versions">Versions</a>
      <a href="/api/v1/openapi.json">OpenAPI JSON</a>
      <a href="/api/v1/openapi.yaml">OpenAPI YAML</a>
    </div>
  </div>
  <div id="swagger-ui"></div>
  <script src="https://unpkg.com/swagger-ui-dist@5.11.0/swagger-ui-bundle.js"></script>
  <script>
    SwaggerUIBundle({
      url: '/api/v1/openapi.json',
      dom_id: '#swagger-ui',
      deepLinking: true,
      presets: [
        SwaggerUIBundle.presets.apis,
        SwaggerUIBundle.SwaggerUIStandalonePreset
      ],
      layout: "BaseLayout",
      defaultModelsExpandDepth: 1,
      docExpansion: "list",
      filter: true,
      tryItOutEnabled: true,
      persistAuthorization: true,
      requestInterceptor: function(req) {
        // Auto-add token from localStorage if available.
        var token = localStorage.getItem('vc_api_token');
        if (token && !req.headers['Authorization']) {
          req.headers['Authorization'] = 'Bearer ' + token;
        }
        return req;
      }
    });
  </script>
</body>
</html>`
