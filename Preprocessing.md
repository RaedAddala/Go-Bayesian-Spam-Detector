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

Rather than using expensive regular expressions, we iterate through the byte stream and evaluate each character (rune) directly:

```golang
for i := 0; i < len(normalized); {
   r, size := utf8.DecodeRune(normalized[i:])
   i += size
   if unicode.IsLetter(r) || unicode.IsDigit(r) {
       word.WriteRune(unicode.ToLower(r))
   } else {
       // Token boundary logic...
   }
```

This approach avoids the overhead of a regex engine and works correctly with multilingual text by leveraging Go's `unicode` package.

This approach works correctly with multilingual text and preserves characters from many writing systems.

## Memory Efficiency

Converting the initial raw file data (`[]byte`) to a `string` creates an unnecessary copy of the entire file in memory. Processing the byte slice directly reduces the load on the Garbage Collector and improves performance during high-concurrency tasks.

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

The preprocessing pipeline applies **NFKC** normalization directly to the raw bytes using `norm.NFKC.Bytes(content)`. This ensures consistent token generation across different Unicode encodings while avoiding unnecessary string allocations before the cleaning phase.

## About Concurrency Handling

To prevent "Lock Contention" tokens should be aggregated into a local map for each file. The global "Bag of Words" should only be locked once per file to merge these local results, ensuring that multiple CPU cores aren't stalled waiting for a single mutex.
