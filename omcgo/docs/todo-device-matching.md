# 设备分组自动匹配 - 待办事项

> 本文档记录设备分组自动匹配功能的后续扩展点，供后续迭代参考。

---

## 已完成功能

- [x] 数据库字段：`device_groups` 表新增 `matching_mode`、`name_rule_list`、`lac_list`、`tac_list`
- [x] 后端模型：`DeviceGroup` 结构体支持匹配规则字段
- [x] 匹配引擎：`DeviceMatcher` 实现设备名称/LAC/TAC 三种匹配模式
- [x] 前端界面：分组编辑对话框支持匹配规则配置
- [x] API 集成：创建/更新分组时传递匹配规则数据

---

## 待扩展功能

### 1. 设备注册/更新时自动触发匹配

**触发时机**：
- 新设备首次 Inform（0-BOOTSTRAP 事件）
- 设备信息更新（`device_info` 表变更）
- 设备 LAC/TAC 参数变更

**实现位置**：
- `internal/acs/handler.go` - Inform 处理后
- `internal/device/service.go` - 设备信息更新后

**代码示例**：
```go
// 在设备注册后触发匹配
func (s *DeviceService) OnDeviceRegistered(ctx context.Context, deviceID uuid.UUID) error {
    // 1. 获取设备信息（名称、LAC、TAC）
    device, err := s.repo.GetByID(ctx, deviceID)
    if err != nil {
        return err
    }

    // 2. 构建匹配请求
    req := topology.MatchRequest{
        DeviceID:   device.ID,
        DeviceName: device.Name,
        LAC:        device.LAC,
        TAC:        device.TAC,
    }

    // 3. 执行匹配并分配分组
    result, err := s.matcher.AssignDeviceToGroup(ctx, req)
    if err != nil {
        s.logger.Warn("auto match device failed", zap.Error(err))
    }
    if result != nil {
        s.logger.Info("device auto-assigned to group",
            zap.String("device_id", deviceID.String()),
            zap.String("group_id", result.GroupID.String()),
            zap.String("matched_by", string(result.MatchedBy)),
        )
    }
    return nil
}
```

**依赖**：
- 需要从 `device_info` 表获取 `device_name`
- 需要从 TR069 参数获取 LAC/TAC

---

### 2. 分组规则变更后重新匹配

**触发时机**：
- L2 分组创建/更新匹配规则
- 分组匹配规则被删除

**实现位置**：
- `internal/topology/service.go` - 分组更新后

**代码示例**：
```go
// 在分组规则变更后触发重新匹配
func (s *TopologyService) OnGroupRulesChanged(ctx context.Context, groupID uuid.UUID) error {
    // 1. 获取所有设备
    devices, err := s.deviceRepo.GetAll(ctx)
    if err != nil {
        return err
    }

    // 2. 构建匹配请求列表
    var matchRequests []topology.MatchRequest
    for _, d := range devices {
        matchRequests = append(matchRequests, topology.MatchRequest{
            DeviceID:   d.ID,
            DeviceName: d.Name,
            LAC:        d.LAC,
            TAC:        d.TAC,
        })
    }

    // 3. 批量匹配
    matched, err := s.matcher.BatchMatchDevices(ctx, matchRequests)
    if err != nil {
        return err
    }

    s.logger.Info("group rules changed, re-matched devices",
        zap.String("group_id", groupID.String()),
        zap.Int("matched_count", matched),
    )
    return nil
}
```

**优化建议**：
- 仅匹配未分配分组的设备（避免重复分配）
- 或者：仅匹配属于当前分组的设备（更新分组内设备）
- 使用 NATS 异步处理大规模匹配任务

---

### 3. LAC/TAC 数据源集成

**数据来源**：

| 字段 | LTE (4G) | 5G NR | 数据模型 |
|------|----------|-------|---------|
| LAC | `Device.FAPCellConfig.LTE.RAN.LAC` | - | TR-098 / TR-181 |
| TAC | - | `Device.FAPCellConfig.NR.TAC` | TR-181 |

