# Security Best Practices for VC Stack

## ⚠️ Critical Security Warnings

This document outlines essential security practices for deploying and operating VC Stack in production environments.

## Table of Contents

- [Authentication & Authorization](#authentication--authorization)
- [Secrets Management](#secrets-management)
- [Database Security](#database-security)
- [Network Security](#network-security)
- [Configuration Security](#configuration-security)
- [Monitoring & Auditing](#monitoring--auditing)
- [Security Checklist](#security-checklist)

## Authentication & Authorization

### Default Credentials

**⚠️ CRITICAL: Change all default credentials before production deployment!**

The following default credentials are provided for development/testing only:

- **Admin User**: `admin` / `admin123`
- **Database**: `vcstack` / `vcstack123`
- **JWT Secret**: `your-super-secret-jwt-key-change-in-production`

**Actions Required:**

1. **Change admin password immediately after first login**
2. **Generate strong, unique passwords** (minimum 16 characters, mix of uppercase, lowercase, numbers, and symbols)
3. **Use a password manager** to store credentials securely

### JWT Secret Configuration

Generate a strong JWT secret using:

```bash
# Generate a 64-byte random secret
openssl rand -base64 64

# Or use /dev/urandom
head -c 64 /dev/urandom | base64
```

**Configuration:**

```yaml
identity:
  jwt:
    secret: <YOUR_GENERATED_SECRET_HERE>
    access_token_expires_in: 24h
    refresh_token_expires_in: 168h
```

**Best Practices:**

- Use unique secrets for each environment (dev, staging, production)
- Rotate JWT secrets periodically (recommended: every 90 days)
- Never commit secrets to version control
- Use environment variables or secret management tools

### Password Policies

Enforce strong password policies:

- Minimum length: 12 characters
- Require: uppercase, lowercase, numbers, and special characters
- Enable password expiration (e.g., 90 days)
- Prevent password reuse (last 5 passwords)
- Implement account lockout after failed attempts

## Secrets Management

### Production Secrets Storage

**Never store secrets in:**

- Configuration files committed to Git
- Docker Compose files in repositories
- Environment variables in CI/CD logs
- Plain text files on servers

**Recommended Solutions:**

1. **HashiCorp Vault** - Enterprise-grade secrets management
2. **AWS Secrets Manager** - For AWS deployments
3. **Azure Key Vault** - For Azure deployments
4. **Kubernetes Secrets** - For K8s deployments
5. **Docker Secrets** - For Docker Swarm

### Environment Variables

Use `.env` files for local development only:

```bash
# .env (add to .gitignore)
DATABASE_PASSWORD=your_secure_password
IDENTITY_JWT_SECRET=your_jwt_secret
```

**Load in Docker Compose:**

```yaml
services:
  vc-controller:
    env_file:
      - .env
```

## Database Security

### Connection Security

**Production Configuration:**

```yaml
database:
  host: db.example.com
  port: 5432
  name: vcstack
  username: vcstack_app
  password: ${DATABASE_PASSWORD}  # From environment or secrets manager
  sslmode: verify-full  # ⚠️ REQUIRED for production
  # SSL certificate configuration
  sslrootcert: /path/to/ca.crt
  sslcert: /path/to/client.crt
  sslkey: /path/to/client.key
```

**SSL/TLS Requirements:**

1. Use `sslmode: require` at minimum
2. Prefer `verify-full` for certificate validation
3. Use separate database credentials for each service
4. Implement least-privilege access control

### Database Hardening

- Disable remote root login
- Use strong, unique passwords for all database users
- Implement IP whitelisting for database access
- Enable audit logging
- Regular security updates and patching
- Encrypt data at rest
- Regular backups with encryption

## Network Security

### TLS/HTTPS Configuration

**Always use HTTPS in production:**

```nginx
server {
    listen 443 ssl http2;
    server_name vcstack.example.com;

    ssl_certificate /path/to/fullchain.pem;
    ssl_certificate_key /path/to/privkey.pem;

    # Modern SSL configuration
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers HIGH:!aNULL:!MD5;
    ssl_prefer_server_ciphers on;

    # Security headers
    add_header Strict-Transport-Security "max-age=31536000; includeSubDomains" always;
    add_header X-Frame-Options DENY;
    add_header X-Content-Type-Options nosniff;
    add_header X-XSS-Protection "1; mode=block";
}
```

### Firewall Configuration

Restrict network access:

```bash
# Allow only necessary ports
# Controller API
ufw allow 8080/tcp

# PostgreSQL (internal only)
ufw allow from 10.0.0.0/8 to any port 5432

# Deny all other traffic
ufw default deny incoming
ufw default allow outgoing
```

### Internal Network Isolation

- Use private networks for inter-service communication
- Implement network segmentation
- Use VPCs or VLANs to isolate environments
- Deploy Web Application Firewall (WAF)

## Configuration Security

### File Permissions

Set restrictive permissions on configuration files:

```bash
# Configuration files
chmod 600 /etc/vc-stack/vc-controller.yaml
chown vcstack:vcstack /etc/vc-stack/vc-controller.yaml

# SSL certificates
chmod 600 /etc/ssl/private/vcstack.key
chown root:ssl-cert /etc/ssl/private/vcstack.key
```

### Configuration Management

1. **Use configuration templates** with environment-specific values
2. **Separate secrets** from configuration
3. **Validate configuration** before deployment
4. **Version control** configuration templates (without secrets)
5. **Audit configuration changes**

### Example Secure Configuration Pattern

```yaml
# vc-controller.yaml
database:
  host: ${DB_HOST}
  port: ${DB_PORT}
  name: ${DB_NAME}
  username: ${DB_USER}
  password: ${DB_PASSWORD}  # From secret manager
  sslmode: verify-full
```

```bash
# Load from environment or secret manager
export DB_HOST="db.example.com"
export DB_PASSWORD=$(vault kv get -field=password secret/vcstack/db)
```

## Monitoring & Auditing

### Security Monitoring

Implement comprehensive logging and monitoring:

1. **Authentication Events**
   - Failed login attempts
   - Password changes
   - Token generation and validation

2. **Authorization Events**
   - Access denials
   - Permission changes
   - Privilege escalations

3. **System Events**
   - Configuration changes
   - Service restarts
   - Error conditions

### Log Management

- **Centralize logs** using ELK Stack, Splunk, or similar
- **Enable audit logging** for database operations
- **Set up alerts** for suspicious activities
- **Retain logs** according to compliance requirements
- **Encrypt logs** in transit and at rest

### Security Scanning

Regular security assessments:

```bash
# Dependency scanning
go list -json -m all | docker run --rm -i sonatypeiq/nancy:latest sleuth

# Container scanning
trivy image vcstack/controller:latest

# Code scanning
gosec ./...
```

## Security Checklist

### Pre-Production Deployment

- [ ] Changed all default passwords and credentials
- [ ] Generated strong, unique JWT secret
- [ ] Enabled SSL/TLS for all network communications
- [ ] Configured SSL mode for database connections
- [ ] Implemented secrets management solution
- [ ] Set restrictive file permissions on configuration files
- [ ] Configured firewall rules
- [ ] Enabled audit logging
- [ ] Set up security monitoring and alerts
- [ ] Performed security scanning (dependencies, containers, code)
- [ ] Implemented backup and disaster recovery procedures
- [ ] Documented security procedures and incident response plan

### Regular Maintenance

- [ ] Rotate credentials and secrets (every 90 days)
- [ ] Apply security updates and patches
- [ ] Review access logs for anomalies
- [ ] Update SSL/TLS certificates before expiration
- [ ] Conduct security audits and penetration testing
- [ ] Review and update firewall rules
- [ ] Test backup and recovery procedures
- [ ] Update documentation

## Incident Response

### Security Incident Procedure

1. **Identify and contain** the incident
2. **Assess the impact** and scope
3. **Preserve evidence** for investigation
4. **Eradicate** the threat
5. **Recover** services
6. **Document** lessons learned
7. **Update** security measures

### Emergency Contacts

Maintain a list of emergency contacts:

- Security team
- System administrators
- Database administrators
- Network operations
- Management

## Additional Resources

- [OWASP Security Guidelines](https://owasp.org/)
- [CIS Benchmarks](https://www.cisecurity.org/cis-benchmarks/)
- [NIST Cybersecurity Framework](https://www.nist.gov/cyberframework)
- [Go Security Best Practices](https://github.com/OWASP/Go-SCP)

## Reporting Security Vulnerabilities

If you discover a security vulnerability in VC Stack, please report it to:

**Email:** <security@vcstack.com> (if available)

**Please do not:**

- Disclose the vulnerability publicly
- Test the vulnerability on production systems
- Access or modify other users' data

We appreciate responsible disclosure and will acknowledge your report within 48 hours.

---

**Last Updated:** January 2026

**Version:** 1.0

**Review Schedule:** Quarterly
