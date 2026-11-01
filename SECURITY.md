# Security Policy

## Overview

Thémis is an open-source, self-hosted personal finance management application.

Because Thémis processes sensitive financial information, security, data integrity, and privacy are core project priorities.

The project aims to follow secure development practices, including automated security checks, dependency vulnerability scanning, access controls, and regression testing.

However, no software can guarantee complete security.

## Supported Versions

Thémis is currently under active development.

Security fixes are generally targeted at the latest version of the `main` branch.

| Version | Security Support |
|---|---|
| Latest `main` | Supported |
| Older commits and releases | Not guaranteed |
| Forks and modified deployments | Maintained by their respective operators |

Security fixes may require updating the application, its dependencies, or its deployment configuration.

## Reporting a Vulnerability

**Please do not disclose security vulnerabilities through public GitHub issues, pull requests, or discussions before coordinated disclosure.**

If you discover a potential security vulnerability, please report it privately using GitHub's security advisory mechanism:

[Report a vulnerability privately](https://github.com/Madi-Toumani-benmohamed/Themis-gestion-finance/security/advisories/new)

This reporting method is available when private vulnerability reporting is enabled in the repository settings.

If the reporting form is unavailable, please avoid publishing technical exploit details and request an alternative private communication channel from the maintainer.

### Information to include

Where possible, provide:

- A description of the vulnerability.
- The affected component or functionality.
- The application version or commit SHA.
- Steps required to reproduce the issue.
- The expected and observed behavior.
- A description of the potential security impact.
- A proof of concept using synthetic data, if applicable.
- Any suggested mitigation.

Do not include real banking statements, financial information, passwords, API keys, authentication tokens, or other sensitive user data.

## Disclosure and Response

Security reports will be reviewed on a best-effort basis.

Thémis is an independently maintained personal project. Response and resolution times are not guaranteed.

When a report is confirmed, the maintainer may:

1. Assess the severity and scope of the vulnerability.
2. Develop and test a remediation.
3. Release or merge the security fix.
4. Publish a GitHub Security Advisory when appropriate.
5. Coordinate public disclosure after a fix or suitable mitigation becomes available.

Responsible, good-faith security research is appreciated.

## Security Architecture

Thémis includes several security controls intended to protect financial data and application access.

### Authentication and Authorization

- Authenticated access to protected API endpoints.
- User-specific access controls.
- Ownership checks for financial records.
- Administrative authorization for sensitive configuration operations.

### Financial Data Integrity

- Transaction and statement validation.
- Financial reconciliation controls.
- Auditable financial data modifications.
- Exact monetary calculations in supported financial workflows.
- Regression tests for critical operations.

### Application Security

The development workflow includes security-oriented checks, such as:

- Static application security testing (SAST).
- Software composition and dependency vulnerability scanning.
- Go vulnerability checks.
- Automated backend and frontend tests.
- Browser-based integration and end-to-end tests.

The availability and execution of individual checks may vary by workflow and branch.

Passing automated security checks does not guarantee the absence of vulnerabilities.

### AI and External Services

Thémis can interact with optional AI providers and memory services.

Depending on the selected configuration, some financial data may be transmitted to external AI services.

Users should review provider privacy policies, retention practices, and deployment settings before enabling these integrations.

Local AI configurations can reduce reliance on third-party services, but do not eliminate all security risks.

## Deployment Security

Thémis is designed primarily for self-hosted environments.

Operators are responsible for securing their infrastructure.

Recommended practices include:

- Keep the operating system, containers, and dependencies updated.
- Use strong, unique credentials.
- Restrict access to PostgreSQL and internal services.
- Do not expose database, memory, or AI service ports directly to the public internet.
- Use HTTPS when accessing Thémis over untrusted networks.
- Store secrets outside the Git repository.
- Restrict access to environment files and database backups.
- Maintain encrypted backups where appropriate.
- Test backup restoration procedures.
- Apply the principle of least privilege.
- Monitor security advisories and update supported dependencies.

## Sensitive Data

Never commit the following to the repository:

- Real bank statements.
- Production database dumps or backups.
- Personal financial transaction exports.
- `.env` files containing credentials.
- API keys or authentication tokens.
- Private keys or certificates containing private material.
- Real user datasets.
- Sensitive diagnostic logs.

Synthetic datasets should be used for automated testing and demonstrations.

If sensitive information is accidentally committed, removing the file in a later commit is not sufficient to remove it from Git history.

Affected secrets should be rotated, and the repository history and exposure should be assessed.

## Scope

Security issues of particular interest include:

- Authentication bypass.
- Broken access control or cross-user data exposure.
- SQL injection.
- Cross-site scripting (XSS).
- Cross-site request forgery (CSRF), where applicable.
- Server-side request forgery (SSRF).
- Sensitive information disclosure.
- Insecure handling of financial records.
- Unauthorized modification or deletion of transactions.
- Secrets exposed through application logs or API responses.
- AI integration vulnerabilities affecting confidentiality or authorization.
- Container or deployment configuration weaknesses.

## Out of Scope

The following are generally not considered project security vulnerabilities unless they demonstrate a concrete security impact:

- Vulnerabilities exclusively affecting unsupported, outdated versions.
- Findings without a reproducible security impact.
- Social engineering directed at the maintainer.
- Issues requiring an already fully compromised host without additional impact.
- Denial-of-service testing against infrastructure without prior authorization.

## Security Research Guidelines

Please:

- Use your own test environment.
- Use synthetic financial data.
- Avoid accessing information belonging to others.
- Avoid destructive tests against systems you do not control.
- Do not publicly disclose exploitable vulnerabilities before reasonable coordination.
- Never include confidential data in public reports.

## Disclaimer

Thémis is provided as open-source software without any guarantee of fitness for a particular purpose or complete protection against security threats.

Users deploying Thémis are responsible for evaluating its suitability for their environment and implementing appropriate infrastructure safeguards.

For additional information, consult the project documentation and deployment configuration.
