package device

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/pkg/tr069"
)

// UPSRuntimeRepository persists UPS-specific list/detail projections.
// The raw TR-069 values still live in device_parameters; this projection only
// keeps high-frequency UPS fields out of the radio-oriented device_info table.
type UPSRuntimeRepository interface {
	UpsertFromInform(ctx context.Context, device *model.Device, inform *tr069.InformMessage) error
	GetDetail(ctx context.Context, device *model.Device) (*UPSDetail, error)
}

type PgUPSRuntimeRepository struct {
	pool *pgxpool.Pool
}

func NewPgUPSRuntimeRepository(pool *pgxpool.Pool) *PgUPSRuntimeRepository {
	return &PgUPSRuntimeRepository{pool: pool}
}

func (r *PgUPSRuntimeRepository) GetDetail(ctx context.Context, device *model.Device) (*UPSDetail, error) {
	if r == nil || r.pool == nil || device == nil {
		return nil, nil
	}

	detail := &UPSDetail{
		PowerSystem: UPSPowerSystemParam{
			Manufacturer:    stringPtrIfNotBlank(device.Manufacturer),
			ManufacturerOUI: stringPtrIfNotBlank(device.OUI),
			SerialNumber:    device.SerialNumber,
			SoftwareVersion: stringPtrIfNotBlank(device.FirmwareVersion),
			ProductClass:    device.ProductClass,
		},
		BatteryRun: []UPSBatteryRuntimeParam{},
	}

	var (
		externalIP, totalVoltage, totalTemperature, totalCurrent                *string
		softwareVersion, hardwareVersion, manufacturer, manufacturerOUI         *string
		bmsCharging, acPower, acVoltage, dcVoltage, dcCurrent, boardTemperature *string
		sfpState, port0State, port1State, port2State, port3State                *string
		averageSOCValues                                                        *string
		upTimeSeconds                                                           *int64
		averageSOC, packCounts                                                  *int
		lastInformAt                                                            *time.Time
	)
	const runtimeQuery = `SELECT
		external_ip, total_voltage, total_temperature, total_current,
		software_version, hardware_version, manufacturer, manufacturer_oui,
		run_time, bms_charging,
		ac_power, ac_voltage, dc_voltage, dc_current, board_temperature,
		sfp_state, port0_state, port1_state, port2_state, port3_state,
		average_soc, average_soc_values, pack_counts, last_inform_at
	FROM device_ups_info
	WHERE device_id = $1`
	err := r.pool.QueryRow(ctx, runtimeQuery, device.ID).Scan(
		&externalIP, &totalVoltage, &totalTemperature, &totalCurrent,
		&softwareVersion, &hardwareVersion, &manufacturer, &manufacturerOUI,
		&upTimeSeconds, &bmsCharging,
		&acPower, &acVoltage, &dcVoltage, &dcCurrent, &boardTemperature,
		&sfpState, &port0State, &port1State, &port2State, &port3State,
		&averageSOC, &averageSOCValues, &packCounts, &lastInformAt,
	)
	if err == pgx.ErrNoRows {
		return detail, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get UPS runtime detail: %w", err)
	}

	detail.PowerSystem.Manufacturer = firstStringPtr(manufacturer, detail.PowerSystem.Manufacturer)
	detail.PowerSystem.ManufacturerOUI = firstStringPtr(manufacturerOUI, detail.PowerSystem.ManufacturerOUI)
	detail.PowerSystem.HardwareVersion = hardwareVersion
	detail.PowerSystem.SoftwareVersion = firstStringPtr(softwareVersion, detail.PowerSystem.SoftwareVersion)
	detail.PowerSystem.UpTimeSeconds = upTimeSeconds
	detail.PowerRun = UPSPowerRunParam{
		ExternalIP:       externalIP,
		TotalVoltage:     totalVoltage,
		TotalTemperature: totalTemperature,
		TotalCurrent:     totalCurrent,
		BMSCharging:      bmsCharging,
		ACPower:          acPower,
		ACVoltage:        acVoltage,
		DCVoltage:        dcVoltage,
		DCCurrent:        dcCurrent,
		BoardTemperature: boardTemperature,
		SFPState:         sfpState,
		Port0State:       port0State,
		Port1State:       port1State,
		Port2State:       port2State,
		Port3State:       port3State,
		AverageSOC:       averageSOC,
		AverageSOCValues: averageSOCValues,
		PackCounts:       packCounts,
		LastInformAt:     lastInformAt,
	}

	rows, err := r.pool.Query(ctx, `SELECT
		pack_index, serial_number, soc, soc_values, voltage, temperature, current_value, status,
		recycle_count, charging, model, software_version
	FROM device_ups_batteries
	WHERE device_id = $1
	ORDER BY pack_index ASC`, device.ID)
	if err != nil {
		return nil, fmt.Errorf("list UPS batteries: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var item UPSBatteryRuntimeParam
		if err := rows.Scan(
			&item.PackIndex,
			&item.SerialNumber,
			&item.SOC,
			&item.SOCValues,
			&item.Voltage,
			&item.Temperature,
			&item.Current,
			&item.Status,
			&item.RecycleCount,
			&item.Charging,
			&item.Model,
			&item.SoftwareVersion,
		); err != nil {
			return nil, fmt.Errorf("scan UPS battery detail: %w", err)
		}
		detail.BatteryRun = append(detail.BatteryRun, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate UPS batteries: %w", err)
	}
	return detail, nil
}

func (r *PgUPSRuntimeRepository) UpsertFromInform(ctx context.Context, device *model.Device, inform *tr069.InformMessage) error {
	if r == nil || r.pool == nil {
		return nil
	}
	runtime, batteries, ok := buildUPSRuntimeProjection(device, inform)
	if !ok {
		return nil
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin UPS runtime transaction: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx)
		}
	}()

	const upsertUPSInfoRuntime = `INSERT INTO device_ups_info (
		device_id, device_serial_number, external_ip,
		total_voltage, total_temperature, total_current,
		software_version, hardware_version, manufacturer, manufacturer_oui,
		run_time, bms_charging,
		ac_power, ac_voltage, dc_voltage, dc_current, board_temperature,
		sfp_state, port0_state, port1_state, port2_state, port3_state,
		average_soc, average_soc_values, pack_counts, last_inform_at,
		first_online_time, last_online_time, created_at, updated_at
	) VALUES (
		$1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
		$11, $12, $13, $14, $15, $16, $17, $18, $19, $20,
		$21, $22, $23, $24, $25, $26, $26, $26, NOW(), NOW()
	)
	ON CONFLICT (device_id) DO UPDATE SET
		device_serial_number = EXCLUDED.device_serial_number,
		external_ip = EXCLUDED.external_ip,
		total_voltage = EXCLUDED.total_voltage,
		total_temperature = EXCLUDED.total_temperature,
		total_current = EXCLUDED.total_current,
		software_version = EXCLUDED.software_version,
		hardware_version = EXCLUDED.hardware_version,
		manufacturer = EXCLUDED.manufacturer,
		manufacturer_oui = EXCLUDED.manufacturer_oui,
		run_time = COALESCE(EXCLUDED.run_time, device_ups_info.run_time),
		bms_charging = EXCLUDED.bms_charging,
		ac_power = EXCLUDED.ac_power,
		ac_voltage = EXCLUDED.ac_voltage,
		dc_voltage = EXCLUDED.dc_voltage,
		dc_current = EXCLUDED.dc_current,
		board_temperature = EXCLUDED.board_temperature,
		sfp_state = EXCLUDED.sfp_state,
		port0_state = EXCLUDED.port0_state,
		port1_state = EXCLUDED.port1_state,
		port2_state = EXCLUDED.port2_state,
		port3_state = EXCLUDED.port3_state,
		average_soc = EXCLUDED.average_soc,
		average_soc_values = EXCLUDED.average_soc_values,
		pack_counts = EXCLUDED.pack_counts,
		last_inform_at = COALESCE(EXCLUDED.last_inform_at, device_ups_info.last_inform_at),
		first_online_time = COALESCE(device_ups_info.first_online_time, EXCLUDED.first_online_time),
		last_online_time = COALESCE(EXCLUDED.last_online_time, device_ups_info.last_online_time),
		updated_at = NOW()`

	if _, err := tx.Exec(ctx, upsertUPSInfoRuntime,
		runtime.DeviceID,
		runtime.DeviceSerialNumber,
		blankToNil(runtime.ExternalIP),
		blankToNil(runtime.TotalVoltage),
		blankToNil(runtime.TotalTemperature),
		blankToNil(runtime.TotalCurrent),
		blankToNil(runtime.SoftwareVersion),
		blankToNil(runtime.HardwareVersion),
		blankToNil(runtime.Manufacturer),
		blankToNil(runtime.ManufacturerOUI),
		optionalInt64(runtime.UpTimeSeconds),
		blankToNil(runtime.BMSCharging),
		blankToNil(runtime.ACPower),
		blankToNil(runtime.ACVoltage),
		blankToNil(runtime.DCVoltage),
		blankToNil(runtime.DCCurrent),
		blankToNil(runtime.BoardTemperature),
		blankToNil(runtime.SFPState),
		blankToNil(runtime.Port0State),
		blankToNil(runtime.Port1State),
		blankToNil(runtime.Port2State),
		blankToNil(runtime.Port3State),
		optionalInt(runtime.AverageSOC),
		blankToNil(runtime.AverageSOCValues),
		optionalInt(runtime.PackCounts),
		runtime.LastInformAt,
	); err != nil {
		return fmt.Errorf("upsert UPS info runtime fields: %w", err)
	}

	if _, err := tx.Exec(ctx, "DELETE FROM device_ups_batteries WHERE device_id = $1", runtime.DeviceID); err != nil {
		return fmt.Errorf("clear UPS batteries: %w", err)
	}

	const insertBattery = `INSERT INTO device_ups_batteries (
		device_id, device_serial_number, pack_index, serial_number,
		soc, soc_values, voltage, temperature, current_value, status,
		recycle_count, charging, model, software_version
	) VALUES (
		$1, $2, $3, $4, $5, $6, $7,
		$8, $9, $10, $11, $12, $13, $14
	)`
	for _, battery := range batteries {
		if _, err := tx.Exec(ctx, insertBattery,
			battery.DeviceID,
			battery.DeviceSerialNumber,
			battery.PackIndex,
			optionalString(battery.SerialNumber),
			optionalInt(battery.SOC),
			blankToNil(battery.SOCValues),
			blankToNil(battery.Voltage),
			blankToNil(battery.Temperature),
			blankToNil(battery.Current),
			blankToNil(battery.Status),
			optionalInt(battery.RecycleCount),
			blankToNil(battery.Charging),
			blankToNil(battery.Model),
			blankToNil(battery.SoftwareVersion),
		); err != nil {
			return fmt.Errorf("insert UPS battery pack %d: %w", battery.PackIndex, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit UPS runtime transaction: %w", err)
	}
	committed = true
	return nil
}

type UPSRuntimeProjection struct {
	DeviceID           uuid.UUID
	DeviceSerialNumber string
	ExternalIP         string
	TotalVoltage       string
	TotalTemperature   string
	TotalCurrent       string
	SoftwareVersion    string
	HardwareVersion    string
	Manufacturer       string
	ManufacturerOUI    string
	UpTimeSeconds      *int64
	BMSCharging        string
	ACPower            string
	ACVoltage          string
	DCVoltage          string
	DCCurrent          string
	BoardTemperature   string
	SFPState           string
	Port0State         string
	Port1State         string
	Port2State         string
	Port3State         string
	AverageSOC         *int
	AverageSOCValues   string
	PackCounts         *int
	LastInformAt       *time.Time
}

type UPSBatteryProjection struct {
	DeviceID           uuid.UUID
	DeviceSerialNumber string
	PackIndex          int
	SerialNumber       *string
	SOC                *int
	SOCValues          string
	Voltage            string
	Temperature        string
	Current            string
	Status             string
	RecycleCount       *int
	Charging           string
	Model              string
	SoftwareVersion    string
}

func buildUPSRuntimeProjection(device *model.Device, inform *tr069.InformMessage) (*UPSRuntimeProjection, []UPSBatteryProjection, bool) {
	if device == nil || inform == nil || !isUPSProductClass(inform.DeviceId.ProductClass) {
		return nil, nil, false
	}
	values := parameterValuesByPath(inform.ParameterList)
	value := func(path string) string { return values[path] }
	firstValue := func(paths ...string) string {
		for _, path := range paths {
			if v := strings.TrimSpace(values[path]); v != "" {
				return v
			}
		}
		return ""
	}

	runtime := &UPSRuntimeProjection{
		DeviceID:           device.ID,
		DeviceSerialNumber: device.SerialNumber,
		ExternalIP:         firstNonBlank(informUPSExternalIPAddress(inform.ParameterList), device.IPAddress),
		TotalVoltage:       firstValue("InternetGatewayDevice.BMSInfo.Voltage", "InternetGatewayDevice.BMSInfo.1.Voltage"),
		TotalTemperature:   firstValue("InternetGatewayDevice.BMSInfo.Temperature", "InternetGatewayDevice.BMSInfo.1.Temperature"),
		TotalCurrent:       firstValue("InternetGatewayDevice.BMSInfo.Current", "InternetGatewayDevice.BMSInfo.1.Current"),
		SoftwareVersion:    firstNonBlank(value(igdSoftwareVersionPath), device.FirmwareVersion),
		HardwareVersion:    value("InternetGatewayDevice.DeviceInfo.HardwareVersion"),
		Manufacturer:       firstNonBlank(value("InternetGatewayDevice.DeviceInfo.Manufacturer"), inform.DeviceId.Manufacturer, device.Manufacturer),
		ManufacturerOUI:    firstNonBlank(value("InternetGatewayDevice.DeviceInfo.ManufacturerOUI"), inform.DeviceId.OUI, device.OUI),
		UpTimeSeconds:      parseOptionalInt64(value("InternetGatewayDevice.DeviceInfo.UpTime")),
		BMSCharging:        firstValue("InternetGatewayDevice.BMSInfo.Charging", "InternetGatewayDevice.BMSInfo.1.Charging"),
		ACPower:            value("InternetGatewayDevice.ChargerInfo.ACPower"),
		ACVoltage:          value("InternetGatewayDevice.ChargerInfo.ACVoltage"),
		DCVoltage:          value("InternetGatewayDevice.ChargerInfo.DCVoltage"),
		DCCurrent:          value("InternetGatewayDevice.ChargerInfo.DCCurrent"),
		BoardTemperature:   value("InternetGatewayDevice.ChargerInfo.BoardTemperature"),
		SFPState:           value("InternetGatewayDevice.ChargerInfo.SFPState"),
		Port0State:         firstValue("InternetGatewayDevice.ChargerInfo.PORT0State", "InternetGatewayDevice.ChargerInfo.Port0State"),
		Port1State:         firstValue("InternetGatewayDevice.ChargerInfo.PORT1State", "InternetGatewayDevice.ChargerInfo.Port1State"),
		Port2State:         firstValue("InternetGatewayDevice.ChargerInfo.PORT2State", "InternetGatewayDevice.ChargerInfo.Port2State"),
		Port3State:         firstValue("InternetGatewayDevice.ChargerInfo.PORT3State", "InternetGatewayDevice.ChargerInfo.Port3State"),
		AverageSOC:         parseOptionalInt(value("InternetGatewayDevice.ChargerInfo.AverageSOC")),
		AverageSOCValues:   value("InternetGatewayDevice.ChargerInfo.AverageSOC"),
		PackCounts:         parseOptionalInt(value("InternetGatewayDevice.ChargerInfo.PackCounts")),
		LastInformAt:       device.LastInformAt,
	}

	batteries := buildUPSBatteryProjections(runtime, values)
	return runtime, batteries, true
}

func buildUPSBatteryProjections(runtime *UPSRuntimeProjection, values map[string]string) []UPSBatteryProjection {
	if runtime == nil {
		return nil
	}
	if runtime.PackCounts != nil {
		count := *runtime.PackCounts
		if count <= 0 {
			return nil
		}
		batteries := make([]UPSBatteryProjection, 0, count)
		for index := 1; index <= count; index++ {
			prefix := fmt.Sprintf("InternetGatewayDevice.BMSInfo.%d.", index)
			if !hasIndexedUPSBattery(values, prefix) {
				continue
			}
			batteries = append(batteries, UPSBatteryProjection{
				DeviceID:           runtime.DeviceID,
				DeviceSerialNumber: runtime.DeviceSerialNumber,
				PackIndex:          index,
				SerialNumber:       legacyUPSBatterySerial(values[prefix+"SerialNumber"]),
				SOC:                parseOptionalInt(values[prefix+"SOC"]),
				SOCValues:          values[prefix+"SOC"],
				Voltage:            values[prefix+"Voltage"],
				Temperature:        values[prefix+"Temperature"],
				Current:            values[prefix+"Current"],
				Status:             values[prefix+"Status"],
				RecycleCount:       parseOptionalInt(values[prefix+"RecycleCount"]),
				Charging:           values[prefix+"Charging"],
				Model:              legacyUPSBatteryPlaceholder(values[prefix+"Model"]),
				SoftwareVersion:    legacyUPSBatteryPlaceholder(values[prefix+"SoftwareVersion"]),
			})
		}
		return batteries
	}

	if !hasLegacyUPSBattery(values) {
		return nil
	}
	return []UPSBatteryProjection{{
		DeviceID:           runtime.DeviceID,
		DeviceSerialNumber: runtime.DeviceSerialNumber,
		PackIndex:          1,
		SerialNumber:       legacyUPSBatterySerial(values["InternetGatewayDevice.BMSInfo.SerialNumber"]),
		SOC:                parseOptionalInt(values["InternetGatewayDevice.BMSInfo.SOC"]),
		SOCValues:          values["InternetGatewayDevice.BMSInfo.SOC"],
		Voltage:            firstNonBlank(values["InternetGatewayDevice.BMSInfo.Voltage"], runtime.TotalVoltage),
		Temperature:        firstNonBlank(values["InternetGatewayDevice.BMSInfo.Temperature"], runtime.TotalTemperature),
		Current:            firstNonBlank(values["InternetGatewayDevice.BMSInfo.Current"], runtime.TotalCurrent),
		Status:             values["InternetGatewayDevice.BMSInfo.Status"],
		RecycleCount:       parseOptionalInt(values["InternetGatewayDevice.BMSInfo.RecycleCount"]),
		Charging:           firstNonBlank(values["InternetGatewayDevice.BMSInfo.Charging"], runtime.BMSCharging),
		Model:              legacyUPSBatteryPlaceholder(values["InternetGatewayDevice.BMSInfo.Model"]),
		SoftwareVersion:    legacyUPSBatteryPlaceholder(values["InternetGatewayDevice.BMSInfo.SoftwareVersion"]),
	}}
}

func hasIndexedUPSBattery(values map[string]string, prefix string) bool {
	for _, suffix := range []string{
		"SerialNumber",
		"SOC",
		"SOH",
		"Voltage",
		"Temperature",
		"Current",
		"Status",
		"RecycleCount",
		"Charging",
		"Model",
		"SoftwareVersion",
	} {
		if strings.TrimSpace(values[prefix+suffix]) != "" {
			return true
		}
	}
	return false
}

func parameterValuesByPath(params []tr069.ParameterValueStruct) map[string]string {
	values := make(map[string]string, len(params))
	for _, param := range params {
		values[param.Name] = strings.TrimSpace(param.Value)
	}
	return values
}

func hasLegacyUPSBattery(values map[string]string) bool {
	for _, path := range []string{
		"InternetGatewayDevice.BMSInfo.Voltage",
		"InternetGatewayDevice.BMSInfo.Temperature",
		"InternetGatewayDevice.BMSInfo.Current",
		"InternetGatewayDevice.BMSInfo.Charging",
		"InternetGatewayDevice.BMSInfo.SOC",
	} {
		if strings.TrimSpace(values[path]) != "" {
			return true
		}
	}
	return false
}

func validUPSBatterySerialNumber(serialNumber string) bool {
	normalized := strings.TrimSpace(serialNumber)
	return normalized != "" && !strings.EqualFold(normalized, "N/A")
}

func legacyUPSBatterySerial(serialNumber string) *string {
	normalized := strings.TrimSpace(serialNumber)
	if !validUPSBatterySerialNumber(normalized) {
		return stringPtr("--")
	}
	return &normalized
}

func legacyUPSBatteryPlaceholder(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "--"
	}
	return trimmed
}

func parseOptionalInt(value string) *int {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	if strings.Contains(trimmed, ",") {
		trimmed = strings.TrimSpace(strings.Split(trimmed, ",")[0])
		if trimmed == "" {
			return nil
		}
	}
	parsed, err := strconv.Atoi(trimmed)
	if err != nil {
		return nil
	}
	return &parsed
}

func parseOptionalInt64(value string) *int64 {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	if strings.Contains(trimmed, ",") {
		trimmed = strings.TrimSpace(strings.Split(trimmed, ",")[0])
		if trimmed == "" {
			return nil
		}
	}
	if dot := strings.Index(trimmed, "."); dot >= 0 {
		trimmed = strings.TrimSpace(trimmed[:dot])
		if trimmed == "" {
			return nil
		}
	}
	parsed, err := strconv.ParseInt(trimmed, 10, 64)
	if err != nil {
		return nil
	}
	return &parsed
}

func optionalInt(value *int) interface{} {
	if value == nil {
		return nil
	}
	return *value
}

func optionalInt64(value *int64) interface{} {
	if value == nil {
		return nil
	}
	return *value
}

func optionalString(value *string) interface{} {
	if value == nil || strings.TrimSpace(*value) == "" {
		return nil
	}
	return *value
}

func blankToNil(value string) interface{} {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	return trimmed
}

func firstNonBlank(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func stringPtr(value string) *string {
	trimmed := strings.TrimSpace(value)
	return &trimmed
}

func stringPtrIfNotBlank(value string) *string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func firstStringPtr(values ...*string) *string {
	for _, value := range values {
		if value == nil {
			continue
		}
		trimmed := strings.TrimSpace(*value)
		if trimmed != "" {
			return &trimmed
		}
	}
	return nil
}
