# PRD: 后端 backup policy endpoint + 前端 BackupPolicy 接入（T-0071 / R-102 followup）

> **关联**: Backlog T-0071 / Sprint-06 / Domain=F06/backup + frontend / Type=feat
> **作者**: Claude（代 Owner=电信+前端）
> **创建**: 2026-04-29
> **状态**: 草案 → 实施
> **关键决策**: **持久化 MVP**（schema 容纳 19 字段，executor 端目前不执行；enforcement 拆 4-5 个 follow-up）

---

## 1. 业务背景

T-0016 audit：BackupPolicy 305 行纯本地 form（19 字段 / 7 类）+ `console.log` 假保存（CLAUDE.md 禁止 production console.log），后端无 `/backup/policies` endpoint，无 `backup_policies` 表。

T-0070 已闭环 BackupSchedule；R-102 Backup 子模块剩 Policy + Restore 两块。**Policy 是存与取的问题，Restore 是流程设计的问题**。本任务先解决 Policy。

---

## 2. ULTRATHINK 决策：MVP 范围

### 2.1 19 字段 × 7 类盘点

| 类别 | 字段 | enforcement 复杂度 | MVP 处理 |
|------|------|-------------------|---------|
| **保留策略** | retentionDays / maxBackupCount / minBackupCount | retentionDays 可由 cleanup 任务读取；max/min 需 list-after-create 校验 | **存** |
| **自动清理** | autoCleanup / cleanupTime / cleanupDayOfWeek / keepLastN | 需新建 cron 任务（cleanup-job） | **存**；enforcement = 后续 T-0073 |
| **压缩** | enableCompression / compressionLevel / compressionFormat | 需 backup executor 集成 compress lib（gzip/bzip2/lz4/zstd） | **存**；enforcement = 后续 T-0074 |
| **存储** | storageBackend / ftpConfigId / localPath / maxStorageGB | local/ftp executor 已基本 OK；maxStorageGB 需磁盘监控 | **存** + storageBackend / ftpConfigId 已存（FK）；磁盘监控 = T-0073（合并） |
| **加密** | enableEncryption / encryptionAlgorithm | 需 executor + KMS / 本地密钥管理（**安全敏感**） | **存**；enforcement = 后续 T-0075 |
| **告警** | alertOnFailure / alertEmail / alertThresholdPercent | F04 邮件通道（T-0007）已就绪；只需 backup executor publish event | **存** + 集成 = 后续 T-0073 |
| **其他** | （无） | | |

### 2.2 MVP 决策

**持久化 MVP**：本 PRD 交付的是"配置持久化层"——schema、CRUD、前后端连通，**不交付 executor enforcement**。

- ✅ 用户在 UI 改 19 个字段，PUT /backup/policy 成功，GET 取回原值（行为正确）
- ✅ 单实例（singleton）行 — 一份全局策略；UI 改的就是它
- ❌ **不交付**：cleanup cron / 压缩执行 / 加密执行 / 磁盘监控 / 告警发送（每项都是独立 executor 工作）
- ✅ UI 在 cleanup / 压缩 / 加密 / 告警 4 段加 `Tag` "尚未生效" 标签 + tooltip "本字段持久化但 executor 待 followup PR 接入"

**为什么不直接拆 4-5 个 followup 不做这个**：因为 schema 是 enforcement 的基础；先建表 + CRUD 后，每个 followup 只需做 executor 端工作，schema/UI 变更小。

### 2.3 后续 followup（本任务结束后立 backlog）

| ID（待登记） | 内容 | Est | 依赖 |
|------------|------|-----|------|
| T-0073 | backup cleanup cron + 告警事件发布（autoCleanup + retentionDays + alertOnFailure 三字段实际生效）| L | T-0071 ✅, T-0007 ✅ |
| T-0074 | backup executor 压缩集成（gzip/bzip2/lz4/zstd）| M | T-0071 ✅ |
| T-0075 | backup 加密执行 + 密钥管理（**安全敏感**，需 SecOps 评审） | L | T-0071 ✅, KMS 设计 |

---

## 3. 用户故事

| 角色 | 故事 |
|------|------|
| 网管运维 | 我希望在"备份策略"页改的配置真的存到数据库，刷新后还在；不是 console.log 假保存 |
| 后端开发者 | 我希望有一份 `backup_policies` schema 作为后续 cleanup/compression/encryption 工作的统一配置源 |
| QA | 我希望前后端契约 e2e 可见 — GET 和 PUT /backup/policy 走真实 endpoint |
| PM | 我希望明确知道哪些字段 today **真生效**、哪些字段是"持久化先行" — UI 上要看得见区分 |

---

## 4. 验收标准（GWT）

### V1 — GET /backup/policy 返回单实例策略（无则 default）
- **Given** 数据库 `backup_policies` 表为空
- **When** 调 `GET /api/v1/backup/policy`
- **Then** 200 OK，返回 BackupPolicyResponse with default values（retentionDays=30 等，与前端 DEFAULT_VALUES 一致）

