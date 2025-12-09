# SonarQube Integration Guide

## Overview

VC Stack uses SonarQube for continuous code quality inspection and static analysis.

## SonarQube Configuration

### Project Details

- **Project Key**: `vc-stack`
- **Project Name**: VC Stack
- **Language**: Go

## Local Setup

### 1. Install SonarQube Scanner

#### macOS

```bash
brew install sonar-scanner
```

#### Linux

```bash
wget https://binaries.sonarsource.com/Distribution/sonar-scanner-cli/sonar-scanner-cli-5.0.1.3006-linux.zip
unzip sonar-scanner-cli-5.0.1.3006-linux.zip
sudo mv sonar-scanner-5.0.1.3006-linux /opt/sonar-scanner
export PATH=$PATH:/opt/sonar-scanner/bin
```

### 2. Configure Scanner

Create or edit `~/.sonar/sonar-scanner.properties`:

```properties
sonar.host.url=https://your-sonarqube-server.com
sonar.login=your-token-here
```

### 3. Get Authentication Token

1. Log in to SonarQube
2. Go to **My Account** → **Security** → **Generate Tokens**
3. Create a token for "vc-stack-analysis"
4. Save the token securely

## Running Analysis Locally

### Quick Analysis

```bash
make sonar
```

This will:

1. Run tests with coverage
2. Run golangci-lint
3. Upload results to SonarQube

### Manual Analysis

```bash
# 1. Run tests with coverage
go test -v -race -coverprofile=coverage.out -covermode=atomic ./...

# 2. Run linter
golangci-lint run --out-format checkstyle > golangci-lint-report.xml

# 3. Run sonar-scanner
sonar-scanner \
  -Dsonar.projectKey=vc-stack \
  -Dsonar.sources=cmd,internal,pkg \
  -Dsonar.host.url=https://your-sonarqube.com \
  -Dsonar.login=your-token
```

## CI/CD Integration

### GitHub Actions

The project includes a GitHub Actions workflow (`.github/workflows/code-quality.yml`) that:

1. **Runs on every push and PR**
2. **Executes tests with coverage**
3. **Runs golangci-lint**
4. **Uploads to SonarQube**
5. **Checks quality gate**
6. **Runs security scan**

### Required Secrets

Add these secrets to your GitHub repository:

1. **SONAR_TOKEN**
   - Go to GitHub repository → Settings → Secrets → New repository secret
   - Name: `SONAR_TOKEN`
   - Value: Your SonarQube token

2. **SONAR_HOST_URL**
   - Name: `SONAR_HOST_URL`
   - Value: `https://your-sonarqube-server.com`

## Quality Metrics

### Code Coverage

- **Target**: > 70%
- **Critical threshold**: > 60%

### Code Smells

- **Target**: < 50 per 1000 lines
- **Critical threshold**: < 100 per 1000 lines

### Duplications

- **Target**: < 3%
- **Critical threshold**: < 5%

### Bugs

- **Target**: 0
- **Critical threshold**: < 5

### Vulnerabilities

- **Target**: 0
- **Critical threshold**: 0

### Security Hotspots

- All hotspots should be reviewed

## Quality Gate

The default quality gate requires:

1. ✅ No new bugs
2. ✅ No new vulnerabilities
3. ✅ No new security hotspots
4. ✅ Coverage on new code > 80%
5. ✅ Duplicated lines on new code < 3%
6. ✅ Maintainability rating = A

## Configuration Files

### sonar-project.properties

Project-level configuration:

```properties
sonar.projectKey=vc-stack
sonar.projectName=VC Stack
sonar.sources=cmd,internal,pkg
sonar.exclusions=**/*_test.go,**/vendor/**,**/testdata/**,web/**
sonar.tests=.
sonar.test.inclusions=**/*_test.go
sonar.go.coverage.reportPaths=coverage.out
sonar.go.golangci-lint.reportPaths=golangci-lint-report.xml
```

### .golangci.yml

Linter configuration with rules matching SonarQube expectations:

- Enabled linters: errcheck, gosimple, govet, ineffassign, staticcheck, etc.
- Custom rules for Go best practices
- Exclusions for test files

## Make Commands

### Quality Checks

```bash
# Run all quality checks
make quality-check

# Run linters
make lint

# Run tests with coverage
make test-coverage

# Run security scan
make security-scan

# Run SonarQube analysis
make sonar

# Install development tools
make install-tools
```

## Viewing Results

### SonarQube Dashboard

1. Go to your SonarQube server
2. Find the **vc-stack** project
3. View:
   - **Overview**: Summary of all metrics
   - **Issues**: Code smells, bugs, vulnerabilities
   - **Measures**: Detailed metrics
   - **Code**: Browse source with annotations
   - **Activity**: Historical trends

### Pull Request Analysis

SonarQube will comment on PRs with:

- New issues introduced
- Coverage changes
- Quality gate status

## Best Practices

### 1. Fix Issues Before Merge

- Review SonarQube feedback on PRs
- Fix critical and major issues before merging
- Document why issues are marked as false positives

### 2. Maintain Coverage

- Write tests for new code
- Aim for >80% coverage on new code
- Focus on critical paths

### 3. Reduce Technical Debt

- Regularly review and fix code smells
- Refactor complex functions
- Remove duplicated code

### 4. Security First

- Fix all vulnerabilities immediately
- Review all security hotspots
- Keep dependencies updated

## Troubleshooting

### Analysis Fails

Check:

```bash
# Verify token
echo $SONAR_TOKEN

# Check connectivity
curl https://your-sonarqube.com/api/system/status

# Validate project configuration
sonar-scanner -X  # Debug mode
```

### Coverage Not Showing

Ensure:

```bash
# Coverage file exists
ls -lh coverage.out

# Coverage file is valid
go tool cover -func=coverage.out
```

### Linter Report Not Found

```bash
# Generate linter report
golangci-lint run --out-format checkstyle > golangci-lint-report.xml

# Verify file
cat golangci-lint-report.xml
```

## Integration with Development Workflow

### Pre-commit

Run quality checks before commit:

```bash
# Add to .git/hooks/pre-commit
make lint
make test
```

### Pre-push

Run full quality check before push:

```bash
# Add to .git/hooks/pre-push
make quality-check
```

### CI Pipeline

```yaml
# Runs on every push and PR
1. Build
2. Test with coverage
3. Lint
4. Security scan
5. SonarQube analysis
6. Quality gate check
```

## Advanced Configuration

### Custom Quality Profiles

Create custom quality profiles in SonarQube UI:

1. Go to **Quality Profiles**
2. Select **Go**
3. Create new profile based on "Sonar way"
4. Activate additional rules
5. Set as default for vc-stack

### Project Branches

SonarQube can analyze different branches:

- `main`: Production code
- `develop`: Development branch
- Feature branches: Automatic PR analysis

## Support

For SonarQube issues:

1. Check logs: `sonar-scanner -X`
2. Verify project configuration
3. Review quality gate conditions
4. Check GitHub Actions logs

## References

- [SonarQube Documentation](https://docs.sonarqube.org/)
- [SonarQube Go Plugin](https://docs.sonarqube.org/latest/analysis/languages/go/)
- [golangci-lint](https://golangci-lint.run/)
