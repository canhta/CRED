# Demand interview 001 — Seta International

- **Date:** 2026-07-20
- **Respondent:** the founder of this project, describing their own
  organization
- **Org:** Seta International, ~300 people, software services
- **Sourcing channel:** self
- **Bias, stated plainly:** this is the **most biased source available**. The
  respondent is building CRED. Founders recall incidents supporting their
  thesis more readily than the weeks in which nothing went wrong. Counted as
  1 of the target n = 12–15, and never as confirmation.

Instrument: [demand-test.md](../../demand-test.md).

---

## Q1 — Who uses agents, for what

**~300 people, across the entire SDLC** — requirements writing, coding, test
generation, planning, slides, spikes, proof-of-concept work.

Wider than the design assumed. CRED's PRD and evaluation both target
engineering work specifically; this organization's agent usage extends well
past code into requirements and planning artifacts.

## Q2 — How agents learn what is not in the code

Five stores, in the respondent's own order: **ADRs, Claude memory, `docs/`
folders, Confluence, and verbal exchange.**

This is the strongest pro-thesis observation in the interview. Knowledge about
how the organization works is spread across five locations with different
access models, different freshness, and one of them — verbal — with no
persistence at all.

Recorded as an observation, not as a validated pain. Fragmentation is
consistent with the thesis but does not by itself establish that anyone is
paying a cost for it.

## Q3 — Who maintains it, when last updated

> "cái này git tracking được rồi mà"

The question was poorly aimed and the respondent said so. Staleness of
git-tracked files is already observable; it is not a mystery requiring a
product. The question should have asked whether anyone *acts* on that
observability, and whether the non-git stores (Confluence, verbal, Claude
memory) have any equivalent.

**Unanswered.** Re-ask.

## Q4–Q7 — Evidence of pain

> "agent thường sẽ là ở chế độ hỗ trợ chứ không phải thay thế"

**Agents assist; they do not replace.** A human is in the loop on the output.

This materially weakens the pain the product is built around. If a person
reviews what the agent produces, an agent working from stale or contradictory
context costs **review time**, not a production incident. Lower stakes imply
lower willingness to pay, and imply the failure mode CRED prevents is
routinely caught by a control that already exists.

> "phần Bằng chứng đau, không gọi tên nó đang sa đà vào một cái gì đó lệch
> hướng"

**The respondent judged this entire question block to be aimed at the wrong
thing.** Recorded as instrument feedback: the "agent confidently did something
the team had decided against" framing does not describe the shape of the
problem in this organization. The questions presumed a failure mode and then
searched for instances of it.

No incident was reported. Under the pre-registered reading, absence of a
recalled incident is a finding, not a gap to be filled by asking harder.

## Q8 — Would you use a hosted version? **Load-bearing**

> "có và không, chúng tôi thì có để bảo vệ tài nguyên cho công ty, nhưng nếu
> dự án đủ lớn chúng tôi hoàn toàn có thể cân nhắc cloud để phát triển nhanh"

Both. Self-hosting is the **default preference** to protect company resources,
but it **yields to speed** when a project is large enough to justify cloud.

## Q9 — What specifically stops you? **Load-bearing**

> "thường là chi phí, lòng tin, bảo mật, và nó có đáng không"

Cost, trust, security, and whether it is worth it. Cost appears **first** and
worth-it appears **last** — data residency is one of four factors, and is not
the one named first.

---

## Reading against the pre-registration

`demand-test.md` registered this in advance:

> If the answer is cost or usefulness rather than data residency, the premise
> fails and the direction should be revisited.

The answer given is **cost, trust, security, and whether it is worth it.**
Security is present; it is neither first nor alone, and the sovereignty
preference is explicitly conditional on project size.

**D-004's premise does not survive this cleanly.** Its load-bearing sentence
claims the binding constraint on adoption is that the system must run on the
organization's own infrastructure. What this interview describes is a
**default preference that trades away against speed** — which makes
sovereignty a tiebreaker rather than a wedge. A tiebreaker cannot carry the
differentiation D-004 assigns to it.

This is one interview, and n = 1 falsifies nothing. But the direction matters
more than usual here: **the most biased available source returned
disconfirming evidence.** Biased sources normally confirm.

## What this does not say

- It does not say the fragmentation in Q2 is imaginary. Five stores is a real
  observation from a real 300-person organization.
- It does not say nobody would buy this. It says the *reason* D-004 gives for
  why they would buy it is weaker than recorded.
- It does not close Q3, Q5, Q6, Q7, or Q11–Q13, which remain unanswered.

## What to do differently in interviews 002+

1. **Drop the incident-hunting frame.** It presumed a failure mode. Ask what
   the last hour of wasted work looked like, and let the shape emerge.
2. **Ask about the human-in-the-loop cost directly.** If agents assist rather
   than replace, the question is what review costs, not what breaches escape.
3. **Ask Q6 again and mean it** — whether anyone built anything to fix this,
   and whether it still runs. It is the question least vulnerable to
   founder bias and it was not answered.
4. **Test the fragmentation observation on someone else.** Five knowledge
   stores is the strongest signal here and it needs a non-founder source.
