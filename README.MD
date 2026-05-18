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

- `main.go`: wires everything together (train on `enron1..enron5`, test on `enron6`)
- `ingest.go`: concurrent ingestion and vocabulary (bag-of-words) construction
- `preprocessing.go`: Unicode normalization, cleaning, and tokenization
- `model.go`: Naive Bayes model (log-space) and Laplace smoothing
- `evaluate.go`: evaluation loop over `ham/` and `spam/` folders
- `metrics.go`: accuracy, precision, recall, F1, and confusion matrix counters

Dataset is expected under `./data/` with folders like `enron1/ham`, `enron1/spam`, ...

## How it works

### Ingestion (training bags)

Training data is read from multiple Enron folders (configured in `main.go`).

`ingestFolders()` builds two global maps:

- `hamBag: map[token]count`
- `spamBag: map[token]count`

It also counts the number of training documents (emails) per class.

To improve throughput, ingestion walks the directory and processes files using a worker semaphore:

- `maxWorkers` controls the maximum number of concurrent file-processing goroutines
- each file is tokenized into a **local** bag first, then merged into the global map under a mutex

This keeps shared-map contention low and makes ingestion scale better on multi-core machines.

### Preprocessing

All tokenization goes through `cleanup(content []byte) []string` in `preprocessing.go`.

Key properties:

- **Unicode normalization:** NFKC (`norm.NFKC.Bytes`) to canonicalize visually equivalent text
- **Unicode-aware scanning:** iterates runes (`utf8.DecodeRune`) instead of regex
- **Token rules:**
  - keep letters and digits; lowercase everything
  - treat everything else as a delimiter
  - discard very short tokens (`<= 2`) and very long tokens (`> 30`)

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

To reduce noise and vocabulary fragmentation, `logWordProb()` ignores tokens whose combined frequency $(hamCount + spamCount) < minCount$ (`minCount` is configured in `main.go`).

### Evaluation

`evaluateModel()` walks `testDir/ham` and `testDir/spam`, classifies each email, and updates:

- totals (`Total`, `Correct`)
- confusion matrix counters (`TP`, `FP`, `TN`, `FN`)

`metrics.go` derives:

- Accuracy
- Precision
- Recall
- F1

## Running

From the repository root:

- Train on `enron1..enron5`
- Evaluate on `enron6`

Use:

```bash
go run .
```

See [`RESULTS.md`](RESULTS.md) for the latest recorded metrics and confusion matrix.

## Configuration knobs

Edit constants in `main.go`:

- `maxWorkers`: concurrency for ingestion
- `minCount`: ignore tokens rarer than this across both corpora
- `trainFolders` / `testFolder`: dataset split
