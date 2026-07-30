package backtest

import "invest/pkg/indicator"

// indicatorFieldAt extracts an indicator field value at index i from a Series.
func indicatorFieldAt(s indicator.Series, field IndField, i int) float64 {
	switch field {
	case FieldMA5:
		return s.MA5[i]
	case FieldMA20:
		return s.MA20[i]
	case FieldMA60:
		return s.MA60[i]
	case FieldRSI14:
		return s.RSI14[i]
	case FieldBollUpper:
		return s.BollUpper[i]
	case FieldBollMid:
		return s.BollMid[i]
	case FieldBollLower:
		return s.BollLower[i]
	case FieldMACDLine:
		return s.MACDLine[i]
	case FieldMACDSignal:
		return s.MACDSignal[i]
	case FieldMACDHist:
		return s.MACDHist[i]
	default:
		return 0
	}
}

func indicatorFieldLen(s indicator.Series, field IndField) int {
	switch field {
	case FieldMA5:
		return len(s.MA5)
	case FieldMA20:
		return len(s.MA20)
	case FieldMA60:
		return len(s.MA60)
	case FieldRSI14:
		return len(s.RSI14)
	case FieldBollUpper:
		return len(s.BollUpper)
	case FieldBollMid:
		return len(s.BollMid)
	case FieldBollLower:
		return len(s.BollLower)
	case FieldMACDLine:
		return len(s.MACDLine)
	case FieldMACDSignal:
		return len(s.MACDSignal)
	case FieldMACDHist:
		return len(s.MACDHist)
	default:
		return -1
	}
}
