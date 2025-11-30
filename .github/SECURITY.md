# Security Policy

## Supported Versions

| Version | Supported          |
| ------- | ------------------ |
| Latest  | :white_check_mark: |

## Reporting a Vulnerability

If you discover a security vulnerability in codemap, please report it responsibly:

1. **Do not** open a public GitHub issue
2. Email the maintainer directly or use [GitHub's private vulnerability reporting](https://github.com/JordanCoin/codemap/security/advisories/new)
3. Include steps to reproduce the issue
4. Allow reasonable time for a fix before public disclosure

## Scope

codemap is a CLI tool that:
- Reads local files and directories
- Respects `.gitignore` patterns
- Does not make network requests (except for grammar downloads during build)
- Does not execute arbitrary code from scanned files

Security concerns would typically involve:
- Path traversal vulnerabilities
- Sensitive file exposure
- Malicious grammar injection (if using custom grammars)

Thank you for helping keep codemap secure!
