# Security

## Reporting a vulnerability

Report privately through [GitHub Security
Advisories](https://github.com/canhta/cred/security/advisories/new). Please do
not open a public issue for a vulnerability.

There is one maintainer. Expect an acknowledgement within a week and a fix
timeline in the acknowledgement rather than in advance of it.

## Why this file exists at this stage

Not for parity. Zero of five comparable Go repositories surveyed during
discovery ship a `SECURITY.md`, and most contributor-facing files are
deliberately deferred until a second contributor exists.

This one is different because of design law L8: **ingested content is
untrusted**. CRED's threat model includes stored prompt injection reaching
another agent through shared memory — the testing strategy rates that the
highest-severity case in the system. A product making a security claim needs a
disclosure address.

## What is in scope

- Access-control failures: a principal reading a claim outside
  `claim.acl ⊆ ⋂(evidence_i.acl)`, an expired grant that still admits, or an
  empty ACL treated as public.
- Side channels that distinguish *unauthorized* from *nonexistent* — in bytes,
  in counts, in timing, or in an error message.
- Stored prompt injection that escapes the data fence in `recall` output, or
  that reaches a model as instruction rather than as data.
- Supply-chain issues in the build: an unpinned artifact, a dependency that
  reintroduces cgo, or a model file that loads without its hash being checked.
- SQL injection, authentication bypass, and remote code execution, as usual.

## What is out of scope

- The absence of features listed under "What is not built yet" in the README.
  This build is read-only by design; a missing write-path control is a scope
  boundary, not a vulnerability.
- Denial of service through a self-hosted instance's own resource limits. CRED
  ships no default resource limits on purpose.
- Anything requiring an attacker to already control the `DATABASE_URL` or the
  model directory. Both are operator-supplied configuration.

## Supported versions

While CRED is `0.x`, only the latest release is supported. Minor bumps may
contain breaking changes to the MCP tool schema, the configuration, or the
database schema; patch bumps never do.
