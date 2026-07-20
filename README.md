# CRED

**C**laims — **R**ights — **E**vidence — **D**erivation

> A claim lives only while its evidence does.

Evidence-governed memory for AI agents.

---

## What this is

CRED is an open-source, self-hostable memory layer that agents connect to over
MCP. They retrieve what an organization already knows before starting work, and
contribute what they learn as they finish it.

Four ideas carry the whole design:

- **Claims** — the atomic unit of knowledge. Small, typed, independently
  expirable.
- **Rights** — access control evaluated at recall, failing closed. A claim
  derived from several sources inherits the **intersection** of their
  permissions, never the union.
- **Evidence** — no claim exists without a pointer to what produced it. Human
  attestation counts; orphan claims do not.
- **Derivation** — where a claim came from, and how its permissions were
  inherited, is always reconstructible.

### The difference

Other systems decide what to trust from **usage signal** — a claim becomes
trusted because it is retrieved and upvoted. That lags reality: a claim that was
true and became false stays trusted until enough negative signal accumulates.

CRED decides from **evidence**. When a source changes, every claim resting on it
is invalidated — no inference call, no waiting for signal, and the reason is a
diff rather than a score.

## Status

Pre-implementation. Discovery is complete and documented, including the evidence
that contradicts parts of the original thesis.

**Nothing is built yet.** The first deliverable is an experiment, not a feature:
whether retrieved memory outperforms plain long context at all. If it does not,
that is a finding about the category, and this project publishes it and stops.

## Documentation

- [Product requirements](docs/product/prd.md) — what to build, and the laws it
  must not violate
- [Research synthesis](docs/research/synthesis.md) — what discovery found, what
  it killed, and what survived
- [Decision log](docs/research/decision-log.md) — decisions with their reasoning
  and what each rules out

## License

Copyright © 2026 canhta.

Licensed under the [Apache License 2.0](LICENSE).
