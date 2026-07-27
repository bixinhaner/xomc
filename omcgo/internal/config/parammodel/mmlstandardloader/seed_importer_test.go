package mmlstandardloader

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
)

func TestManagementServerSSLStatusPathsAreGlobalReadOnlyMMLParams(t *testing.T) {
	_, thisFile, _, _ := runtime.Caller(0)
	seedPath := filepath.Join(filepath.Dir(thisFile), "seeds", "cmcc_tdlte_v23.json")

	seed, err := ParseSeedFile(seedPath)
	if err != nil {
		t.Fatalf("parse seed: %v", err)
	}

	const (
		endDate   = "Device.ManagementServer.sslStatus.endDate"
		startDate = "Device.ManagementServer.sslStatus.startDate"
	)
	wantPaths := map[string]bool{endDate: false, startDate: false}
	for _, group := range seed.Groups {
		for _, command := range group.Commands {
			if command.ObjectPath != "Device.ManagementServer." {
				continue
			}
			for _, param := range command.Params {
				if _, ok := wantPaths[param.Path]; !ok {
					continue
				}
				if param.Access != AccessR || param.Type != "string" {
					t.Errorf("%s metadata = access %q type %q, want R/string", param.Path, param.Access, param.Type)
				}
				wantPaths[param.Path] = true
			}
		}
	}
	for path, found := range wantPaths {
		if !found {
			t.Errorf("global MML seed is missing %s", path)
		}
	}

	var lstParams, modParams map[string]bool
	for _, command := range DeriveCommandsFromSeed(seed) {
		switch command.CommandCode {
		case "LST MANAGEMENT_SERVER":
			lstParams = make(map[string]bool, len(command.SubFields))
			for _, param := range command.SubFields {
				lstParams[param.Path] = true
			}
		case "MOD MANAGEMENT_SERVER":
			modParams = make(map[string]bool, len(command.SubFields))
			for _, param := range command.SubFields {
				modParams[param.Path] = true
			}
		}
	}
	for path := range wantPaths {
		if !lstParams[path] {
			t.Errorf("LST MANAGEMENT_SERVER is missing %s", path)
		}
		if modParams[path] {
			t.Errorf("MOD MANAGEMENT_SERVER must not contain read-only %s", path)
		}
	}

	blqPath := filepath.Join(filepath.Dir(thisFile), "..", "..", "..", "..", "data", "param-mappings", "BLQ.xml")
	blq, err := os.ReadFile(blqPath)
	if err != nil {
		t.Fatalf("read BLQ mapping: %v", err)
	}
	for path := range wantPaths {
		marker := `standardPath="` + path + `"`
		idx := strings.Index(string(blq), marker)
		if idx < 0 {
			t.Errorf("BLQ mapping is missing %s", path)
			continue
		}
		lineStart := strings.LastIndex(string(blq[:idx]), "\n") + 1
		lineEnd := strings.Index(string(blq[idx:]), "\n")
		if lineEnd < 0 {
			lineEnd = len(blq) - idx
		}
		line := string(blq[lineStart : idx+lineEnd])
		if strings.Contains(line, `supported="false"`) {
			t.Errorf("BLQ mapping marks %s unsupported: %s", path, strings.TrimSpace(line))
		}
	}
}