### V2 — PUT /backup/policy 创建或更新单实例
- **Given** 用户提交 19 字段 valid payload
- **When** 调 `PUT /api/v1/backup/policy`
- **Then** 200 OK；DB 单实例 upsert；GET 后 19 字段全部 round-trip 一致

### V3 — Singleton 强制（即使表里多行也只取最新一行）
- **Given** 数据库被人手动塞 N 行 `backup_policies`
- **When** GET 调用
- **Then** 取 `updated_at DESC LIMIT 1`，确保单实例语义

### V4 — Field validation
- **Given** retentionDays < 1 或 > 3650 / alertEmail 非邮箱格式 / compressionLevel < 1 或 > 9
- **When** PUT 调用
- **Then** 400 ErrInvalidInput；DB 不变

### V5 — 前端 GET on mount + PUT on save
- **Given** 用户访问 BackupPolicy 页
- **When** mount 完成
- **Then** form 用 GET 数据 setFieldsValue（不再用 hardcoded DEFAULT_VALUES）

### V6 — 前端"尚未生效" 标签
- **Given** UI 渲染 7 个 Collapse panel
- **When** 用户展开 cleanup / compression / encryption / alert 4 段
- **Then** panel header 显示 `Tag color="default"` 文字 "持久化（executor 待集成）"；tooltip 解释；保留策略 / 存储段不显示标签

### V7 — TypeScript + i18n
- 无 `any`；BackupPolicyResponse / BackupPolicyUpdateRequest 类型完整
- i18n +2 keys（"persistedNotEnforced" + tooltip 内容）

### V8 — E2E claim
- e2e_verify.sh +1 claim：`GET /backup/policy` 返回 200/401（singleton 不依赖 seed）

### V9 — 后端测试
- handler / service / repository 各 ≥3 表驱动测试用例（默认值返回 / upsert 更新 / 单实例语义 / 校验失败）

---

## 5. 运营商差异矩阵

| 维度 | CMCC | CTCC | CUCC |
|------|------|------|------|
| 策略字段 | 一致 | 一致 | 一致 |
| 默认值 | 一致 | 一致 | 一致 |
| **实际差异** | **无** | **无** | **无** |

> 备份策略属设备生命周期能力，运营商规范一致。

---

## 6. 非目标

- ❌ **任何 enforcement 实现**：cleanup cron / 压缩 / 加密 / 磁盘监控 / 告警发送 — 全部 followup（T-0073/0074/0075）
- ❌ **多策略 / per-tenant 策略**：本期 singleton 全局；未来按 tenant_id 维度扩展时另立任务
- ❌ **Policy 历史版本 / 审计日志**：本期 PUT 直接 upsert；审计日志整合 T-0063 audit pkg 时补
- ❌ **API 鉴权 / RBAC 检查**：admin 中间件复用既有，本任务不新增 RBAC 规则
- ❌ **Cron 表达式校验** for cleanupTime / cleanupDayOfWeek：直接存原值，运行时 cleanup cron 自己校验

---

## 7. 设计备忘

### 7.1 Schema（migration 000047）

```sql
-- +goose Up
CREATE TABLE IF NOT EXISTS backup_policies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    -- 保留策略
    retention_days INT NOT NULL DEFAULT 30 CHECK (retention_days BETWEEN 1 AND 3650),
    max_backup_count INT NOT NULL DEFAULT 100 CHECK (max_backup_count >= 1),
    min_backup_count INT NOT NULL DEFAULT 3 CHECK (min_backup_count >= 1),
    -- 自动清理
    auto_cleanup BOOLEAN NOT NULL DEFAULT true,
    cleanup_time VARCHAR(8) NOT NULL DEFAULT '03:00',
    cleanup_day_of_week INT NOT NULL DEFAULT -1 CHECK (cleanup_day_of_week BETWEEN -1 AND 6),
    keep_last_n INT NOT NULL DEFAULT 5 CHECK (keep_last_n >= 1),
    -- 压缩
    enable_compression BOOLEAN NOT NULL DEFAULT true,
    compression_level INT NOT NULL DEFAULT 6 CHECK (compression_level BETWEEN 1 AND 9),
    compression_format VARCHAR(16) NOT NULL DEFAULT 'gzip'
        CHECK (compression_format IN ('gzip', 'bzip2', 'lz4', 'zstd')),
    -- 存储
    storage_backend VARCHAR(16) NOT NULL DEFAULT 'local'
        CHECK (storage_backend IN ('local', 'ftp', 'sftp', 'nfs')),
    ftp_config_id UUID REFERENCES ftp_configs(id) ON DELETE SET NULL,
    local_path TEXT NOT NULL DEFAULT '/var/backup/omc',
    max_storage_gb INT NOT NULL DEFAULT 500 CHECK (max_storage_gb >= 1),
    -- 加密
    enable_encryption BOOLEAN NOT NULL DEFAULT false,
    encryption_algorithm VARCHAR(32) NOT NULL DEFAULT 'AES-256-GCM',
    -- 告警
    alert_on_failure BOOLEAN NOT NULL DEFAULT true,
    alert_email VARCHAR(256) NOT NULL DEFAULT '',
    alert_threshold_percent INT NOT NULL DEFAULT 80 CHECK (alert_threshold_percent BETWEEN 50 AND 95),
    -- 时间戳
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 单实例语义不在 DB 强制（为 future per-tenant 留余地），由 service 层 ensureSingleton 保证
-- 服务侧逻辑：GET = SELECT * ORDER BY updated_at DESC LIMIT 1；PUT = if 0 row INSERT else UPDATE the latest row

-- +goose Down
DROP TABLE IF EXISTS backup_policies;
```

