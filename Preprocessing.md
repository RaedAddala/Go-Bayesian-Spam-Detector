# Text Preprocessing for More Efficient Naive Bayes Results

Spam filters face a crucial challenge when dealing with textual data: semantically identical words can be written in different ways. This occurs in several forms:

- Conjugation
- Slang
- The use of abbreviations
- Spelling mistakes
- Punctuation and case differences
- The use of different symbols to represent the same thing, for example, using a specific character instead of an existing combination, or omitting symbols when writing (e.g., `c` vs `ç`). Some malicious users may also insert non-printable characters to mislead filters.

From this list, the last two issues are relatively easy to handle, as they are straightforward and largely language-independent, especially when users write in multiple languages or use slang.

## Text Preprocessing

Text preprocessing is one of the most important steps in a Naive Bayes spam filter because *the model relies **entirely** on token frequencies*. Small textual variations can artificially increase the vocabulary size and fragment statistically identical words into separate entries. For example:

| Variant | Semantic Meaning |
| --- | --- |
| `FREE` | same word |
| `free` | same word |
| `FrEe` | same word |
| `frëe` | same word |
| `f.r.e.e` | same word |

Without normalization, the classifier would treat all of these as unrelated tokens, weakening probability estimation and reducing classification accuracy.

The goal of preprocessing is therefore to reduce semantically equivalent text into a consistent canonical representation before tokenization.

### What the current pipeline actually handles

| Variant | Result of `cleanup()` | Handled? |
| --- | --- | --- |
| `FREE`, `free`, `FrEe` | `free` | Yes (lowercasing) |
| `ｆｒｅｅ` (full-width) | `free` | Yes (NFKC) |
| `frëe` | `frëe` | **No**: NFKC keeps diacritics; stripping them needs NFD + removing combining marks |
| `f.r.e.e` | *(nothing)* | **No**: the dots split it into 1-letter pieces, which are then dropped as too short |

The two unhandled cases are tracked in [`TODO.md`](TODO.md).

## Unicode-Aware Text Cleaning

Rather than using expensive regular expressions, we iterate through the byte stream and evaluate each character (rune) directly:

```go
for i := 0; i < len(normalized); {
    r, size := utf8.DecodeRune(normalized[i:])
    i += size
    if unicode.IsLetter(r) || unicode.IsDigit(r) {
        word.WriteRune(unicode.ToLower(r))
    } else {
        // Token boundary logic...
    }
}
```

This approach avoids the overhead of a regex engine and works correctly with multilingual text by leveraging Go's `unicode` package.

Tokens shorter than 3 or longer than 30 characters are discarded. These lengths are counted in **runes**, not bytes, so that a 2-letter word such as `ça` (3 bytes in UTF-8) is treated the same as `of`.

## Memory Efficiency

Converting the initial raw file data (`[]byte`) to a `string` creates an unnecessary copy of the entire file in memory. Processing the byte slice directly reduces the load on the Garbage Collector and improves performance during high-concurrency tasks.

## Unicode Normalization

Unicode introduces an important problem in NLP: visually identical text may have multiple binary representations.

For example, the character `ç` can be represented in two different ways:

| Representation | Description |
| --- | --- |
| `U+00E7` (`ç`) | precomposed character |
| `U+0063 + U+0327` | `c` + combining cedilla |

Although these strings look identical, they are different byte sequences and would produce different tokens without normalization. Unicode normalization solves this problem by transforming equivalent sequences into a canonical form. One can use Go's standard Unicode normalization package:

```go
import "golang.org/x/text/unicode/norm"
```

The preprocessing pipeline applies **NFKC** normalization directly to the raw bytes using `norm.NFKC.Bytes(content)`. This ensures consistent token generation across different Unicode encodings while avoiding unnecessary string allocations before the cleaning phase.

## About Concurrency Handling

To avoid lock contention, each ingestion worker counts tokens into its own map for every file it processes. The worker maps are merged once, after all files in the directory are done, so no mutex is taken while tokenizing.

Each Enron folder is tokenized only once. Cross-validation folds reuse the cached per-folder bags and sum them, instead of re-tokenizing the training emails for every fold.
