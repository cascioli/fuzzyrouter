# Security Policy

## Supported Versions

| Version | Supported |
|---------|-----------|
| latest (`main`) | Yes |

Only the latest commit on `main` receives security fixes.

## Reporting a Vulnerability

**Do not open a public GitHub issue for security vulnerabilities.**

Report via email: `rent.dev.commercial@gmail.com`

Include:
- Description of the vulnerability
- Steps to reproduce
- Potential impact
- Suggested fix (optional)

Expected response within **72 hours**. A patch will be released and the reporter credited (unless anonymity is requested).

## Threat Model

FuzzyRouter is a **redirect-only** service. It makes no outbound requests, holds no state, and touches no database. The attack surface is intentionally narrow:

| Component | Notes |
|-----------|-------|
| HTTP input | Only `Host` header and request path are processed |
| Config | Read once at startup from a mounted file or ENV — no runtime writes |
| Redirects | Targets are constructed from a fixed allow-list (`subdomains`) + fixed `base_domain` — no open redirect possible |
| Logging | Logs `remote_addr`, `method`, `subdomain`, `user_agent` — no request body, no credentials |

## Known Constraints

**Open redirect is not possible by design.** The redirect target is always `<matched_subdomain>.<base_domain>`, where `matched_subdomain` must exist in the configured allow-list. Arbitrary targets are never accepted.

**Path and query string are preserved** in redirects. If a downstream service is vulnerable to path-based attacks, FuzzyRouter will forward them — it does not sanitize paths.

**No TLS termination.** FuzzyRouter speaks plain HTTP and relies on a reverse proxy (nginx, Traefik, Caddy) for TLS. Run it behind a proxy in production; never expose port 8080 directly to the internet.

**`X-Forwarded-Proto` is trusted unconditionally.** If a client can reach FuzzyRouter directly (bypassing the proxy), it can force `https://` in the redirect target. Restrict network access to prevent direct client connections.

## Secure Deployment Checklist

- [ ] FuzzyRouter is not exposed directly on the internet — reverse proxy handles TLS
- [ ] `config.yaml` is mounted read-only (`:ro` volume)
- [ ] No sensitive values committed to the repository (`config.yaml` is gitignored)
- [ ] `FUZZY_REDIRECT_CODE: 302` during initial rollout (avoids browser-cached bad redirects)
- [ ] `match_threshold` tuned high enough to prevent low-confidence redirects (≥ 0.6 recommended in production)
