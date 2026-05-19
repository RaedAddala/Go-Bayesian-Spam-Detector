# Current Results

Command:

```bash
go run .
```

## Evaluation protocol

This project reports **6-fold cross-validation** over the Enron corpora (`enron1..enron6`): each `enron*` collection is used once as the test fold and the remaining 5 are used for training.

For each fold, the program prints:

- Train corpus summary (ham/spam doc counts, vocab size, `minCount`)
- Metrics: Accuracy, Precision, Recall, F1 + confusion-matrix counts
- Timing: preprocess+ingest, train, classify+eval

At the end, it prints aggregate metrics:

- **Micro**: summed confusion matrix across folds
- **Macro**: mean of per-fold metrics

(Commit/paste the latest output here if you want this file to track a particular run.)

## Aggregate results (6 folds)

| Scope | Total emails | Accuracy | Precision | Recall | F1 | TP | FP | TN | FN |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| Micro (summed confusion matrix) | 33716 | 0.9681 | 0.9703 | 0.9670 | 0.9686 | 16605 | 509 | 16036 | 566 |
| Macro (mean per-fold) | 33716 | 0.9684 | 0.9582 | 0.9714 | 0.9643 | — | — | — | — |

Total time (all folds):

- preprocess+ingest = $3.699164311$*s*
- train             = $260.943715$*ms*
- classify+eval     = $2.035116713$*s*
