# Current Results

Command:

```bash
go run .
```

## Evaluation protocol

This project reports **6-fold cross-validation** over the Enron corpora (`enron1..enron6`): each `enron*` collection is used once as the test fold and the remaining 5 are used for training.

The program first tokenizes every folder once and prints the time this took. For each fold, it then prints:

- Train corpus summary (ham/spam doc counts, vocab size, `minCount`)
- Metrics: Accuracy, Precision, Recall, F1 + confusion-matrix counts
- Timing: merge (summing the cached per-folder bags), train, classify+eval

At the end, it prints aggregate metrics:

- **Micro**: summed confusion matrix across folds
- **Macro**: mean of per-fold metrics

## Aggregate results (6 folds, `minCount=20`)

| Scope | Total emails | Accuracy | Precision | Recall | F1 | TP | FP | TN | FN |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| Micro (summed confusion matrix) | 33716 | 0.9681 | 0.9569 | 0.9815 | 0.9691 | 16854 | 759 | 15786 | 317 |
| Macro (mean per-fold) | 33716 | 0.9685 | 0.9405 | 0.9846 | 0.9613 | — | — | — | — |

## Per-fold results

| Test fold | Accuracy | Precision | Recall | F1 | TP | FP | TN | FN | Ham misclassified |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| enron1 | 0.9611 | 0.8920 | 0.9853 | 0.9363 | 1478 | 179 | 3493 | 22 | 4.9% |
| enron2 | 0.9589 | 0.8638 | 0.9960 | 0.9252 | 1490 | 235 | 4126 | 6 | 5.4% |
| enron3 | 0.9819 | 0.9470 | 0.9887 | 0.9674 | 1483 | 83 | 3929 | 17 | 2.1% |
| enron4 | 0.9503 | 0.9702 | 0.9633 | 0.9668 | 4335 | 133 | 1367 | 165 | 8.9% |
| enron5 | 0.9853 | 0.9907 | 0.9886 | 0.9896 | 3633 | 34 | 1466 | 42 | 2.3% |
| enron6 | 0.9733 | 0.9790 | 0.9856 | 0.9823 | 4435 | 95 | 1405 | 65 | 6.3% |

"Ham misclassified" is FP / (FP + TN): the share of legitimate email that would be flagged as spam.

## Change from the previous version

The previous version computed the Laplace denominator $|V|$ over **all** tokens, including those dropped by `minCount`. Now $|V|$ and the per-class token totals cover only the kept vocabulary. Same 6 folds:

| Version | Accuracy | Precision | Recall | F1 | FP | FN |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| Before (\|V\| over all tokens) | 0.9681 | 0.9703 | 0.9670 | 0.9686 | 509 | 566 |
| After (\|V\| over kept tokens) | 0.9681 | 0.9569 | 0.9815 | 0.9691 | 759 | 317 |

Accuracy is unchanged, but the trade-off moved toward catching more spam at the cost of more false positives. The tokenizer fix (lengths counted in runes) changes nothing on this mostly-ASCII dataset: with only that fix applied, the summed confusion matrix is identical to before. Choosing the decision threshold for a target false-positive rate is in [`TODO.md`](TODO.md).

## Timing

Measured on a 12-thread machine (WSL2), `maxWorkers=8`, 5 runs of each built binary. Ranges are min–max over the runs:

| Stage | Before (re-tokenize per fold, sequential eval) | After (tokenize once, parallel eval) |
| --- | ---: | ---: |
| preprocess+ingest | 3.89–4.10 s | 0.69–0.90 s |
| merge per-folder bags (all folds) | — | 0.26–0.37 s |
| train (all folds) | 0.19–0.24 s | 0.10–0.16 s |
| classify+eval (all folds) | 1.42–1.55 s | 0.47–0.61 s |
| wall clock (whole run) | 5.50–7.01 s | 1.53–2.00 s |
