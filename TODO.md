# TODO

Planned experiments and improvements. Rule for every experiment: compare against the current baseline in [`RESULTS.md`](RESULTS.md) with the same 6-fold protocol, and never tune anything on the test fold (see Hyperparameter tuning).

## 1. Evaluation: better metrics

- [ ] Report the **ham misclassification rate** FP / (FP + TN) and the **spam misclassification rate** FN / (FN + TP) per fold and in aggregate. For a spam filter, losing legitimate mail matters more than letting some spam through.
- [ ] Add a **decision threshold** on `LogPostSpam - LogPostHam` instead of the fixed `logSpam > logHam` rule (the log-posteriors are already in `ClassifyResult`).
- [ ] Compute **ROC curves and AUC** by sweeping that threshold, per fold and pooled.
- [ ] Report **spam recall at a fixed low ham-error rate** (e.g. 0.1%, 0.5%, 1%), as in Metsis et al., so results can be compared with the paper.
- [ ] Report the **spread across folds** (std or min/max), not only micro/macro means.
- [ ] A cost-weighted metric (e.g. a false positive costs λ = 9 or 99 false negatives) for picking operating points.
- [ ] Export per-email scores (CSV) so plots can be made outside Go.

## 2. Training priors

The folds have very different class balances (enron1–3 are about 71–75% ham, enron4–6 about 25–29% ham), so the prior learned from 5 training folds rarely matches the test fold.

- [ ] **Uniform priors**: log P(ham) = log P(spam) = log 0.5.
- [ ] **Empirical priors** (current behavior) as the baseline.
- [ ] **Fixed priors** swept over a range, e.g. P(spam) ∈ {0.1, 0.25, 0.5, 0.75, 0.9}.
- [ ] Check how much of the per-fold precision gap (enron1/2 vs enron5/6) is explained by the prior, compared with the vocabulary differences between users.
- [ ] Note: changing the prior only shifts the decision threshold by a constant, so compare priors using threshold-free metrics (AUC) as well as at the default threshold.

## 3. Hyperparameter tuning

- [ ] Make hyperparameters command-line flags instead of constants in `main.go` (`-mincount`, `-alpha`, `-workers`, `-data`, `-variant`, `-prior`, `-threshold`).
- [ ] Use **nested cross-validation**: for each outer test fold, choose hyperparameters by an inner CV over the 5 training folds only, then evaluate once on the test fold. Picking settings by looking at the outer test scores inflates the results.
- [ ] Grid: `minCount` ∈ {1, 2, 5, 10, 20, 50}, `alpha` (see section 6), the NB variant (section 5), the prior (section 2), and the vocabulary size (section 7).
- [ ] Record the selected values per outer fold to see whether they are stable.

## 4. Boolean multinomial Naive Bayes

- [ ] Count each token **at most once per email** when building the bags (deduplicate the tokens of a document in ingestion) and when classifying (sum each distinct token once in `Classify`).
- [ ] Compare with the current term-frequency multinomial NB on the same folds. The paper reports that this variant is often the best or close to it.

## 5. Benchmark multiple NB variants × hyperparameters

- [ ] Put the variants behind one interface so they share ingestion and evaluation:
  - Multinomial NB, term frequency (current)
  - Multinomial NB, Boolean attributes (section 4)
  - Multivariate Bernoulli NB (models the absence of tokens too; needs per-token document frequencies)
  - Optional: Multivariate Gauss NB and Flexible Bayes, which the paper also compares, and Complement NB
- [ ] Ingestion must then record **document frequencies** (number of emails containing each token) as well as term counts. Collect both in one pass so each folder is still tokenized once.
- [ ] Run the full grid (variant × alpha × minCount × prior × vocabulary size) and output one table, plus a CSV for plotting.
- [ ] Track runtime and memory per configuration as well as quality.
- [ ] Add Go benchmarks (`testing.B`) for `cleanup`, `NewModel` and `Classify` to catch performance regressions.

