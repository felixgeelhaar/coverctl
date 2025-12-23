# Security Policy

## Supported Versions

| Version | Supported          |
| ------- | ------------------ |
| 0.2.x   | :white_check_mark: |
| < 0.2   | :x:                |

## Reporting a Vulnerability

If you discover a security vulnerability in coverctl, please report it responsibly:

1. **Do not** open a public GitHub issue for security vulnerabilities
2. Email the maintainer directly or use GitHub's private vulnerability reporting
3. Include as much detail as possible:
   - Description of the vulnerability
   - Steps to reproduce
   - Potential impact
   - Suggested fix (if any)

## Response Timeline

- **Initial Response**: Within 48 hours
- **Status Update**: Within 7 days
- **Fix Timeline**: Depends on severity
  - Critical: Within 24-48 hours
  - High: Within 7 days
  - Medium/Low: Next regular release

## Security Best Practices

When using coverctl:

- Keep coverctl updated to the latest version
- Review `.coverctl.yaml` before running in CI pipelines
- Avoid storing sensitive information in coverage reports
- Use the `--config` flag to specify trusted configuration files

## Scope

This security policy covers:
- The coverctl CLI tool
- The coverctl GitHub Action
- Official documentation and examples

Third-party integrations and forks are outside the scope of this policy.
