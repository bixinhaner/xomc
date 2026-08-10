package geofence

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	thirdPartyLocationMaxDevices = 1000
	thirdPartyLocationBatchSize  = 100
	thirdPartyTimeLayout         = "2006-01-02 15:04:05"
)

type ThirdPartyLocationDevice struct {
	VesselName   string `json:"vesselName"`
	SerialNumber string `json:"serialNumber"`
	Longitude    string `json:"longitude"`
	Latitude     string `json:"latitude"`
	UpdateTime   string `json:"updateTime"`
}

type ThirdPartyLocationRequest struct {
	Devices []ThirdPartyLocationDevice `json:"devices"`
}

type ThirdPartyLocationFailure struct {
	SerialNumber string `json:"serialNumber"`
	Reason       string `json:"reason"`
}

type ThirdPartyLocationResult struct {
	TotalCount    int                         `json:"totalCount"`
	SuccessCount  int                         `json:"successCount"`
	FailCount     int                         `json:"failCount"`
	FailedDevices []ThirdPartyLocationFailure `json:"failedDevices"`
}

var ErrThirdPartyLocationBatchInProgress = fmt.Errorf("third-party location batch is already processing")
var ErrThirdPartyLocationBatchConflict = fmt.Errorf("third-party location idempotency key conflicts with another request")

type ThirdPartyLocationBatchRepository interface {
	Claim(context.Context, string, string) (*ThirdPartyLocationResult, error)
	Complete(context.Context, string, ThirdPartyLocationResult) error
}

type thirdPartyLocationItemResult struct {
	success bool
	failure ThirdPartyLocationFailure
}

type ThirdPartyLocationDeviceStore interface {
	SaveThirdPartyLocation(
		context.Context,
		ThirdPartyLocationDevice,
		time.Time,
		float64,
		float64,
	) error
}

func validateThirdPartyLocationRequest(
	request ThirdPartyLocationRequest,
) error {
	if len(request.Devices) == 0 {
		return fmt.Errorf("devices must not be empty")
	}
	if len(request.Devices) > thirdPartyLocationMaxDevices {
		return fmt.Errorf("devices must not exceed %d items", thirdPartyLocationMaxDevices)
	}
	return nil
}

func parseThirdPartyLocation(
	item ThirdPartyLocationDevice,
) (time.Time, float64, float64, error) {
	if strings.TrimSpace(item.VesselName) == "" || len([]rune(item.VesselName)) > 128 {
		return time.Time{}, 0, 0, fmt.Errorf("vesselName is required and must not exceed 128 characters")
	}
	if strings.TrimSpace(item.SerialNumber) == "" || len([]rune(item.SerialNumber)) > 64 {
		return time.Time{}, 0, 0, fmt.Errorf("serialNumber is required and must not exceed 64 characters")
	}
	longitude, err := parseThirdPartyCoordinate(item.Longitude, -180, 180)
	if err != nil {
		return time.Time{}, 0, 0, fmt.Errorf("invalid longitude: %w", err)
	}
	latitude, err := parseThirdPartyCoordinate(item.Latitude, -90, 90)
	if err != nil {
		return time.Time{}, 0, 0, fmt.Errorf("invalid latitude: %w", err)
	}
	observedAt, err := time.ParseInLocation(
		thirdPartyTimeLayout,
		strings.TrimSpace(item.UpdateTime),
		time.UTC,
	)
	if err != nil {
		return time.Time{}, 0, 0, fmt.Errorf("invalid updateTime: expected %s", thirdPartyTimeLayout)
	}
	return observedAt, longitude, latitude, nil
}

func normalizeThirdPartyLocationDevice(
	item ThirdPartyLocationDevice,
) ThirdPartyLocationDevice {
	item.VesselName = strings.TrimSpace(item.VesselName)
	item.SerialNumber = strings.TrimSpace(item.SerialNumber)
	item.Longitude = strings.TrimSpace(item.Longitude)
	item.Latitude = strings.TrimSpace(item.Latitude)
	item.UpdateTime = strings.TrimSpace(item.UpdateTime)
	return item
}

func parseThirdPartyCoordinate(raw string, min, max float64) (float64, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return 0, fmt.Errorf("value is required")
	}
	value, err := strconv.ParseFloat(trimmed, 64)
	if err != nil || math.IsNaN(value) || math.IsInf(value, 0) || value < min || value > max {
		return 0, fmt.Errorf("value is outside [%g,%g]", min, max)
	}
	return value, nil
}

