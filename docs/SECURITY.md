# Security Policy

## Supported Versions

`sharp` is pre-1.0. Security fixes are applied to the main branch unless a release branch is explicitly maintained.

## Reporting a Vulnerability

Please do not disclose suspected vulnerabilities in public issues before maintainers have had a chance to review them.

Preferred process:

1. Contact the maintainers privately when a security contact is available.
2. Include a concise description, affected command/tool, reproduction steps, and potential impact.
3. Avoid sharing sensitive real-world secrets, tokens, or production data.

If no private security contact is available yet, open a minimal public issue that says you have a security concern and ask for a private reporting channel. Do not include exploit details in that public issue.

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
