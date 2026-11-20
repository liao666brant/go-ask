"""Build the embedded English–Chinese word list from the pinned ECDICT CSV."""

import csv
import hashlib
from pathlib import Path
import re
import sys

SOURCE_HASH = "d1ecbdf9ee2b920afb6f430ca6b6bb9f559588cf0dff86e04bd255f9390843cc"
ENGLISH_WORD = re.compile(r"[A-Za-z]+(?:[\x27-][A-Za-z]+)*\Z")
WORD_CLASS = re.compile(r"^\s*(n|adj|a|adv|ad|num|pron|v|vi|vt|prep|conj|int|interj|det|art|aux|pl)\.(?:\s|$)", re.I)
EXTRA_WORDS = {"async", "await", "linux", "unix", "windows", "python", "sqlite", "json", "regex"}


def main(source):
    if hashlib.sha256(source.read_bytes()).hexdigest() != SOURCE_HASH:
        sys.exit("Please supply ecdict.csv from the revision in SOURCE.md")
    words = {}
    with source.open(encoding="utf-8", newline="") as stream:
        for row in csv.DictReader(stream):
            word = row["word"]
            key = word.lower()
            translation = row["translation"].strip()
            if not ENGLISH_WORD.fullmatch(word) or not translation:
                continue
            if not (row["tag"] or int(row["bnc"] or 0) or int(row["frq"] or 0)
                    or "[计]" in translation or key in EXTRA_WORDS):
                continue
            if key in words and word != key:
                continue
            tags = dict.fromkeys(match.group(1) + "." for line in translation.split(r"\n")
                                 if (match := WORD_CLASS.match(line)))
            words[key] = [key, word, row["phonetic"].strip(), " / ".join(tags), translation]
    target = Path(__file__).with_name("entries.tsv")
    with target.open("w", encoding="utf-8", newline="\n") as output:
        for _, fields in sorted(words.items()):
            output.write("\t".join(re.sub(r"[\r\n\t]+", " ", v) for v in fields) + "\n")
    print(f"{len(words)} entries")


if __name__ == "__main__":
    main(Path(sys.argv[1]))
