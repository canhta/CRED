#!/usr/bin/env python3
"""Generate the WordPiece character-class tables by probing the reference tokenizer.

The target is byte-identity with the tokenizer that trained the model, NOT
Unicode correctness. The two disagree on 824 codepoints, and every disagreement
is invisible on English text (D-008, go-embeddings-tokenizer.md).

So: never derive these tables from Go's `unicode` package. Probe the pinned
HuggingFace `tokenizers` release, one codepoint at a time, and emit sorted range
tables.

Usage:
    pip install 'tokenizers==0.22.2'
    python3 probe_tables.py --out ../../tables_gen.go

The `tokenizers` version is pinned in PINNED_TOKENIZERS below and asserted at
run time. Regenerate deliberately; never hand-edit the output.
"""

import argparse
import sys
import unicodedata

import tokenizers
from tokenizers import normalizers, pre_tokenizers

PINNED_TOKENIZERS = "0.22.2"

# U+0000..U+2FFFF, excluding surrogates.
#
# go-embeddings-tokenizer.md probes conformance from U+0020, but the tables
# themselves must start at U+0000: tab, newline and carriage return are
# whitespace to the reference tokenizer and live below U+0020. Probing from
# U+0020 yields 19 whitespace codepoints where the correct answer is 22, and
# the three that go missing are the three that occur in every source file.
LO, HI = 0x00, 0x2FFFF
SURROGATES = range(0xD800, 0xE000)


def codepoints():
    for cp in range(LO, HI + 1):
        if cp in SURROGATES:
            continue
        yield cp


def to_ranges(members):
    """Collapse a sorted iterable of ints into inclusive [lo, hi] ranges."""
    out = []
    for cp in members:
        if out and cp == out[-1][1] + 1:
            out[-1][1] = cp
        else:
            out.append([cp, cp])
    return out


def probe():
    # Four normalizers, each isolating one stage of BertNormalizer so the
    # stages cannot mask one another. `strip_accents=None` in the shipped
    # tokenizer.json means "inherit lowercase", i.e. true; it is set
    # explicitly here because the probe must not depend on that inheritance.
    clean = normalizers.BertNormalizer(
        clean_text=True, handle_chinese_chars=False,
        strip_accents=False, lowercase=False)
    accents = normalizers.BertNormalizer(
        clean_text=False, handle_chinese_chars=False,
        strip_accents=True, lowercase=False)
    chinese = normalizers.BertNormalizer(
        clean_text=False, handle_chinese_chars=True,
        strip_accents=False, lowercase=False)
    lower = normalizers.BertNormalizer(
        clean_text=False, handle_chinese_chars=False,
        strip_accents=False, lowercase=True)
    pretok = pre_tokenizers.BertPreTokenizer()

    control, whitespace, mn, punct, cjk = [], [], [], [], []
    lowermap = {}

    for cp in codepoints():
        ch = chr(cp)

        # clean_text: control characters are dropped, whitespace becomes ' '.
        got = clean.normalize_str("ab" + ch + "cd")
        if got == "abcd":
            control.append(cp)
        elif got == "ab cd":
            whitespace.append(cp)

        # strip_accents: NFD, then drop anything the Rust table calls Mn.
        # Only NFD-stable codepoints are probed, because Go applies NFD before
        # consulting this table, so a decomposing codepoint never reaches it.
        if unicodedata.normalize("NFD", ch) == ch:
            if accents.normalize_str("a" + ch) == "a":
                mn.append(cp)

        # handle_chinese_chars: CJK codepoints are space-padded.
        if chinese.normalize_str("ab" + ch + "cd") == "ab " + ch + " cd":
            cjk.append(cp)

        # BertPreTokenizer splits around punctuation. A whitespace character
        # splits too but is dropped rather than kept, so it yields
        # ["ab", "cd"] and never matches here — the classes stay disjoint
        # without a second exclusion pass.
        pieces = [p for p, _ in pretok.pre_tokenize_str("ab" + ch + "cd")]
        if pieces == ["ab", ch, "cd"]:
            punct.append(cp)

        # Rust's str::to_lowercase is a full mapping: one codepoint can yield
        # several (Turkish dotted I is the case that matters). Go's
        # strings.ToLower uses current Unicode tables, so it is not usable.
        low = lower.normalize_str(ch)
        if low != ch:
            lowermap[cp] = low

    return {
        "control": to_ranges(control),
        "whitespace": to_ranges(whitespace),
        "mn": to_ranges(mn),
        "punct": to_ranges(punct),
        "cjk": to_ranges(cjk),
    }, lowermap


def emit_ranges(w, name, ranges):
    w.write("// %s has %d ranges covering %d codepoints.\n" % (
        name, len(ranges), sum(hi - lo + 1 for lo, hi in ranges)))
    w.write("var %s = []rrange{\n" % name)
    for lo, hi in ranges:
        w.write("\t{0x%04X, 0x%04X},\n" % (lo, hi))
    w.write("}\n\n")


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--out", required=True)
    args = ap.parse_args()

    if tokenizers.__version__ != PINNED_TOKENIZERS:
        sys.exit("tokenizers %s installed, %s pinned. The tables are only "
                 "meaningful against the pinned release." % (
                     tokenizers.__version__, PINNED_TOKENIZERS))

    tables, lowermap = probe()

    with open(args.out, "w", encoding="utf-8") as w:
        w.write("// Code generated by internal/gen/probe_tables.py. DO NOT EDIT.\n")
        w.write("//\n")
        w.write("// Probed from HuggingFace tokenizers %s over U+%04X..U+%04X.\n" % (
            PINNED_TOKENIZERS, LO, HI))
        w.write("// Regenerate with `go generate ./internal/embed/wordpiece/...`.\n")
        w.write("\n")
        w.write("package wordpiece\n\n")
        for name, key in (("controlRanges", "control"),
                          ("whitespaceRanges", "whitespace"),
                          ("nonspacingMarkRanges", "mn"),
                          ("punctRanges", "punct"),
                          ("cjkRanges", "cjk")):
            emit_ranges(w, name, tables[key])

        w.write("// lowerMap holds every codepoint whose reference lowercase form\n")
        w.write("// differs from itself. A full mapping: one rune can yield several.\n")
        w.write("var lowerMap = map[rune]string{\n")
        for cp in sorted(lowermap):
            w.write("\t0x%04X: %s,\n" % (cp, go_quote(lowermap[cp])))
        w.write("}\n")

    for k, v in tables.items():
        print("%-12s %6d ranges %8d codepoints" % (
            k, len(v), sum(hi - lo + 1 for lo, hi in v)), file=sys.stderr)
    print("%-12s %6d entries" % ("lowercase", len(lowermap)), file=sys.stderr)


def go_quote(s):
    out = ['"']
    for c in s:
        if c == '"':
            out.append('\\"')
        elif c == "\\":
            out.append("\\\\")
        elif 0x20 <= ord(c) < 0x7F:
            out.append(c)
        else:
            out.append("\\u%04X" % ord(c) if ord(c) <= 0xFFFF
                       else "\\U%08X" % ord(c))
    out.append('"')
    return "".join(out)


if __name__ == "__main__":
    main()