func (s *Service) UpdateThirdPartyLocations(
	ctx context.Context,
	request ThirdPartyLocationRequest,
) (ThirdPartyLocationResult, error) {
	if err := validateThirdPartyLocationRequest(request); err != nil {
		return ThirdPartyLocationResult{}, err
	}
	if s.thirdPartyLocationStore == nil {
		return ThirdPartyLocationResult{}, fmt.Errorf("third-party location store is not configured")
	}

	result := ThirdPartyLocationResult{
		TotalCount:    len(request.Devices),
		FailedDevices: make([]ThirdPartyLocationFailure, 0),
	}
	itemResults := make([]thirdPartyLocationItemResult, len(request.Devices))
	for start := 0; start < len(request.Devices); start += thirdPartyLocationBatchSize {
		end := start + thirdPartyLocationBatchSize
		if end > len(request.Devices) {
			end = len(request.Devices)
		}
		var wg sync.WaitGroup
		for offset, item := range request.Devices[start:end] {
			item := normalizeThirdPartyLocationDevice(item)
			index := start + offset
			wg.Add(1)
			go func() {
				defer wg.Done()
				observedAt, longitude, latitude, err := parseThirdPartyLocation(item)
				if err == nil && duplicateSerialNumber(request.Devices, index, item.SerialNumber) {
					err = fmt.Errorf("duplicate serialNumber")
				}
				if err == nil {
					err = s.thirdPartyLocationStore.SaveThirdPartyLocation(
						ctx, item, observedAt, longitude, latitude,
					)
				}
				if err != nil {
					itemResults[index] = thirdPartyLocationItemResult{failure: ThirdPartyLocationFailure{
						SerialNumber: item.SerialNumber,
						Reason:       publicThirdPartyLocationError(err),
					}}
					return
				}
				itemResults[index] = thirdPartyLocationItemResult{success: true}
			}()
		}
		wg.Wait()
	}
	for _, itemResult := range itemResults {
		if itemResult.success {
			result.SuccessCount++
			continue
		}
		result.FailCount++
		result.FailedDevices = append(result.FailedDevices, itemResult.failure)
	}
	return result, nil
}

func (s *Service) UpdateThirdPartyLocationsIdempotent(
	ctx context.Context,
	request ThirdPartyLocationRequest,
	idempotencyKey string,
) (ThirdPartyLocationResult, bool, error) {
	idempotencyKey = strings.TrimSpace(idempotencyKey)
	if idempotencyKey == "" {
		result, err := s.UpdateThirdPartyLocations(ctx, request)
		return result, false, err
	}
	if s.thirdPartyLocationBatchRepository == nil {
		return ThirdPartyLocationResult{}, false, fmt.Errorf("third-party location batch repository is not configured")
	}
	claimed, err := s.thirdPartyLocationBatchRepository.Claim(
		ctx,
		idempotencyKey,
		thirdPartyLocationRequestHash(request),
	)
	if err != nil {
		return ThirdPartyLocationResult{}, false, err
	}
	if claimed != nil {
		return *claimed, true, nil
	}
	result, err := s.UpdateThirdPartyLocations(ctx, request)
	if err != nil {
		return ThirdPartyLocationResult{}, false, err
	}
	if err := s.thirdPartyLocationBatchRepository.Complete(ctx, idempotencyKey, result); err != nil {
		return ThirdPartyLocationResult{}, false, fmt.Errorf("complete third-party location batch: %w", err)
	}
	return result, false, nil
}

func thirdPartyLocationRequestHash(request ThirdPartyLocationRequest) string {
	data, _ := json.Marshal(request)
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func duplicateSerialNumber(
	items []ThirdPartyLocationDevice,
	index int,
	serialNumber string,
) bool {
	serialNumber = strings.TrimSpace(serialNumber)
	for previous := 0; previous < index; previous++ {
		if strings.TrimSpace(items[previous].SerialNumber) == serialNumber {
			return true
		}
	}
	return false
}

func publicThirdPartyLocationError(err error) string {
	message := strings.TrimSpace(err.Error())
	if message == "" {
		return "location update failed"
	}
	return message
}
