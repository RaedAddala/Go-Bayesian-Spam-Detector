package main

type Metrics struct {
	Total   int
	Correct int
	TP      int // spam predicted spam
	FP      int // ham predicted spam
	TN      int // ham predicted ham
	FN      int // spam predicted ham
}

func (met Metrics) Accuracy() float64 { return float64(met.Correct) / float64(met.Total) }
func (met Metrics) Precision() float64 {
	if met.TP+met.FP == 0 {
		return 0
	}
	return float64(met.TP) / float64(met.TP+met.FP)
}
func (met Metrics) Recall() float64 {
	if met.TP+met.FN == 0 {
		return 0
	}
	return float64(met.TP) / float64(met.TP+met.FN)
}
func (met Metrics) F1() float64 {
	p, r := met.Precision(), met.Recall()
	if p+r == 0 {
		return 0
	}
	return 2 * p * r / (p + r)
}

// Add returns a metrics value where all counters are summed.
func (met Metrics) Add(o Metrics) Metrics {
	return Metrics{
		Total:   met.Total + o.Total,
		Correct: met.Correct + o.Correct,
		TP:      met.TP + o.TP,
		FP:      met.FP + o.FP,
		TN:      met.TN + o.TN,
		FN:      met.FN + o.FN,
	}
}

// MeanMetrics are mean (per-fold) derived metrics, computed from per-fold Metrics.
// Note: confusion matrix counts are not averaged here; use the summed Metrics if needed.
type MeanMetrics struct {
	Accuracy  float64
	Precision float64
	Recall    float64
	F1        float64
}

func meanMetrics(folds []Metrics) MeanMetrics {
	if len(folds) == 0 {
		return MeanMetrics{}
	}
	var sumA, sumP, sumR, sumF float64
	for _, m := range folds {
		sumA += m.Accuracy()
		sumP += m.Precision()
		sumR += m.Recall()
		sumF += m.F1()
	}
	d := float64(len(folds))
	return MeanMetrics{Accuracy: sumA / d, Precision: sumP / d, Recall: sumR / d, F1: sumF / d}
}
