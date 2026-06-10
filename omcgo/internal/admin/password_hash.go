package admin

import (
	"os"
	"strconv"
	"sync"

	"golang.org/x/crypto/bcrypt"
)

// 口令 / API Key 哈希的 bcrypt cost 策略（issue #6 安全加固）。
//
// 背景：bcrypt.DefaultCost 当前仅为 10，面向运营商商用网管、考虑 5 年抗暴破，
// 需要把工作因子抬到 ≥12。本项目默认 13，并允许运维按硬件能力经环境变量微调。
//
// 取值规则（在 [bcryptCostFloor, bcrypt.MaxCost] 间夹取）：
//   - 默认 bcryptCostDefault = 13；
//   - 环境变量 OMC_BCRYPT_COST 可覆盖，但永不低于 bcryptCostFloor = 12
//     （防止误配把安全强度降到不可接受的水平），也不高于 bcrypt.MaxCost。
const (
	// bcryptCostFloor 是允许的最低 cost；低于此值的配置一律抬到此值，
	// 以保证即使误配也不会降低到弱于历史基线的安全强度。
	bcryptCostFloor = 12
	// bcryptCostDefault 是未显式配置时采用的 cost。
	bcryptCostDefault = 13
	// bcryptCostEnv 是覆盖默认 cost 的环境变量名。
	bcryptCostEnv = "OMC_BCRYPT_COST"
)

var (
	bcryptCostOnce  sync.Once
	bcryptCostValue int
)

// resolveBcryptCost 解析一次进程级 bcrypt cost（带夹取与环境变量覆盖），
// 之后缓存复用，避免每次哈希都读环境变量。
func resolveBcryptCost() int {
	bcryptCostOnce.Do(func() {
		cost := bcryptCostDefault
		if raw := os.Getenv(bcryptCostEnv); raw != "" {
			if parsed, err := strconv.Atoi(raw); err == nil {
				cost = parsed
			}
		}
		bcryptCostValue = clampBcryptCost(cost)
	})
	return bcryptCostValue
}

// clampBcryptCost 把任意输入夹取到 [bcryptCostFloor, bcrypt.MaxCost]。
func clampBcryptCost(cost int) int {
	if cost < bcryptCostFloor {
		return bcryptCostFloor
	}
	if cost > bcrypt.MaxCost {
		return bcrypt.MaxCost
	}
	return cost
}

// hashSecret 用进程级 bcrypt cost 生成口令 / API Key 哈希。
// 所有口令/密钥的哈希点都应走此函数，确保 cost 策略单一来源。
func hashSecret(plaintext []byte) ([]byte, error) {
	return bcrypt.GenerateFromPassword(plaintext, resolveBcryptCost())
}
