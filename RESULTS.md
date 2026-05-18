# Current Results

Command:

```bash
go run .
```

## Dataset summary

| Item | Value |
| --- | ---: |
| Training folders | `enron1..enron5` |
| Test folder | `enron6` |
| Training ham docs | 15045 |
| Training spam docs | 12671 |
| Vocab size (`minCount=20`) | 135169 |
| log-prior ham | -0.6110 |
| log-prior spam | -0.7827 |
| Test emails | 6000 |

## Metrics (enron6)

| Metric | Value |
| --- | ---: |
| Accuracy | 0.9718 |
| Precision | 0.9892 |
| Recall | 0.9731 |
| F1 | 0.9811 |

## Confusion matrix

| | Predicted spam | Predicted ham |
| --- | ---: | ---: |
| Actual spam | TP = 4379 | FN = 121 |
| Actual ham | FP = 48 | TN = 1452 |
