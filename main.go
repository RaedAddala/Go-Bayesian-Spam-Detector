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

	// Preprocessing + ingestion: tokenize every folder exactly once.
	// Each fold's training corpus is then the sum of the other folders' corpora.
	preStart := time.Now()
	corpora := make(map[string]Corpus, len(folds))
	for _, f := range folds {
		c, err := ingestFolder(baseDir, f, maxWorkers)
		if err != nil {
			log.Fatalf("Ingestion failed: %v", err)
		}
		corpora[f] = c
	}
	preDur := time.Since(preStart)
	fmt.Printf("Preprocess+ingest (all folders, once): %s\n\n", preDur)

	perFold := make([]Metrics, 0, len(folds))
	var sum Metrics

	var totalMerge, totalTrain, totalClassify time.Duration

	for i, testFold := range folds {
		trainFolds := make([]string, 0, len(folds)-1)
		trainParts := make([]Corpus, 0, len(folds)-1)
		for _, f := range folds {
			if f != testFold {
				trainFolds = append(trainFolds, f)
				trainParts = append(trainParts, corpora[f])
			}
		}

		fmt.Printf("Fold %d/%d  test=%s  train=%v\n", i+1, len(folds), testFold, trainFolds)

		// Merge the cached per-folder bags into this fold's training corpus
		mergeStart := time.Now()
		train := mergeCorpora(trainParts...)
		mergeDur := time.Since(mergeStart)

		// Training
		trainStart := time.Now()
		model, err := NewModel(train.HamBag, train.SpamBag, train.NumHam, train.NumSpam, minCount)
		if err != nil {
			log.Fatalf("Training failed (test=%s): %v", testFold, err)
		}
		trainDur := time.Since(trainStart)

		// Classification
		evalStart := time.Now()
		testDir := filepath.Join(baseDir, testFold)
		met, err := evaluateModel(model, testDir, maxWorkers)
		if err != nil {
			log.Fatalf("Evaluation failed (test=%s): %v", testFold, err)
		}
		evalDur := time.Since(evalStart)

		perFold = append(perFold, met)
		sum = sum.Add(met)

		totalMerge += mergeDur
		totalTrain += trainDur
		totalClassify += evalDur

		fmt.Printf("  Train corpus: %d ham docs, %d spam docs  |V|=%d (minCount=%d)\n", train.NumHam, train.NumSpam, model.vocabSize, minCount)
		fmt.Printf("  Metrics: Acc=%.4f  Prec=%.4f  Rec=%.4f  F1=%.4f  (TP=%d FP=%d TN=%d FN=%d)\n",
			met.Accuracy(), met.Precision(), met.Recall(), met.F1(), met.TP, met.FP, met.TN, met.FN)
		fmt.Printf("  Time: merge=%s  train=%s  classify+eval=%s\n\n", mergeDur, trainDur, evalDur)
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
	fmt.Printf("  Total time: preprocess+ingest=%s  merge=%s  train=%s  classify+eval=%s\n", preDur, totalMerge, totalTrain, totalClassify)
}
