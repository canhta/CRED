You are drafting one **abstention decoy** from a single piece of engineering
discussion. You are given the discussion excerpt and nothing else.

A decoy is a question that sounds exactly like the other questions in the
evaluation set — same repository, same register, same kind of subject — but
whose answer is **not recorded anywhere**. The correct behaviour for the agent
under test is to say so. Decoys exist because a memory system can otherwise win
by confidently confabulating, and a judge will reward it.

## Rules

1. The question must be about a *plausible neighbour* of the excerpt's subject:
   a detail one would reasonably expect to have been discussed, that this
   excerpt shows was not. Good decoys: the rejected alternative nobody named,
   the benchmark nobody ran, the person who was not consulted, the date nobody
   recorded.
2. It must **not** be answerable from the excerpt. If the excerpt answers it,
   it is not a decoy — it is a task, and you have the wrong prompt.
3. It must not be unanswerable for a boring reason. "What will they do in
   2030?" is not a decoy, it is a nonsense question. The decoy must be a
   question a maintainer could have answered but no maintainer wrote down.
4. Give the distinctive terms a checker should search the record for. If any of
   them turn out to appear in the record, this decoy will be discarded — so
   list the terms that would actually indicate the answer is present.

## Output

Reply with a single JSON object and no other text.

```json
{
  "usable": true,
  "question": "...",
  "why_unrecorded": "...",
  "search_terms": ["...", "...", "..."]
}
```

Or:

```json
{"usable": false, "reason": "..."}
```

## Excerpt

Repository: {{REPO}}
Written on: {{ANCHOR_DATE}} by @{{ANCHOR_AUTHOR}}
Context: pull request #{{PR_NUMBER}} — {{PR_TITLE}}

<excerpt>
{{ANCHOR_TEXT}}
</excerpt>
