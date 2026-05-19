package main

import (
	"fmt"
	"log"
	"path/filepath"
	"time"
)

func main() {
	const (
		baseDir    = "./data"
		maxWorkers = 8
		minCount   = 20
	)

	folds := []string{"enron1", "enron2", "enron3", "enron4", "enron5", "enron6"}

	fmt.Printf("6-fold cross-validation over %v\n\n", folds)

	perFold := make([]Metrics, 0, len(folds))
	var sum Metrics

	var totalPreprocess, totalTrain, totalClassify time.Duration

	for i, testFold := range folds {
		trainFolds := make([]string, 0, len(folds)-1)
		for _, f := range folds {
			if f != testFold {
				trainFolds = append(trainFolds, f)
			}
		}

		fmt.Printf("Fold %d/%d  test=%s  train=%v\n", i+1, len(folds), testFold, trainFolds)

		// Preprocessing + ingestion
		preStart := time.Now()
		hamBag, spamBag, numHam, numSpam, err := ingestFolders(baseDir, trainFolds, maxWorkers)
		if err != nil {
			log.Fatalf("Ingestion failed (test=%s): %v", testFold, err)
		}
		preDur := time.Since(preStart)

		// Training
		trainStart := time.Now()
		model := NewModel(hamBag, spamBag, numHam, numSpam, minCount)
		trainDur := time.Since(trainStart)

		// Classification
		evalStart := time.Now()
		testDir := filepath.Join(baseDir, testFold)
		met, err := evaluateModel(model, testDir)
		if err != nil {
			log.Fatalf("Evaluation failed (test=%s): %v", testFold, err)
		}
		evalDur := time.Since(evalStart)

		perFold = append(perFold, met)
		sum = sum.Add(met)

		totalPreprocess += preDur
		totalTrain += trainDur
		totalClassify += evalDur

		fmt.Printf("  Train corpus: %d ham docs, %d spam docs  |V|=%d (minCount=%d)\n", numHam, numSpam, model.vocabSize, minCount)
		fmt.Printf("  Metrics: Acc=%.4f  Prec=%.4f  Rec=%.4f  F1=%.4f  (TP=%d FP=%d TN=%d FN=%d)\n",
			met.Accuracy(), met.Precision(), met.Recall(), met.F1(), met.TP, met.FP, met.TN, met.FN)
		fmt.Printf("  Time: preprocess+ingest=%s  train=%s  classify+eval=%s\n\n", preDur, trainDur, evalDur)
	}

	means := meanMetrics(perFold)

	fmt.Println("Aggregate over all folds:")
	fmt.Printf("  Total emails: %d\n", sum.Total)
	fmt.Printf("  Micro (summed confusion matrix):\n")
	fmt.Printf("    Accuracy  : %.4f\n", sum.Accuracy())
	fmt.Printf("    Precision : %.4f\n", sum.Precision())
	fmt.Printf("    Recall    : %.4f\n", sum.Recall())
	fmt.Printf("    F1        : %.4f\n", sum.F1())
	fmt.Printf("    TP=%d  FP=%d  TN=%d  FN=%d\n", sum.TP, sum.FP, sum.TN, sum.FN)
	fmt.Printf("  Macro (mean per-fold): Acc=%.4f  Prec=%.4f  Rec=%.4f  F1=%.4f\n", means.Accuracy, means.Precision, means.Recall, means.F1)
	fmt.Printf("  Total time: preprocess+ingest=%s  train=%s  classify+eval=%s\n", totalPreprocess, totalTrain, totalClassify)
}