**实现位置**：
- `internal/device/service.go` - 设备信息同步

**代码示例**：
```go
// 从 TR069 参数同步 LAC/TAC 到 device_info
func (s *DeviceService) SyncLacTacFromTR069(ctx context.Context, deviceID uuid.UUID, params map[string]string) error {
    var lac, tac *int

    // 解析 LAC (LTE)
    if lacStr, ok := params["Device.FAPCellConfig.LTE.RAN.LAC"]; ok {
        if val, err := strconv.Atoi(lacStr); err == nil {
            lac = &val
        }
    }

    // 解析 TAC (5G)
    if tacStr, ok := params["Device.FAPCellConfig.NR.TAC"]; ok {
        if val, err := strconv.Atoi(tacStr); err == nil {
            tac = &val
        }
    }

    // 更新 device_info
    return s.repo.UpdateLacTac(ctx, deviceID, lac, tac)
}
```

**数据库变更**：
- `device_info` 表需要新增 `lac` 和 `tac` 字段

```sql
ALTER TABLE device_info ADD COLUMN lac INTEGER;
ALTER TABLE device_info ADD COLUMN tac INTEGER;

COMMENT ON COLUMN device_info.lac IS '位置区码 (LTE/4G)';
COMMENT ON COLUMN device_info.tac IS '跟踪区码 (5G NR)';
```

---

### 4. 前端增强

**功能点**：
- [ ] 分组列表显示匹配规则摘要
- [ ] 批量匹配预览（应用规则前预览将匹配的设备）
- [ ] 匹配规则测试工具（输入设备名称，测试是否匹配）
- [ ] 匹配历史记录（设备何时被分配到分组）

---

### 5. API 扩展

**新增端点**：

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/v1/device-groups/{id}/match` | 手动触发分组匹配 |
| POST | `/api/v1/device-groups/match-all` | 全量重新匹配 |
| POST | `/api/v1/device-groups/{id}/preview` | 预览匹配结果（不实际分配） |
| GET | `/api/v1/devices/{id}/match-result` | 获取设备匹配结果 |

---

### 6. 性能优化

**场景**：10 万设备 + 100 分组

**优化点**：
- 缓存分组规则（Redis），避免每次查询数据库
- 批量匹配使用 goroutine 并发处理
- 增量匹配：仅匹配变更的设备

**代码示例**：
```go
// 使用 Redis 缓存分组规则
func (m *DeviceMatcher) getCachedGroups(ctx context.Context) ([]DeviceGroup, error) {
    cacheKey := "device_groups:matching_rules"

    // 尝试从缓存获取
    cached, err := m.redis.Get(ctx, cacheKey).Bytes()
    if err == nil {
        var groups []DeviceGroup
        if json.Unmarshal(cached, &groups) == nil {
            return groups, nil
        }
    }

    // 从数据库获取并缓存
    groups, err := m.repo.GetTreeWithCounts(ctx)
    if err != nil {
        return nil, err
    }

    data, _ := json.Marshal(groups)
    m.redis.Set(ctx, cacheKey, data, 5*time.Minute)

    return groups, nil
}
```

---

## 优先级建议

| 优先级 | 功能 | 理由 |
|--------|------|------|
| P0 | LAC/TAC 数据源集成 | 匹配功能依赖数据 |
| P1 | 设备注册时自动匹配 | 核心业务价值 |
| P2 | 规则变更后重新匹配 | 运维便利性 |
| P3 | 前端增强 | 用户体验提升 |
| P3 | API 扩展 | 运维便利性 |
| P3 | 性能优化 | 规模化后需要 |

---

## 相关文件

- `omcgo/internal/topology/matcher.go` - 匹配引擎
- `omcgo/internal/topology/model.go` - 数据模型
- `omcgo/internal/topology/pg_repository.go` - 数据持久层
- `omcmb/webcode/src/pages/device/DeviceGrouping/` - 前端页面
- `omcgo/migrations/000074_add_group_matching_rules.up.sql` - 数据库迁移
