package record

type Model struct {
	Opens   []float64 `json:"opens"`
	Highs   []float64 `json:"highs"`
	Lows    []float64 `json:"lows"`
	Closes  []float64 `json:"closes"`
	Volumes []float64 `json:"volumes"`
	VWAPs   []float64 `json:"vwaps"`
}

func MarketToModel(records []Market) Model {
	m := Model{
		Opens:   make([]float64, len(records)),
		Highs:   make([]float64, len(records)),
		Lows:    make([]float64, len(records)),
		Closes:  make([]float64, len(records)),
		Volumes: make([]float64, len(records)),
		VWAPs:   make([]float64, len(records)),
	}
	for i, rec := range records {
		m.Opens[i] = rec.Open
		m.Highs[i] = rec.High
		m.Lows[i] = rec.Low
		m.Closes[i] = rec.Close
		m.Volumes[i] = rec.Volume
		m.VWAPs[i] = rec.VWAP
	}
	return m
}
