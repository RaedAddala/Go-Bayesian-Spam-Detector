package main

import (
	"fmt"
	"log"
	"path/filepath"
)

func main() {
	const (
		baseDir    = "./data"
		maxWorkers = 8
		minCount   = 20
	)

	trainFolders := []string{"enron1", "enron2", "enron3", "enron4", "enron5"}
	testFolder := filepath.Join(baseDir, "enron6")

	fmt.Println("Ingesting training folders…")
	hamBag, spamBag, numHam, numSpam, err := ingestFolders(baseDir, trainFolders, maxWorkers)
	if err != nil {
		log.Fatalf("Ingestion failed: %v", err)
	}
	fmt.Printf("Training corpus: %d ham docs, %d spam docs\n", numHam, numSpam)

	model := NewModel(hamBag, spamBag, numHam, numSpam, minCount)
	fmt.Printf("Vocab size: %d  (minCount=%d)\n", model.vocabSize, minCount)
	fmt.Printf("log-prior  ham=%.4f  spam=%.4f\n\n", model.logPriorHam, model.logPriorSpam)

	fmt.Println("Evaluating on enron6…")
	met, err := evaluateModel(model, testFolder)
	if err != nil {
		log.Fatalf("Evaluation failed: %v", err)
	}

	fmt.Printf("Results on %d emails:\n", met.Total)
	fmt.Printf("Accuracy  : %.4f\n", met.Accuracy())
	fmt.Printf("Precision : %.4f\n", met.Precision())
	fmt.Printf("Recall    : %.4f\n", met.Recall())
	fmt.Printf("F1        : %.4f\n", met.F1())
	fmt.Printf("TP=%d  FP=%d  TN=%d  FN=%d\n", met.TP, met.FP, met.TN, met.FN)
}
