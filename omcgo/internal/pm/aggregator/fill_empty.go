package aggregator

import (
	"strings"
	"time"

	"github.com/omcgo/omcgo/internal/pm/metrics"
)

// FillEmptyBuckets 数据驱动补齐占位行（T-0192d / #54）。
//
// 语义：判断单位 = 一条测量记录身份 = (object_ldn, 时间桶)。默认沿用旧行为：只遍历查询
// 已返回的真实行（Filled=false），按 (object_ldn 归一, Time) 分组；对每个**已存在**的分组，
// req.MetricPaths 里缺失的指标补一行占位。
//
// 当请求带 object_ldns + start/end + granularity 时，object_ldns 是本次展示全集：
// 它可以来自用户显式选择，也可以来自后端自动发现出的"全部小区"对象集。此时按查询条件补齐
// (object_ldn × 时间桶 × 指标) 骨架，缺数据 object 不应从结果消失。
//
// 仅 device 维度（单 SN）+ metric_paths 非空时启用；多 SN / 组维度原样返回。
func FillEmptyBuckets(rows []Row, req QueryRequest) []Row {
	if req.Dimension == DimensionDeviceGroup || req.Dimension == DimensionAggregateGroup {
		return rows
	}
	if len(req.MetricPaths) == 0 {
		return rows
	}
	if len(req.DeviceSNs) != 1 {
		return rows
	}

	nameByPath := make(map[string]string, len(req.MetricPaths))
	realByKeyMetric := make(map[string]map[string]struct{})

	type group struct {
		rep  Row
		have map[string]struct{}
	}
	groups := make(map[string]*group)
	order := make([]string, 0)

	for _, r := range rows {
		if r.Filled {
			continue
		}
		if req.MetricType != nil && r.MetricType != *req.MetricType {
			continue
		}
		if r.DisplayName != "" {
			nameByPath[r.MetricPath] = r.DisplayName
		}
		key := fillBucketKey(r.ObjectLDN, r.Time)
		seen := realByKeyMetric[key]
		if seen == nil {
			seen = make(map[string]struct{})
			realByKeyMetric[key] = seen
		}
		seen[r.MetricPath] = struct{}{}
		g := groups[key]
		if g == nil {
			g = &group{rep: r, have: make(map[string]struct{})}
			groups[key] = g
			order = append(order, key)
		}
		g.have[r.MetricPath] = struct{}{}
	}

	for _, key := range order {
		g := groups[key]
		for _, mp := range req.MetricPaths {
			if _, ok := g.have[mp]; ok {
				continue
			}
			rows = append(rows, Row{
				DeviceOUI:   g.rep.DeviceOUI,
				DeviceSN:    g.rep.DeviceSN,
				MetricPath:  mp,
				DisplayName: nameByPath[mp],
				MetricType:  fillMetricType(g.rep.MetricType, req.MetricType),
				Granularity: g.rep.Granularity,
				Time:        g.rep.Time,
				StartTime:   g.rep.StartTime,
				EndTime:     g.rep.EndTime,
				ObjectLDN:   g.rep.ObjectLDN,
				Filled:      true,
			})
		}
	}

	if !IsExplicitObjectSkeletonRequest(req) {
		return rows
	}
	template := skeletonTemplateRow(rows, req)
	for _, bucket := range skeletonBuckets(req) {
		for _, objectLDN := range req.ObjectLDNs {
			ldn := objectLDN
			key := fillBucketKey(&ldn, bucket)
			seen := realByKeyMetric[key]
			if seen == nil {
				seen = make(map[string]struct{})
				realByKeyMetric[key] = seen
			}
			for _, mp := range req.MetricPaths {
				if _, ok := seen[mp]; ok {
					continue
				}
				rows = append(rows, Row{
					DeviceOUI:   template.DeviceOUI,
					DeviceSN:    template.DeviceSN,
					MetricPath:  mp,
					DisplayName: nameByPath[mp],
					MetricType:  skeletonMetricType(mp, template.MetricType, req.MetricType),
					Granularity: req.Granularity,
					Time:        bucket,
					StartTime:   bucket,
					EndTime:     nextSkeletonBucket(bucket, req.Granularity),
					ObjectLDN:   &ldn,
					Filled:      true,
				})
				seen[mp] = struct{}{}
			}
		}
	}
	return rows
}

