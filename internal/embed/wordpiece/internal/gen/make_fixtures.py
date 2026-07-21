#!/usr/bin/env python3
"""Emit tokenizer conformance fixtures from the pinned reference tokenizer.

Three suites, reused from go-embeddings-tokenizer.md rather than re-derived:

  curated.jsonl.gz    hand-picked edge cases
  fuzz.jsonl.gz       generated strings across Unicode blocks, code, and docs
  codepoints.jsonl.gz "ab" + chr(cp) + "cd" for every probed codepoint

Each line is {"t": <input>, "i": [<token ids>]}.

Usage:
    pip install 'tokenizers==0.22.2'
    python3 make_fixtures.py --tokenizer /path/to/tokenizer.json \
        --repo /path/to/cred --out ../../testdata
"""

import argparse
import glob
import gzip
import json
import os
import random
import sys

import tokenizers
from tokenizers import Tokenizer

PINNED_TOKENIZERS = "0.22.2"
LO, HI = 0x00, 0x2FFFF
SURROGATES = range(0xD800, 0xE000)

CURATED = [
    "",
    " ",
    "\t\n\r ",
    "The quick brown fox jumps over the lazy dog.",
    "Hybrid retrieval fuses BM25 and dense vectors with reciprocal rank fusion.",
    "func (s *Server) Recall(ctx context.Context, q *Query) (*Result, error)",
    "SELECT * FROM claims WHERE embedding <=> $1 LIMIT 10;",
    "internal/store/pg/claims.go:120",
    "CGO_ENABLED=0 go build ./...",
    "https://huggingface.co/BAAI/bge-small-en-v1.5/resolve/main/vocab.txt",
    "user@example.com",
    "a_b-c.d/e\\f|g",
    "{{if .CgoFiles}}{{.ImportPath}}{{end}}",
    "claim.acl ⊆ ⋂(evidence_i.acl)",
    "café naïve résumé",
    "İstanbul",           # Turkish dotted I
    "straße",             # German eszett
    "ﬁne ﬂour",      # ligatures
    "Αθήνα",   # Greek
    "Москва",  # Cyrillic
    "東京都",      # CJK
    "한국어",      # Hangul
    "\U0001f600\U0001f680",    # emoji
    "\U0001f468‍\U0001f469‍\U0001f467",  # ZWJ family
    "\x00\x07 control",
    "� replacement",
    " nbsp ",
    "z" * 120,                 # over max_input_chars_per_word
    "[CLS] literal special token in text [SEP]",
    "[MASK] and [PAD] and [UNK]",
    "prefix[MASK]suffix",
    "[[CLS]]",
    "⮂0 boundary \U0002b820\U0002b81f",  # the CJK-range hole
    "؝ arabic end of text mark",
    "ᪿ combining latin small letter w below",
    "Mixed 中文 and English слово together",
    "snake_case camelCase PascalCase SCREAMING_SNAKE",
    "-- comment\n// comment\n# comment\n/* comment */",
    "1234567890 3.14159 0xdeadbeef 1e-9",
    "  leading and trailing   ",
    "tab\tseparated\tvalues",
    "line\nbreak\r\nwindows",
    " ".join(["word"] * 1900),  # forces 512-token truncation
]

BLOCKS = [
    (0x0020, 0x007E), (0x00A0, 0x00FF), (0x0100, 0x017F), (0x0370, 0x03FF),
    (0x0400, 0x04FF), (0x0530, 0x058F), (0x0590, 0x05FF), (0x0600, 0x06FF),
    (0x0900, 0x097F), (0x0E00, 0x0E7F), (0x1E00, 0x1EFF), (0x2000, 0x206F),
    (0x3040, 0x30FF), (0x4E00, 0x4FFF), (0xAC00, 0xACFF), (0x1F300, 0x1F5FF),
]

IDENT_PARTS = ["get", "set", "new", "Claim", "Evidence", "ACL", "recall",
               "seed", "pg", "ctx", "err", "http", "v2", "ID", "_", "__"]


def gen_fuzz(rng, repo, n):
    out = []
    for _ in range(n // 3):
        lo, hi = rng.choice(BLOCKS)
        k = rng.randint(1, 24)
        out.append("".join(chr(rng.randint(lo, hi)) for _ in range(k)))
    for _ in range(n // 3):
        parts = [rng.choice(IDENT_PARTS) for _ in range(rng.randint(1, 5))]
        sep = rng.choice(["", "_", ".", "/", "-", " "])
        s = sep.join(parts)
        if rng.random() < 0.15:
            s = s + rng.choice(["[MASK]", "[CLS]", "[SEP]", "[UNK]", "[PAD]"])
        out.append(s)
    docs = []
    for path in sorted(glob.glob(os.path.join(repo, "**", "*.md"),
                                 recursive=True)):
        with open(path, encoding="utf-8", errors="replace") as f:
            body = f.read()
        docs.extend(body[i:i + 300] for i in range(0, len(body), 300))
    rng.shuffle(docs)
    out.extend(docs[:n - len(out)])
    return out


def gen_codepoints():
    for cp in range(LO, HI + 1):
        if cp in SURROGATES:
            continue
        yield "ab" + chr(cp) + "cd"


def write(path, tok, texts):
    n = 0
    with gzip.open(path, "wt", encoding="utf-8") as w:
        for t in texts:
            ids = tok.encode(t).ids
            w.write(json.dumps({"t": t, "i": ids}, ensure_ascii=False) + "\n")
            n += 1
    print("%-24s %7d cases  %8d bytes" % (
        os.path.basename(path), n, os.path.getsize(path)), file=sys.stderr)


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--tokenizer", required=True)
    ap.add_argument("--repo", required=True)
    ap.add_argument("--out", required=True)
    ap.add_argument("--fuzz", type=int, default=20000)
    args = ap.parse_args()

    if tokenizers.__version__ != PINNED_TOKENIZERS:
        sys.exit("tokenizers %s installed, %s pinned." % (
            tokenizers.__version__, PINNED_TOKENIZERS))

    tok = Tokenizer.from_file(args.tokenizer)
    tok.enable_truncation(max_length=512)

    os.makedirs(args.out, exist_ok=True)
    write(os.path.join(args.out, "curated.jsonl.gz"), tok, CURATED)
    write(os.path.join(args.out, "fuzz.jsonl.gz"), tok,
          gen_fuzz(random.Random(20260720), args.repo, args.fuzz))
    write(os.path.join(args.out, "codepoints.jsonl.gz"), tok, gen_codepoints())


if __name__ == "__main__":
    main()
