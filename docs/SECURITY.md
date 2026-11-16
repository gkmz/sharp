# Security Policy

## Supported Versions

`sharp` is pre-1.0. Security fixes are applied to the main branch unless a release branch is explicitly maintained.

## Reporting a Vulnerability

Please do not disclose suspected vulnerabilities in public issues before maintainers have had a chance to review them.

Preferred process:

1. Open a [private vulnerability report](https://github.com/gkmz/sharp/security/advisories/new) on GitHub. Only repository maintainers and the reporter can access the report while it is being reviewed.
2. Include a concise description, affected command/tool and version, reproduction steps, and potential impact.
3. Avoid sharing sensitive real-world secrets, tokens, or production data. Use minimal test data whenever possible.

If GitHub's private reporting form is temporarily unavailable, open a minimal [public issue](https://github.com/gkmz/sharp/issues/new/choose) asking for a private maintainer channel. Do not include vulnerability details, exploit code, secrets, or sensitive data in that issue.

## Security Scope

Examples in scope:

- Command execution or file access behavior that exceeds user intent.
- Unsafe handling of secrets in options, output, logs, or clipboard paths.
- Parser behavior that can crash the process on untrusted input.
- Dependency vulnerabilities that materially affect `sharp`.

Examples generally out of scope:

- Tool output that reflects user-provided input by design.
- Incorrect results caused by invalid user input when an error is returned.
- Vulnerabilities in a user's shell environment or clipboard provider outside `sharp`.
