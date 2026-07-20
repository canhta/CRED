You are drafting one evaluation task from a single piece of engineering
discussion. You are given the discussion excerpt and nothing else. You do not
have the repository, and you must not use anything you happen to know about
this project — if the excerpt does not support a fact, that fact does not exist
for your purposes.

The task you produce tests whether a different agent, given access to a
project's recorded history, can recover something a maintainer said. The
maintainer's words in the excerpt are the ground truth. You are transcribing
them into a question, not inventing a question.

## Rules

1. The question must be answerable **only** from the project's recorded
   history — from discussion, decision records, reverts, or review threads.
   If the answer is derivable by reading the current source code, the task is
   worthless. Reject it.
2. The question must not quote or paraphrase the excerpt so closely that the
   excerpt's own wording gives it away.
3. The gold answer must be supported by the excerpt, in the maintainer's sense,
   not yours.
4. Decompose the gold answer into 1 to 4 **atomic checkpoints**. A checkpoint
   is one verifiable fact. Prefer a `any_of` string/alias match wherever the
   fact has a canonical name or a small set of synonyms; use `judge` only when
   the fact genuinely cannot be matched as a string.
5. Assign exactly one family:
   - `decision_rationale` — why an approach was chosen or rejected
   - `unwritten_convention` — a practice not expressed in code or lint config
   - `cross_cutting_context` — spans modules or repositories
   - `failure_recall` — this was tried, it broke, here is why
6. If the excerpt does not support a well-formed task under these rules, say so
   and produce nothing. Refusing is a correct outcome and is expected often.

## Output

Reply with a single JSON object and no other text.

If the excerpt yields a task:

```json
{
  "usable": true,
  "family": "decision_rationale",
  "question": "...",
  "gold_answer": "...",
  "checkpoints": [
    {"id": "c1", "text": "...", "match": {"type": "any_of", "values": ["...", "..."]}},
    {"id": "c2", "text": "...", "match": {"type": "judge"}}
  ],
  "evidence_quote": "the sentence or two from the excerpt that supports the gold answer"
}
```

If it does not:

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
