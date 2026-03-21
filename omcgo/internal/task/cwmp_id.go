package task

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"
	"time"
)

// CWMP ID 格式: ID:intrnl.unset.id.{Method}{timestamp}.{random}
// 示例: ID:intrnl.unset.id.GetParameterValues1772248515705.45470049

const (
	cwmpIDPrefix = "ID:intrnl.unset.id."
)

// GenerateCWMPID 生成符合 TR069 规范的 CWMP ID
// 格式: ID:intrnl.unset.id.{Method}{timestamp}.{random}
// 示例: ID:intrnl.unset.id.GetParameterValues1772248515705.45470049
func GenerateCWMPID(method string) string {
	timestamp := time.Now().Unix()
	random := randomInt64(100000000) // 8位随机数
	return fmt.Sprintf("%s%s%d.%08d", cwmpIDPrefix, method, timestamp, random)
}

// ParseCWMPID 解析 CWMP ID，提取方法名和时间戳
// 返回: method, timestamp, ok
func ParseCWMPID(cwmpID string) (method string, timestamp int64, ok bool) {
	if !strings.HasPrefix(cwmpID, cwmpIDPrefix) {
		return "", 0, false
	}

	// 移除前缀
	rest := strings.TrimPrefix(cwmpID, cwmpIDPrefix)

	// 查找最后一个点号，分隔时间戳和随机数
	lastDot := strings.LastIndex(rest, ".")
	if lastDot == -1 {
		return "", 0, false
	}

	// 在最后一个点号之前查找方法名和时间戳的分隔点
	// 格式: {Method}{timestamp}.{random}
	// 方法名是字母，时间戳是数字
	methodAndTs := rest[:lastDot]

	// 从后往前找到第一个数字的位置，那就是方法名和时间戳的分隔点
	splitIdx := -1
	for i := len(methodAndTs) - 1; i >= 0; i-- {
		if methodAndTs[i] >= '0' && methodAndTs[i] <= '9' {
			splitIdx = i
		} else {
			break
		}
	}

	if splitIdx == -1 || splitIdx == 0 {
		return "", 0, false
	}

	method = methodAndTs[:splitIdx]
	timestampStr := methodAndTs[splitIdx:]

	// 解析时间戳
	ts, err := parseInt64(timestampStr)
	if err != nil {
		return "", 0, false
	}

	return method, ts, true
}

// generateUUID 生成 UUID v4
func generateUUID() string {
	uuidBytes := make([]byte, 16)
	rand.Read(uuidBytes)
	// Set version (4) and variant bits per RFC 4122
	uuidBytes[6] = (uuidBytes[6] & 0x0f) | 0x40
	uuidBytes[8] = (uuidBytes[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		uuidBytes[0:4], uuidBytes[4:6], uuidBytes[6:8], uuidBytes[8:10], uuidBytes[10:16])
}

// randomInt64 生成 [0, max) 范围内的随机整数
func randomInt64(max int64) int64 {
	if max <= 0 {
		return 0
	}
	n, err := rand.Int(rand.Reader, big.NewInt(max))
	if err != nil {
		// Fallback to timestamp-based random
		return time.Now().UnixNano() % max
	}
	return n.Int64()
}

// parseInt64 解析字符串为 int64
func parseInt64(s string) (int64, error) {
	var result int64
	for _, c := range s {
		if c < '0' || c > '9' {
			break
		}
		result = result*10 + int64(c-'0')
	}
	return result, nil
}

// CWMPIDHash 生成 CWMP ID 的短哈希，用于 Redis key
func CWMPIDHash(cwmpID string) string {
	if len(cwmpID) <= 32 {
		return cwmpID
	}
	// 使用前 32 个字符作为哈希
	hash := make([]byte, 16)
	for i, c := range cwmpID {
		if i >= 32 {
			break
		}
		hash[i%16] ^= byte(c)
	}
	return hex.EncodeToString(hash)
}
