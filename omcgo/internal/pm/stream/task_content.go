package stream

import (
	"crypto/sha256"
	"encoding/json"
	"sort"
)

type canonicalTaskContent struct {
	Enabled       bool              `json:"enabled"`
	Technology    string            `json:"technology"`
	Dimension     Dimension         `json:"dimension"`
	Granularities []Granularity     `json:"granularities"`
	ObjectLDNs    []string          `json:"object_ldns"`
	Metrics       []MetricRule      `json:"metrics"`
	Members       []canonicalMember `json:"members"`
}

type canonicalMember struct {
	DeviceID      string `json:"device_id"`
	DeviceSN      string `json:"device_sn"`
	DimensionKey  string `json:"dimension_key"`
	DimensionName string `json:"dimension_name"`
	ObjectLDN     string `json:"object_ldn"`
}

func taskContentHash(req SaveTaskRequest) []byte {
	content := canonicalTaskContent{
		Enabled: req.Enabled, Technology: req.Technology, Dimension: req.Dimension,
		Granularities: append([]Granularity(nil), req.Granularities...),
		ObjectLDNs:    append([]string(nil), req.ObjectLDNs...),
		Metrics:       append([]MetricRule(nil), req.Metrics...),
		Members:       make([]canonicalMember, 0, len(req.Members)),
	}
	sort.Slice(content.Granularities, func(i, j int) bool {
		return content.Granularities[i] < content.Granularities[j]
	})
	sort.Strings(content.ObjectLDNs)
	sort.Slice(content.Metrics, func(i, j int) bool {
		left, right := content.Metrics[i], content.Metrics[j]
		if left.MetricID != right.MetricID {
			return left.MetricID < right.MetricID
		}
		if left.MetricPath != right.MetricPath {
			return left.MetricPath < right.MetricPath
		}
		if left.MetricType != right.MetricType {
			return left.MetricType < right.MetricType
		}
		return left.Aggregation < right.Aggregation
	})
	for _, member := range req.Members {
		content.Members = append(content.Members, canonicalMember{
			DeviceID: member.DeviceID.String(), DeviceSN: member.DeviceSN,
			DimensionKey: member.DimensionKey, DimensionName: member.DimensionName,
			ObjectLDN: member.ObjectLDN,
		})
	}
	sort.Slice(content.Members, func(i, j int) bool {
		left, right := content.Members[i], content.Members[j]
		if left.DeviceID != right.DeviceID {
			return left.DeviceID < right.DeviceID
		}
		if left.DimensionKey != right.DimensionKey {
			return left.DimensionKey < right.DimensionKey
		}
		if left.ObjectLDN != right.ObjectLDN {
			return left.ObjectLDN < right.ObjectLDN
		}
		if left.DeviceSN != right.DeviceSN {
			return left.DeviceSN < right.DeviceSN
		}
		return left.DimensionName < right.DimensionName
	})
	encoded, err := json.Marshal(content)
	if err != nil {
		panic("marshal canonical PM aggregation task content: " + err.Error())
	}
	sum := sha256.Sum256(encoded)
	return sum[:]
}
