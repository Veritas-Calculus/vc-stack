# Sentry Integration Guide

## Overview

VC Stack is integrated with Sentry for comprehensive error tracking and performance monitoring across all services.

## Sentry Server

- **URL**: `https://sentry.infra.plz.ac`
- **Project**: vc-stack

## Configuration

### 1. Environment Variables

Add the following environment variables to your deployment:

#### For vc-controller

```bash
# /etc/vc-stack/controller.env or systemd environment file
SENTRY_DSN=https://your-public-key@sentry.infra.plz.ac/project-id
SENTRY_ENVIRONMENT=production  # or staging, development
```

#### For vc-node

```bash
# /etc/vc-stack/node.env or systemd environment file
SENTRY_DSN=https://your-public-key@sentry.infra.plz.ac/project-id
SENTRY_ENVIRONMENT=production
```

### 2. Get Your DSN

1. Log in to Sentry: `https://sentry.infra.plz.ac`
2. Navigate to **Settings** → **Projects** → **vc-stack**
3. Go to **Client Keys (DSN)**
4. Copy the DSN string

### 3. Update Systemd Service Files

Edit `/etc/systemd/system/vc-controller.service`:

```ini
[Service]
EnvironmentFile=/etc/vc-stack/controller.env
Environment="SENTRY_DSN=https://your-dsn@sentry.infra.plz.ac/1"
Environment="SENTRY_ENVIRONMENT=production"
```

Reload and restart:

```bash
sudo systemctl daemon-reload
sudo systemctl restart vc-controller
sudo systemctl restart vc-node
```

## Features

### 1. Automatic Error Capture

All unhandled errors and panics are automatically captured and sent to Sentry with:

- Stack traces
- Request context (for HTTP errors)
- Environment information
- Server hostname
- Git commit and version

### 2. Performance Monitoring

Transaction tracing is enabled with 20% sampling rate for:

- HTTP requests
- Database queries
- Background jobs

### 3. Breadcrumbs

Automatic breadcrumb collection for debugging:

- HTTP requests (success and failure)
- Database operations
- User actions

### 4. Custom Error Reporting

Use the Sentry SDK in your code:

```go
import pkgsentry "github.com/Veritas-Calculus/vc-stack/pkg/sentry"

// Capture error with context
pkgsentry.CaptureError(err, map[string]string{
    "component": "network",
    "operation": "create_subnet",
}, map[string]interface{}{
    "subnet_id": subnetID,
    "network_id": networkID,
})

// Capture message
pkgsentry.CaptureMessage("Important event occurred",
    sentry.LevelWarning,
    map[string]string{"user_id": userID})

// Add breadcrumb
pkgsentry.AddBreadcrumb("database", "Query executed", map[string]interface{}{
    "table": "instances",
    "query_time_ms": 45,
})
```

## Monitoring Dashboard

Access your Sentry dashboard at: `https://sentry.infra.plz.ac`

### Key Metrics

- **Error rate**: Track error frequency over time
- **Affected users**: See how many users are impacted
- **Performance**: Monitor slow transactions
- **Releases**: Track errors by deployment version

## Best Practices

### 1. Use Environments

Always set `SENTRY_ENVIRONMENT` appropriately:

- `development` - Local development
- `staging` - Staging/test environment
- `production` - Production environment

### 2. Add Context

When capturing errors, always add relevant context:

```go
pkgsentry.CaptureError(err, map[string]string{
    "component": "compute",
    "instance_id": instanceID,
    "node": nodeName,
}, map[string]interface{}{
    "memory_mb": 2048,
    "vcpus": 2,
})
```

### 3. Set User Context

For user-facing operations:

```go
pkgsentry.SetUser(userID, username, email)
```

### 4. Filter Sensitive Data

The SDK is configured to automatically filter:

- Passwords
- API keys
- Tokens
- Credit card numbers

## Troubleshooting

### Errors Not Appearing in Sentry

1. Check DSN is correct:

   ```bash
   echo $SENTRY_DSN
   ```

2. Check logs for Sentry initialization:

   ```bash
   journalctl -u vc-controller | grep sentry
   ```

3. Verify network connectivity:

   ```bash
   curl https://sentry.infra.plz.ac
   ```

### High Volume of Events

Adjust sample rates in code if needed:

```go
pkgsentry.Init(pkgsentry.Config{
    DSN:              sentryDSN,
    SampleRate:       1.0,    // 100% of errors
    TracesSampleRate: 0.1,    // 10% of transactions
})
```

## Security

- DSN contains public key (safe to expose in client-side code)
- Never commit DSN to version control
- Use environment variables for configuration
- Sensitive data is filtered before sending

## Support

For Sentry-related issues:

1. Check logs: `journalctl -u vc-controller -f`
2. Verify configuration in Sentry dashboard
3. Review error details in Sentry UI
4. Contact platform team for access issues
