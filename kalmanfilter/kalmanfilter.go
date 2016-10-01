package kalmanfilter

import (
	"math"
)

type FilterData struct {
	R float64 // 1
	Q float64 // 1

	A float64 // 1
	B float64 // 0
	C float64 // 1

	cov   float64
	state float64 // estimated
}

func (filterData *FilterData) Update(z float64) float64 {

	x := 0.0

	if math.IsNaN(filterData.state) {
		filterData.state = (1.0 / filterData.C) * z
		filterData.cov = (1.0 / filterData.C) * filterData.Q * (1.0 / filterData.C)
	} else {
		// Compute prediction
		pred := (filterData.A * filterData.state) + (filterData.B * x)
		predCov := ((filterData.A * filterData.cov) * filterData.A) + filterData.R

		// Kalman gain
		K := predCov * filterData.C * (1.0 / ((filterData.C * predCov * filterData.C) + filterData.Q))

		// Correction
		filterData.state = pred + K*(z-(filterData.C*pred))
		filterData.cov = predCov - (K * filterData.C * predCov)
	}

	return filterData.state
}
