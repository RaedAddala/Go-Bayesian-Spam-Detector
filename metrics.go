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
