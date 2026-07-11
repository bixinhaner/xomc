// Package resultnorm centralizes PM result value normalization.
package resultnorm

import (
	"errors"
	"math"
	"strings"
)

const (
	ConfigCategory = "indicator.process"
	ConfigKey      = "number"
	ConfigName     = ConfigCategory + "." + ConfigKey
)

const (
	NumberProcessIntUp     = "intUp"
	NumberProcessIntDown   = "intDown"
	NumberProcessIntHalfUp = "intHalfUp"
	NumberProcessNone      = "none"
	DefaultNumberProcess   = NumberProcessIntHalfUp
)

var (
	ErrMissingMetadata = errors.New("pm result normalization: missing indicator metadata")
	ErrInvalidValue    = errors.New("pm result normalization: invalid result value")
)

type Metadata struct {
	Unit       string
	StatisType string
}

func Normalize(value float64, metadata *Metadata, numberProcess string) (float64, error) {
	if metadata == nil {
		return 0, ErrMissingMetadata
	}
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return 0, ErrInvalidValue
	}

	unit := strings.TrimSpace(metadata.Unit)
	statisType := strings.ToLower(strings.TrimSpace(metadata.StatisType))
	if unit == "" || statisType == "" {
		return 0, ErrMissingMetadata
	}

	process := strings.TrimSpace(numberProcess)
	if process == "" {
		process = DefaultNumberProcess
	}

	if unit == "%" || statisType == "pct" || statisType == "avg" {
		value = round2(value)
		if unit == "%" && value > 100 {
			return 100, nil
		}
		return value, nil
	}

	// avg/pct 是连续量统计，必须先于 unit=number 的整数化处理。尤其设备数、连接数等
	// 原始 counter 虽以 number 为单位，跨桶平均值仍可能是小数；先整数化会让 daily avg
	// 产生系统性偏差（#32）。只有非 avg/pct 的 number 结果才服从 intUp/intDown/intHalfUp。
	if unit == "number" && process != NumberProcessNone {
		return normalizeInteger(value, process), nil
	}

	if isInteger(value) {
		return value, nil
	}
	return round2(value), nil
}

func normalizeInteger(value float64, process string) float64 {
	switch process {
	case NumberProcessIntUp:
		return math.Ceil(value)
	case NumberProcessIntDown:
		return math.Floor(value)
	case NumberProcessIntHalfUp:
		return math.Round(value)
	default:
		return math.Round(value)
	}
}

func round2(value float64) float64 {
	return math.Round(value*100) / 100
}

func isInteger(value float64) bool {
	return math.Trunc(value) == value
}