## 6. Smoothing: the alpha hyperparameter

- [ ] Replace the hard-coded Laplace `+1` with Lidstone smoothing: P(w|c) = (count(w,c) + α) / (totalTokens(c) + α|V|).
- [ ] Sweep α ∈ {0.01, 0.05, 0.1, 0.25, 0.5, 1, 2}.
- [ ] Note: the current results already show how sensitive the trade-off is to the smoothing denominator (moving |V| to the kept vocabulary shifted FP 509 → 759 and FN 566 → 317 at the same accuracy), so α is likely to matter.

## 7. Feature selection

Currently the only selection is the `minCount` frequency cutoff, which keeps frequent but uninformative tokens and drops rare but strongly discriminative ones.

- [ ] Rank tokens by **information gain** (mutual information with the class) computed on the training folds only, and keep the top N. Sweep N ∈ {500, 1000, 3000, 10000, all}; the paper uses IG with 500, 1000 and 3000 attributes.
- [ ] Compare with chi-squared and with the current `minCount` cutoff.
- [ ] Compute the ranking inside each training fold (never using the test fold), as part of nested CV.

## 8. Preprocessing

Issues found in the current `cleanup()` (checked against the code and the dataset):

- [ ] **Invisible characters split words.** Zero-width space (U+200B), soft hyphen (U+00AD) and other format characters (Unicode category `Cf`) are treated as delimiters, so `vi​agra` becomes `agra` and `fr­ee` disappears. These are classic obfuscation tricks; they should be removed, not treated as word boundaries.
- [ ] **Invalid UTF-8.** 1,078 of the 33,716 emails (3.2%) are not valid UTF-8 (probably Latin-1 / Windows-1252). Each invalid byte decodes to `RuneError` and acts as a delimiter, which splits accented words. Detect the encoding and decode, or fall back to Windows-1252 when `utf8.Valid` fails.
- [ ] **Diacritics are not folded.** NFKC keeps `ë`, so `frëe` ≠ `free`. Try NFD → drop combining marks (`unicode.Mn`) → NFC, and measure whether it helps (it can also merge distinct words in some languages).
- [ ] **Dotted/spaced obfuscation.** `f.r.e.e` and `f r e e` produce only 1-letter pieces, which are dropped. Collapse runs of single letters separated by `.`, `-`, `_` or spaces into one token.
- [ ] **Leetspeak and repeated letters.** `v1agra`, `fr33`, `freeeee`: try mapping common digit/letter substitutions inside tokens that mix letters and digits, and capping runs of repeated letters (e.g. to 2).
- [ ] **Subject line.** Every email starts with `Subject:`, so the token `subject` appears in all emails and carries no signal, and subject words are mixed with body words. Try prefixing subject tokens (`subj:free`) so they are separate features.
- [ ] **Discarded punctuation.** `$`, `!` and `%` are strong spam signals but are thrown away. Try emitting them (or runs like `!!!`) as tokens.
- [ ] **Numbers, URLs, e-mail addresses.** Numbers fragment the vocabulary (each price or phone number is its own token) and URLs are split into pieces. Try placeholder tokens (`<num>`, `<url>`, `<email>`) and/or keeping the URL domain as one token.
- [ ] Optional: stop-word removal and stemming. Both are often neutral or harmful for NB spam filtering, so treat them as experiments, not defaults.
- [ ] HTML: this preprocessed Enron corpus contains no HTML, so tag and entity stripping only matters when moving to raw email.
- [ ] Measure each preprocessing change on its own with the CV harness, and add a `cleanup` test case for each.

## 9. Engineering (follow-ups)

- [ ] Add a CI workflow (`go vet`, `go test -race`).
- [ ] Add a `classify <file>` mode and save/load the trained model (gob or JSON) so the filter can be used outside the CV harness.
- [ ] Rename the module (`naive-bayes`) to match the repository (`spam_filter`).
