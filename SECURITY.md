# Security policy

## Reporting a vulnerability

Please do not open a public GitHub issue for security vulnerabilities.

Instead, report the issue privately through GitHub Security Advisories if available, or contact the maintainer through the GitHub profile linked from this repository.

Please include:

- A description of the issue
- Steps to reproduce
- Impact and affected versions, if known
- Any suggested mitigation

## Scope

Security reports are especially useful for:

- Unsafe handling of local files or paths
- Unexpected execution or browser automation behavior
- Malicious Markdown/Mermaid input that causes unintended effects
- Dependency vulnerabilities that affect normal glowm usage

## Non-goals

Rendering untrusted Markdown may involve browser-based Mermaid rendering. glowm should not be treated as a sandbox for hostile documents unless explicitly documented otherwise.
