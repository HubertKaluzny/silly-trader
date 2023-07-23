package record

import "errors"

type CombineStrategy string

const (
	InterleaveCombine CombineStrategy = "interleave"
	ConcatCombine     CombineStrategy = "concat"
)

func ToCombineStrategy(input string) (CombineStrategy, error) {
	switch input {
	case string(InterleaveCombine):
		return InterleaveCombine, nil
	case string(ConcatCombine):
		return ConcatCombine, nil
	}
	return InterleaveCombine, errors.New("invalid combine strategy specified")
}

func CombineModels(x1, x2 Model, strat CombineStrategy) (*Model, error) {
	var combined Model
	switch strat {
	case InterleaveCombine:
		interleaved, err := InterleaveModels(x1, x2)
		if err != nil {
			return nil, err
		}
		combined = *interleaved
	case ConcatCombine:
		combined = ConcatModels(x1, x2)
	default:
		return nil, errors.New("invalid combine strategy specified")
	}
	return &combined, nil
}

func interleaveSplices(x1, x2 []float64) ([]float64, error) {
	if len(x1) != len(x2) {
		return nil, errors.New("cannot combine two unequal length sets")
	}
	length := len(x1) + len(x2)
	res := make([]float64, length, length)
	for i := 0; i < length; i += 2 {
		res[i] = x1[i/2]
		res[i+1] = x2[i/2]
	}
	return res, nil
}

func ConcatModels(x1, x2 Model) Model {
	resModel := Model{
		Opens:   append(x1.Opens, x2.Opens...),
		Highs:   append(x1.Highs, x2.Highs...),
		Lows:    append(x1.Lows, x2.Lows...),
		Closes:  append(x1.Closes, x2.Closes...),
		Volumes: append(x1.Volumes, x2.Volumes...),
		VWAPs:   append(x1.VWAPs, x2.VWAPs...),
	}
	return resModel
}

func InterleaveModels(x1, x2 Model) (*Model, error) {
	var resModel Model
	var err error
	resModel.Opens, err = interleaveSplices(x1.Opens, x2.Opens)
	if err != nil {
		return nil, err
	}
	resModel.Highs, err = interleaveSplices(x1.Highs, x2.Highs)
	if err != nil {
		return nil, err
	}
	resModel.Lows, err = interleaveSplices(x1.Lows, x2.Lows)
	if err != nil {
		return nil, err
	}
	resModel.Closes, err = interleaveSplices(x1.Closes, x2.Closes)
	if err != nil {
		return nil, err
	}
	resModel.Volumes, err = interleaveSplices(x1.Volumes, x2.Volumes)
	if err != nil {
		return nil, err
	}
	resModel.VWAPs, err = interleaveSplices(x1.VWAPs, x2.VWAPs)
	if err != nil {
		return nil, err
	}
	return &resModel, nil
}
