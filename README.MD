# Spam Filter Using Naive Bayes

A simple **multinomial Naive Bayes** spam detector written in Go, trained and evaluated on the **Enron spam/ham dataset**.

This repository is inspired by the paper **“Spam Filtering with Naive Bayes - Which Naive Bayes?”** (open access link below), but the implementation here is a compact, practical baseline focused on:

- A clean **bag-of-words** multinomial NB classifier
- **Unicode-aware** tokenization and normalization
- **Concurrent** corpus ingestion to speed up training
- A small evaluation harness that reports classic classification metrics

**Note (paper alignment):** This code implements **Multinomial Naive Bayes** with **Laplace smoothing** and a **term-frequency (tf)** token model, plus a `minCount` threshold.

In the paper, the authors compare multiple NB variants on the same Enron datasets (e.g., **Multivariate Bernoulli**, **Multinomial (tf)**, and **Multinomial with Boolean attributes**). This project uses standard Multinomial (tf); the paper reports that **Multinomial with Boolean attributes** is often best or very competitive.

References:

- Paper: [Spam Filtering with Naive Bayes - Which Naive Bayes?](https://www.researchgate.net/publication/221650814_Spam_Filtering_with_Naive_Bayes_-_Which_Naive_Bayes)
- Dataset: [Enron spam dataset](https://www2.aueb.gr/users/ion/data/enron-spam/)

## Project layout

- `main.go`: wires everything together (**6-fold cross-validation** over `enron1..enron6`)
- `ingest.go`: concurrent per-folder ingestion (bag-of-words) and corpus merging
- `preprocessing.go`: Unicode normalization, cleaning, and tokenization
- `model.go`: Naive Bayes model (log-space) and Laplace smoothing
- `evaluate.go`: concurrent evaluation over `ham/` and `spam/` folders
- `metrics.go`: accuracy, precision, recall, F1, and confusion matrix counters
- `*_test.go`: unit tests (`go test ./...`)
- [`TODO.md`](TODO.md): planned experiments and improvements

Dataset is expected under `./data/` with folders like `enron1/ham`, `enron1/spam`, ...

## How it works

### Ingestion (training bags)

Every Enron folder is tokenized **exactly once**, at startup. `ingestFolder()` returns a `Corpus` per folder:

- `HamBag`, `SpamBag`: `map[token]count`
- `NumHam`, `NumSpam`: number of documents (emails) read per class

For each fold, `mergeCorpora()` sums the corpora of the 5 training folders into a fresh `Corpus` (the cached per-folder corpora are not modified). This avoids re-reading and re-tokenizing each email 5 times across the 6 folds.

Within a directory, files are tokenized by a pool of `maxWorkers` goroutines. Each worker counts into its own bag, and the worker bags are merged once at the end, so there is no shared lock while tokenizing.

Errors are not swallowed: a missing directory or an unreadable file aborts the run with an error, and a class with zero training documents is rejected by `NewModel()`.

### Preprocessing

All tokenization goes through `cleanup(content []byte) []string` in `preprocessing.go`.

Key properties:

- **Unicode normalization:** NFKC (`norm.NFKC.Bytes`) to canonicalize visually equivalent text
- **Unicode-aware scanning:** iterates runes (`utf8.DecodeRune`) instead of regex
- **Token rules:**
  - keep letters and digits; lowercase everything
  - treat everything else as a delimiter
  - discard very short tokens (`<= 2` characters) and very long tokens (`> 30` characters); lengths are counted in runes, not bytes

More details are documented in [`Preprocessing.md`](Preprocessing.md).

### Model (Multinomial Naive Bayes, log-space)

The model in `model.go` is a standard multinomial NB classifier:

- Class priors:
  - `logPriorHam = log(numHamDocs / totalDocs)`
  - `logPriorSpam = log(numSpamDocs / totalDocs)`

- Conditional probabilities with **Laplace smoothing**:

  $$P(w|c) = (count(w,c) + 1) / (totalTokens(c) + |V|)$$

To avoid underflow, classification is performed in **log-space**:

$$\log P(c|doc) = \log P(c) + \sum_{tok \in doc} \log P(tok|c)$$

#### Rare-token filtering (`minCount`)

To reduce noise and vocabulary fragmentation, `NewModel()` leaves out of the vocabulary every token whose combined frequency $(hamCount + spamCount) < minCount$ (`minCount` is configured in `main.go`). Those tokens are ignored at classification time, just like unseen tokens.

The smoothing terms are computed over the kept vocabulary only: $|V|$ is the number of kept tokens and $totalTokens(c)$ is the sum of their counts in class $c$. The per-token log-probabilities are precomputed when the model is built.

### Evaluation

`evaluateModel()` lists `testDir/ham` and `testDir/spam` and classifies the emails with a pool of `maxWorkers` goroutines (the model is read-only, so it is shared safely). Each worker keeps its own counters, which are summed at the end:

- totals (`Total`, `Correct`)
- confusion matrix counters (`TP`, `FP`, `TN`, `FN`)

`metrics.go` derives:

- Accuracy
- Precision
- Recall
- F1

## Running

From the repository root:

- Run **6-fold cross-validation** (each `enron*` collection is used once as the test fold)

Use:

```bash
go run .
```

Output includes the one-time **preprocess+ingest** time, per-fold metrics with timing for **merge**, **train**, and **classify+eval**, and aggregate (micro/macro) metrics.

Run the unit tests with:

```bash
go test ./...
```

See [`RESULTS.md`](RESULTS.md) for the latest recorded metrics and confusion matrix.

## Configuration knobs

Edit constants in `main.go`:

- `maxWorkers`: concurrency for ingestion and evaluation
- `minCount`: ignore tokens rarer than this across both corpora
- `folds`: which Enron collections to include in the cross-validation