func fillMetricType(repType metrics.MetricType, requested *metrics.MetricType) metrics.MetricType {
	if requested != nil {
		return *requested
	}
	return repType
}

func IsExplicitObjectSkeletonRequest(req QueryRequest) bool {
	return isDeviceObjectSkeletonBaseRequest(req) &&
		len(req.ObjectLDNs) > 0
}

func CanAutoDiscoverObjectSkeletonRequest(req QueryRequest) bool {
	return isDeviceObjectSkeletonBaseRequest(req) &&
		len(req.ObjectLDNs) == 0
}

func isDeviceObjectSkeletonBaseRequest(req QueryRequest) bool {
	return req.Dimension != DimensionDeviceGroup &&
		req.Dimension != DimensionAggregateGroup &&
		len(req.DeviceSNs) == 1 &&
		len(req.MetricPaths) > 0 &&
		req.Granularity != "" &&
		!req.StartTime.IsZero() &&
		req.EndTime.After(req.StartTime)
}

func skeletonTemplateRow(rows []Row, req QueryRequest) Row {
	if len(rows) > 0 {
		return rows[0]
	}
	row := Row{Granularity: req.Granularity}
	if len(req.DeviceOUIs) > 0 {
		row.DeviceOUI = req.DeviceOUIs[0]
	}
	if len(req.DeviceSNs) > 0 {
		row.DeviceSN = req.DeviceSNs[0]
	}
	if req.MetricType != nil {
		row.MetricType = *req.MetricType
	}
	return row
}

func skeletonMetricType(metricPath string, fallback metrics.MetricType, requested *metrics.MetricType) metrics.MetricType {
	if requested != nil {
		return *requested
	}
	if strings.HasPrefix(metricPath, "K") {
		return metrics.MetricTypeKPI
	}
	if fallback != "" {
		return fallback
	}
	return metrics.MetricTypeCounter
}

func skeletonBuckets(req QueryRequest) []time.Time {
	if !knownSkeletonGranularity(req.Granularity) {
		return nil
	}
	out := make([]time.Time, 0)
	for t := req.StartTime; t.Before(req.EndTime); t = nextSkeletonBucket(t, req.Granularity) {
		if !bucketPassesCalendarFilters(t, req) {
			continue
		}
		out = append(out, t)
	}
	return out
}

func knownSkeletonGranularity(gran metrics.Granularity) bool {
	switch gran {
	case metrics.Granularity15Min, metrics.GranularityHourly, metrics.GranularityDaily, metrics.GranularityWeekly, metrics.GranularityMonthly:
		return true
	default:
		return false
	}
}

func nextSkeletonBucket(t time.Time, gran metrics.Granularity) time.Time {
	switch gran {
	case metrics.Granularity15Min:
		return t.Add(15 * time.Minute)
	case metrics.GranularityHourly:
		return t.Add(time.Hour)
	case metrics.GranularityDaily:
		return t.AddDate(0, 0, 1)
	case metrics.GranularityWeekly:
		return t.AddDate(0, 0, 7)
	case metrics.GranularityMonthly:
		return t.AddDate(0, 1, 0)
	default:
		return t
	}
}

func bucketPassesCalendarFilters(t time.Time, req QueryRequest) bool {
	if len(req.Weekdays) > 0 && len(req.Weekdays) < 7 && !intInSlice(int(t.Weekday()), req.Weekdays) {
		return false
	}
	if len(req.Hours) > 0 && len(req.Hours) < 24 && !intInSlice(t.Hour(), req.Hours) {
		return false
	}
	return true
}

func intInSlice(v int, vals []int) bool {
	for _, x := range vals {
		if x == v {
			return true
		}
	}
	return false
}

func fillBucketKey(objectLDN *string, t time.Time) string {
	ldn := "\x00"
	if objectLDN != nil {
		ldn = *objectLDN
	}
	return ldn + "||" + t.UTC().Format(time.RFC3339Nano)
}
