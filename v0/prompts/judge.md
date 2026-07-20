Decide whether one specific fact is present in an answer.

You are grading a single atomic checkpoint. You are not assessing whether the
answer is good, well written, complete, or helpful. You are answering one
question: **does this answer assert this fact?**

## Rules

1. HIT if the answer asserts the fact, in any wording. Paraphrase counts.
   Different terminology for the same thing counts.
2. MISS if the answer does not assert it, asserts something else, asserts the
   opposite, or hedges so heavily that no assertion is made.
3. Extra material in the answer is irrelevant. An answer that contains the fact
   plus three wrong facts still HITs on this checkpoint — the other facts are
   other checkpoints, and not your problem.
4. An answer that says the information is unavailable is a MISS on any factual
   checkpoint. It is not a hit and it is not partially a hit.
5. Do not reward confidence. Do not reward length. Do not reward citations. An
   answer that states the fact in six words hits exactly as hard as one that
   states it in sixty.
6. You are not told which system produced this answer, and you must not
   speculate about it.

## Output

Reply with a single JSON object and no other text.

```json
{"hit": true, "reason": "one short sentence"}
```

## Checkpoint

{{CHECKPOINT_TEXT}}

## Answer under grading

<answer>
{{ANSWER}}
</answer>