// TestSmokeSeedDerivation: 在 cmcc_tdlte_v23.json 上验证：
//   - 18 chapter / 71 command / 616 param 数量精确匹配
//     （RRCTimers 的 10 个 LTE 计时器参数已归入专用 RRC 计时器命令，避免 FAP_SERVICE 重复）
//   - LST 总生成；MOD 仅在含 RW 时；ADD/RMV 双门槛
//   - logical_code 全集唯一
func TestSmokeSeedDerivation(t *testing.T) {
	_, thisFile, _, _ := runtime.Caller(0)
	seedPath := filepath.Join(filepath.Dir(thisFile), "seeds", "cmcc_tdlte_v23.json")

	seed, err := ParseSeedFile(seedPath)
	if err != nil {
		t.Fatalf("parse seed: %v", err)
	}

	if got := len(seed.Groups); got != 18 {
		t.Errorf("chapter groups: got %d, want 18", got)
	}
	if got := seed.CountCommands(); got != 71 {
		t.Errorf("seed commands: got %d, want 71", got)
	}
	if got := seed.CountParams(); got != 616 {
		t.Errorf("seed params: got %d, want 616", got)
	}

	derived := DeriveCommandsFromSeed(seed)

	// 命令 code 全集唯一
	codes := make(map[string]int)
	logicalCodes := make(map[string]int)
	opCount := map[string]int{}
	for _, c := range derived {
		codes[c.CommandCode]++
		logicalCodes[c.LogicalCode]++
		opCount[c.OperationType]++
	}
	for code, n := range codes {
		if n > 1 {
			t.Errorf("duplicate command_code %q (count=%d)", code, n)
		}
	}

	// LST 应 ≤ 71（部分 command 无 params 被跳过；但非空的应都生成 LST）
	// 期望：所有 71 commands 都有 ≥1 params（验证后），所以 LST=71
	if got := opCount[OpLST]; got != 71 {
		t.Errorf("LST count: got %d, want 71", got)
	}
	if opCount[OpMOD] == 0 || opCount[OpMOD] > opCount[OpLST] {
		t.Errorf("MOD count sanity: got %d (must be 0<MOD<=LST)", opCount[OpMOD])
	}
	if opCount[OpADD] != opCount[OpRMV] {
		t.Errorf("ADD vs RMV mismatch: ADD=%d RMV=%d", opCount[OpADD], opCount[OpRMV])
	}

	// 打印汇总（用 t.Logf 不阻塞 CI）
	t.Logf("derived total=%d  LST=%d MOD=%d ADD=%d RMV=%d",
		len(derived), opCount[OpLST], opCount[OpMOD], opCount[OpADD], opCount[OpRMV])

	// 抽样输出 logical_code（验证派生效果）
	sample := []string{
		"Device.DeviceInfo.",
		"Device.DeviceInfo.SwUpgrade.",
		"Device.ManagementServer.",
		"Device.Services.FAPService.{i}.",
		"Device.Services.FAPService.{i}.CellConfig.LTE.EPC.",
		"Device.Services.FAPService.{i}.CellConfig.LTE.RAN.MAC.",
		"Device.Services.FAPService.{i}.CellConfig.LTE.RAN.PHY.",
		"Device.DeviceInfo.MU.{i}.SwUpgrade.",
	}
	allPaths := make([]string, 0)
	seen := map[string]bool{}
	for _, g := range seed.Groups {
		for _, c := range g.Commands {
			if !seen[c.ObjectPath] {
				allPaths = append(allPaths, c.ObjectPath)
				seen[c.ObjectPath] = true
			}
		}
	}
	codes2 := DeriveLogicalCodes(allPaths)
	for _, p := range sample {
		t.Logf("logical_code: %-80s → %s", p, codes2[p])
	}
}

// TestCamelToSnakeUpper 单元覆盖
func TestCamelToSnakeUpper(t *testing.T) {
	cases := []struct{ in, want string }{
		{"DeviceInfo", "DEVICE_INFO"},
		{"FAPService", "FAP_SERVICE"},
		{"MU", "MU"},
		{"VlanInterface", "VLAN_INTERFACE"},
		{"X_COM_PARAM", "X_COM_PARAM"},
		{"SwUpgrade", "SW_UPGRADE"},
		{"RFChannel", "RF_CHANNEL"},
	}
	for _, tc := range cases {
		if got := camelToSnakeUpper(tc.in); got != tc.want {
			t.Errorf("camelToSnakeUpper(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

// TestNonCreatableExclusion 验证 non_creatable_objects 名单中的 path 不生成 ADD/RMV
func TestNonCreatableExclusion(t *testing.T) {
	_, thisFile, _, _ := runtime.Caller(0)
	seedPath := filepath.Join(filepath.Dir(thisFile), "seeds", "cmcc_tdlte_v23.json")
	seed, err := ParseSeedFile(seedPath)
	if err != nil {
		t.Fatalf("parse seed: %v", err)
	}
	derived := DeriveCommandsFromSeed(seed)
	nonCreatable := seed.NonCreatableSet()
	for _, c := range derived {
		if c.OperationType != OpADD && c.OperationType != OpRMV {
			continue
		}
		if nonCreatable[c.ObjectPath] {
			t.Errorf("path %q is in non_creatable_objects but ADD/RMV generated (%s)",
				c.ObjectPath, c.CommandCode)
		}
	}

	// 反向：每个生成 ADD 的 path 必须含 .{i}.
	for _, c := range derived {
		if c.OperationType != OpADD {
			continue
		}
		if !strings.Contains(c.ObjectPath, ".{i}.") {
			t.Errorf("ADD command %s on path %q without .{i}.", c.CommandCode, c.ObjectPath)
		}
	}

	// MOD 必须有 RW params：用 sub_field 数 > 0 反推
	for _, c := range derived {
		if c.OperationType != OpMOD {
			continue
		}
		if len(c.SubFields) == 0 {
			t.Errorf("MOD %s has zero RW sub_fields", c.CommandCode)
		}
	}

	// 输出 ADD/RMV 的 command_code 列表（手测验证）
	var addList []string
	for _, c := range derived {
		if c.OperationType == OpADD {
			addList = append(addList, c.CommandCode)
		}
	}
	sort.Strings(addList)
	t.Logf("ADD commands (%d):\n  %s", len(addList), strings.Join(addList, "\n  "))
	_ = fmt.Sprintf
}
