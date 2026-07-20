Extract the durable claims from one excerpt of a software project's recorded
history.

A **claim** is a statement about the project that would still be worth knowing
six months from now: a decision and its reason, a convention, an approach that
was rejected, a thing that broke and why. It is not a summary of the excerpt
and it is not a description of what a patch does.

You are given the excerpt and nothing else. You do not know what anyone will
ask about this project, and you must not try to guess. Extract what the excerpt
records; do not extract what you think might be useful.

## Rules

1. Every claim must be supported by the excerpt. If you cannot point at the
   sentence that supports it, do not write it.
2. One fact per claim. "We chose X because Y, and Z was rejected" is two claims.
3. Write claims so they stand alone. A reader who has not seen the excerpt must
   be able to understand the claim without it.
4. Quote the supporting span verbatim in `evidence_quote`. Keep it short — one
   or two sentences.
5. Skip mechanical content: changelog lines, version bumps, formatting notes,
   "LGTM", CI output.
6. Extract at most 8 claims. Most excerpts yield fewer. Zero is a normal and
   correct answer for an excerpt that records nothing durable.

## Output

Reply with a single JSON object and no other text.

```json
{
  "claims": [
    {
      "text": "...",
      "evidence_quote": "...",
      "kind": "decision" | "convention" | "rejected_approach" | "failure" | "constraint"
    }
  ]
}
```

## Excerpt

Repository: {{REPO}}
Source: {{SOURCE_TYPE}} — {{PATH}}
Recorded: {{DATE}}

<excerpt>
{{CHUNK_TEXT}}
</excerpt>