### 7.2 Backend 模块布局

```
omcgo/internal/backup/
├── policy_model.go        # BackupPolicy struct + DefaultPolicy()
├── policy_repository.go   # interface PolicyRepository { Get, Upsert }
├── policy_pg_repository.go
├── policy_service.go      # service.GetPolicy / service.UpdatePolicy + validation
├── policy_handler.go      # GET /backup/policy + PUT /backup/policy
└── policy_*_test.go
```

### 7.3 Backend 路由

```go
// handler.go RegisterRoutes 末尾追加
policy := rg.Group("/backup/policy")
policy.GET("",  h.GetPolicy)
policy.PUT("",  h.UpdatePolicy)
```

### 7.4 Frontend layer 改动

**frontend-core**（单一改动点）:
- `services/api/backupApi.ts` +mapBackendPolicy + getPolicy + updatePolicy
- `hooks/api/useBackup.ts` +useBackupPolicy + useUpdateBackupPolicy
- `mock/data/backup.ts` +BackupPolicy interface + DEFAULT_BACKUP_POLICY

**webcode**:
- `pages/backup/BackupPolicy/index.tsx`：删除 `console.log`；form 用 GET 初始化；保存调 mutate；删除 `<input>` 直接 DOM 标签换为 AntD `Input`
- 4 个 panel header 加 "尚未生效" Tag

### 7.5 Singleton 实现策略

```go
// PolicyRepository 接口
type PolicyRepository interface {
    Get(ctx context.Context) (*BackupPolicy, error)  // ORDER BY updated_at DESC LIMIT 1
    Upsert(ctx context.Context, p *BackupPolicy) error  // if Get nil → INSERT else UPDATE by id
}
```

应用层保证 singleton；DB 不加 unique 约束（留 per-tenant 扩展空间）。

### 7.6 i18n 新 key

```
backup.policy.persistedNotEnforced — "持久化（executor 待集成）" / "Stored only (executor integration pending)"
backup.policy.persistedNotEnforcedTooltip — 长说明
backup.policy.saveSuccess
backup.policy.saveFailed
backup.policy.loadFailed
```

5 keys × 2 locale = 10 行。

### 7.7 多皮肤影响

- frontend-core 类型 + hook + mock + i18n 改动 → v2/v3 自动继承（不破）
- v2/v3 不引用 BackupPolicy 页面 → 零 UI 影响
- 仅 webcode `BackupPolicy/index.tsx` 单文件 UI 改动

---

## 8. DoD

- [ ] PRD 七要素 + 运营商一致矩阵
- [ ] V1-V9 全部测试通过
- [ ] go build / vet / test -race 全过
- [ ] tsc / lint baseline 不退化
- [ ] migration 000047 up/down 配对 + check-migrations.sh 通过
- [ ] handler/service/repository test 各 ≥3 用例
- [ ] e2e_verify.sh +1 claim
- [ ] frontend BackupPolicy 删除 console.log
- [ ] backlog T-0071 → done + 3 followup（T-0073/0074/0075）登记
- [ ] R-102 进展更新（Backup 4/4 持久化闭环；enforcement followup 待）

---

## 9. 风险评估

| 风险 | 缓解 |
|------|------|
| Singleton 语义在并发下 race（两个 PUT 同时 INSERT）| Upsert 用 `INSERT ... ON CONFLICT DO UPDATE` 或事务 + SELECT FOR UPDATE；最简：`INSERT (...) RETURNING id` 后续 PUT 用 UPDATE WHERE id |
| schema 19 列后续要加 unique tenant_id 时迁移 | 初版无 unique；future PR 加 `ALTER TABLE ... ADD CONSTRAINT` |
| UI 显示 "executor 待集成" 标签易混淆 | i18n 长说明 + Tag 颜色 default（不是 success/error）+ tooltip |
| 加密字段持久化但不执行 → 用户开启加密但备份未加密 → security false sense | UI 标签明确说"未生效"；加密 panel 加 Alert 横幅强调安全风险；T-0075 followup 优先级提为 P1 |
| FK ftp_config_id 引用删除场景 | ON DELETE SET NULL；backupPolicy 仍保留其他字段 |

---

*PRD by /dev-pipeline pick T-0071 ULTRATHINK A 方案。重点：持久化先行 + enforcement 拆 followup。*
