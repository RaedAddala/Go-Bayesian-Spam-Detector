# Text Preprocessing for More Efficient Naive Bayes Results

Spam filters face a crucial challenge when dealing with textual data: semantically identical words can be written in different ways. This occurs in several forms:

* Conjugation
* Slang
* The use of abbreviations
* Spelling mistakes
* Punctuation and case differences
* The use of different symbols to represent the same thing, for example, using a specific character instead of an existing combination, or omitting symbols when writing (e.g., `c` vs `ç`). Some malicious users may also insert non-printable characters to mislead filters.

From this list, the last two issues are relatively easy to handle, as they are straightforward and largely language-independent especially when users write in multiple languages or use slang.

## Text Preprocessing

Text preprocessing is one of the most important steps in a Naive Bayes spam filter because *the model relies **entirely** on token frequencies*. Small textual variations can artificially increase the vocabulary size and fragment statistically identical words into separate entries. For example:

| Variant | Semantic Meaning |
|---|---|
| `FREE` | same word |
| `free` | same word |
| `FrEe` | same word |
| `frëe` | same word |
| `f.r.e.e` | same word |

Without normalization, the classifier would treat all of these as unrelated tokens, weakening probability estimation and reducing classification accuracy.

The goal of preprocessing is therefore to reduce semantically equivalent text into a consistent canonical representation before tokenization.

## Unicode-Aware Text Cleaning

The preprocessing pipeline starts with a Unicode-aware regular expression:

```golang
var nonWordRegex = regexp.MustCompile(`[^\p{L}\p{N}\s]+`)
```

This expression removes every character that is **not**:

* `\p{L}` : any Unicode letter
* `\p{N}` : any Unicode digit
* `\s` : whitespace

This approach works correctly with multilingual text and preserves characters from many writing systems. The text is also converted to lowercase before tokenization.

## Unicode Normalization

Unicode introduces an important problem in NLP: visually identical text may have multiple binary representations.

For example, the character `ç` can be represented in two different ways:

| Representation | Description |
|---|---|
| `U+00E7` (`ç`) | precomposed character |
| `U+0063 + U+0327` | `c` + combining cedilla |

Although these strings look identical, they are different byte sequences and would produce different tokens without normalization. Unicode normalization solves this problem by transforming equivalent sequences into a canonical form. One case use Go's Standard Unicode normalization package:

```golang
golang.org/x/text/unicode/norm
```

The preprocessing pipeline applies:

* decomposition
* canonicalization
* recomposition

to ensure consistent token generation across different Unicode encodings. This prevents duplicate entries in the vocabulary caused by Unicode inconsistencies. It is especially important in spam filtering because malicious users frequently exploit Unicode ambiguities to bypass detection systems.
