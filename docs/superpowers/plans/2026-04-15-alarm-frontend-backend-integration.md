# 告警模块前后端 API 对接计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 修复前端告警 API 层（`alarmApi.ts`）与后端实际接口的差异，确保字段映射、请求参数、批量操作全部正确对接。

**Architecture:** 前端通过 Axios 拦截器自动转换 camelCase ↔ snake_case。后端使用 snake_case JSON tag。前端类型定义使用 camelCase，API 层负责 `BackendXxx` → `Xxx` 的映射。

**Tech Stack:** React 19 + TypeScript + Ant Design 5 + TanStack React Query + Axios

---

## 差异分析

### 核心问题清单

| # | 问题 | 严重性 | 文件 |
|---|------|--------|------|
| 1 | `alarmApi.ts` 分页参数发送 `pageSize`/`sortField`/`sortOrder`，但 Axios 拦截器只转换了 `pageSize→page_size`，未转换 `sortField→sort_by`、`sortOrder→sort_dir` | 高 | `alarmApi.ts:140-145` |
| 2 | 后端告警列表返回 `device_name`/`technology`/`alarm_source`/`event_type`/`probable_cause`/`is_read` 等字段，前端 `mapBackendAlarm` 未映射 | 高 | `alarmApi.ts:55-80` |
| 3 | 后端支持批量确认/清除接口 `POST /alarms/active/batch/acknowledge` 和 `POST /alarms/active/batch/clear`，前端用循环逐条调用 | 高 | `alarmApi.ts:201-215` |
| 4 | 后端告警规则接口为 `/alarms/alarm-filters`，前端调用 `/alarms/rules`（404） | 高 | `alarmApi.ts:237,272,306,315` |
| 5 | 后端 `AlarmFilterRule` 结构体（`name/filter_type/alarm_codes/device_ids/action/priority`）与前端 `BackendAlarmRule`（`rule_name/rule_type/conditions/actions`）完全不匹配 | 高 | `alarmApi.ts:92-134` |
| 6 | 后端缺少 `keyword` 搜索、`event_type` 过滤、`ne_type` 过滤、`ack_status` 过滤等参数 | 中 | `alarmApi.ts:136-158` vs `handler.go:43-52` |
| 7 | 后端缺少 `markRead` 告警已读标记，前端 Header 的 `AlarmBadges` 组件可能需要此接口 | 中 | `alarmApi.ts` 缺失 |
| 8 | 后端有 `alarm-libraries`（告警支撑库）和 `alarm-filters`（过滤规则）两个独立子模块，前端只有 `AlarmRules` 一个页面 | 低 | 前端页面结构 |

---

### Task 1: 修复分页参数转换

**Files:**
- Modify: `omcmb/webcode/src/services/http.ts`

- [ ] **Step 1: 确认 `http.ts` 的 `transformParams` 已覆盖 `sortField` 和 `sortOrder`**

读取 `http.ts:12-16` 的 `paramKeyMap`，确认是否已包含 `sortField → sort_by` 和 `sortOrder → sort_dir` 的映射。

- [ ] **Step 2: 运行前端类型检查**

Run: `cd /home/baicells/goomc/omcmb/webcode && npx tsc --noEmit`
Expected: 无新增错误

---

### Task 2: 补全告警字段映射

**Files:**
- Modify: `omcmb/webcode/src/services/api/alarmApi.ts`
- Modify: `omcmb/webcode/src/types/alarm.ts`

- [ ] **Step 1: 扩展 `BackendAlarm` 接口，补充后端返回但前端未映射的字段**

在 `alarmApi.ts` 的 `BackendAlarm` 接口中添加缺失字段：

```typescript
interface BackendAlarm {
  // 已有字段...
  id: string;
  device_id: string;
  device_sn: string;
  carrier: string;
  severity: number;
  alarm_type: string;
  alarm_code: string;
  description: string;
  status: string;
  raised_at: string;
  acknowledged_at?: string;
  cleared_at?: string;
  acknowledged_by?: string;
  additional_info?: Record<string, string>;
  created_at: string;
  updated_at: string;
  // 新增缺失字段
  device_name?: string;
  technology?: string;
  alarm_source?: string;
  event_type?: string;
  probable_cause?: string;
  is_read?: boolean;
  ack_count?: number;
  first_raised_at?: string;
  last_updated_at?: string;
}
```

- [ ] **Step 2: 更新 `mapBackendAlarm` 函数，映射新增字段到前端 `Alarm` 类型**

更新映射逻辑，利用后端新增字段：

```typescript
function mapBackendAlarm(ba: BackendAlarm): Alarm {
  return {
    id: ba.id,
    alarmCode: ba.alarm_code,
    alarmName: ba.alarm_type || ba.alarm_code,
    severity: severityNumToStr[ba.severity] || 'warning',
    deviceSn: ba.device_sn,
    deviceName: ba.device_name || ba.device_sn,  // 优先使用后端 device_name
    neType: ba.technology || '',
    alarmContent: ba.description,
    alarmTime: ba.raised_at,
    clearTime: ba.cleared_at,
    duration: ba.cleared_at
      ? Math.floor(
          (new Date(ba.cleared_at).getTime() - new Date(ba.raised_at).getTime()) / 60000
        )
      : undefined,
    ackStatus: ba.acknowledged_at ? 'acknowledged' : 'unacknowledged',
    ackUser: ba.acknowledged_by,
    ackTime: ba.acknowledged_at,
    alarmSource: ba.alarm_source || ba.carrier,
    alarmLocation: ba.additional_info?.location,
    alarmType: ba.alarm_type as AlarmType,
    isActive: ba.status === 'active' || ba.status === 'acknowledged',
    unread: ba.is_read ? '0' : '1',  // 新增：根据 is_read 映射
  };
}
```

- [ ] **Step 3: 运行类型检查**

Run: `cd /home/baicells/goomc/omcmb/webcode && npx tsc --noEmit`
Expected: 无新增错误

---

### Task 3: 使用后端批量操作接口

**Files:**
- Modify: `omcmb/webcode/src/services/api/alarmApi.ts`

- [ ] **Step 1: 修改 `acknowledgeAlarms` 使用批量接口**

将逐条循环改为调用后端批量接口：

```typescript
async acknowledgeAlarms(ids: string[], note?: string): Promise<void> {
  await http.post('/alarms/active/batch/acknowledge', {
    ids,
    acknowledged_by: note || 'operator',
  });
},
```

- [ ] **Step 2: 修改 `clearAlarms` 使用批量接口**

```typescript
async clearAlarms(ids: string[]): Promise<void> {
  await http.post('/alarms/active/batch/clear', { ids });
},
```

- [ ] **Step 3: 新增 `markAlarmRead` 方法**

```typescript
async markAlarmRead(id: string): Promise<void> {
  await http.post(`/alarms/active/${id}/read`);
},
```

- [ ] **Step 4: 运行类型检查**

Run: `cd /home/baicells/goomc/omcmb/webcode && npx tsc --noEmit`
Expected: 无新增错误

---

### Task 4: 修复告警规则 API 路径和数据结构

**Files:**
- Modify: `omcmb/webcode/src/services/api/alarmApi.ts`
- Modify: `omcmb/webcode/src/types/alarm.ts`
- Modify: `omcmb/webcode/src/pages/alarm/AlarmRules/index.tsx`
- Modify: `omcmb/webcode/src/pages/alarm/AlarmRules/AlarmRuleDrawer.tsx`

- [ ] **Step 1: 更新 `BackendAlarmRule` 接口匹配后端实际结构**

后端 `AlarmFilterRule` 实际返回字段：`id`, `name`, `filter_type`, `alarm_sources`, `alarm_codes`, `device_ids`, `device_group_ids`, `action`, `acknowledge_desc`, `priority`, `enabled`, `created_at`, `updated_at`

```typescript
interface BackendAlarmRule {
  id: string;
  name: string;
  filter_type: string;  // alarm_source | alarm_code | device | device_group
  alarm_sources: string[];
  alarm_codes: string[];
  device_ids: string[];
  device_group_ids: string[];
  action: string;  // default | ignore | auto_acknowledge | auto_clear
  acknowledge_desc: string;
  priority: number;
  enabled: boolean;
  created_at: string;
  updated_at: string;
}
```

- [ ] **Step 2: 重写 `mapBackendAlarmRule` 映射函数**

根据后端实际数据结构重写映射，使前端 `AlarmRule` 类型的 `conditions`/`actions` 字段从后端的 `alarm_codes`/`alarm_sources`/`action` 字段派生：

```typescript
function mapBackendAlarmRule(br: BackendAlarmRule): AlarmRule {
  // 从后端字段派生前端 conditions
  const conditions: AlarmRuleCondition[] = [];
  if (br.alarm_codes?.length) {
    conditions.push({ field: 'alarm_code', operator: 'in', value: br.alarm_codes });
  }
  if (br.alarm_sources?.length) {
    conditions.push({ field: 'alarm_source', operator: 'in', value: br.alarm_sources });
  }
  if (br.device_ids?.length) {
    conditions.push({ field: 'device_id', operator: 'in', value: br.device_ids });
  }
  if (br.device_group_ids?.length) {
    conditions.push({ field: 'device_group_id', operator: 'in', value: br.device_group_ids });
  }

  // 从后端 action 派生前端 actions
  const actions: AlarmRuleAction[] = [];
  if (br.action === 'ignore') {
    actions.push({ type: 'suppress' });
  } else if (br.action === 'auto_acknowledge') {
    actions.push({ type: 'notify', target: 'system', params: { desc: br.acknowledge_desc } });
  } else if (br.action === 'auto_clear') {
    actions.push({ type: 'webhook', target: 'system' });
  }

  return {
    id: br.id,
    ruleName: br.name,
    ruleType: br.filter_type,
    severity: 'warning', // 后端规则无 severity，默认 warning
    enabled: br.enabled,
    conditions,
    actions,
    createTime: br.created_at,
    updateTime: br.updated_at,
  };
}
```

- [ ] **Step 3: 修复 `getRules`、`createRule`、`updateRule`、`deleteRules` 的 API 路径**

将 `/alarms/rules` 改为 `/alarms/alarm-filters`：

```typescript
async getRules(params: PageRequest): Promise<PageResponse<AlarmRule>> {
  const query: Record<string, unknown> = {
    page: params.page,
    pageSize: params.pageSize,
  };
  const { data } = await http.get<BackendListResponse<BackendAlarmRule>>(
    '/alarms/alarm-filters',
    { params: query }
  );
  return {
    items: (data.items || []).map(mapBackendAlarmRule),
    total: data.total,
    page: data.page,
    pageSize: data.page_size,
  };
},
```

`createRule` 路径改为 `/alarms/alarm-filters`，请求体适配后端结构：

```typescript
async createRule(data: Omit<AlarmRule, 'id' | 'createTime' | 'updateTime'>): Promise<AlarmRule> {
  const payload: Record<string, unknown> = {
    name: data.ruleName,
    filter_type: data.ruleType || 'alarm_code',
    enabled: data.enabled,
    action: 'default',
    priority: 0,
    alarm_codes: [],
    alarm_sources: [],
    device_ids: [],
    device_group_ids: [],
  };
  // 从前端 conditions 反向填充后端字段
  for (const c of data.conditions || []) {
    if (c.field === 'alarm_code' && Array.isArray(c.value)) payload.alarm_codes = c.value;
    if (c.field === 'alarm_source' && Array.isArray(c.value)) payload.alarm_sources = c.value;
    if (c.field === 'device_id' && Array.isArray(c.value)) payload.device_ids = c.value;
    if (c.field === 'device_group_id' && Array.isArray(c.value)) payload.device_group_ids = c.value;
  }
  // 从前端 actions 反向填充
  for (const a of data.actions || []) {
    if (a.type === 'suppress') payload.action = 'ignore';
    if (a.type === 'notify' && a.params?.desc) payload.acknowledge_desc = a.params.desc;
    if (a.type === 'webhook') payload.action = 'auto_clear';
  }
  const { data: created } = await http.post<BackendAlarmRule>('/alarms/alarm-filters', payload);
  return mapBackendAlarmRule(created);
},
```

`updateRule` 路径改为 `/alarms/alarm-filters/${id}`，同理适配请求体。

`deleteRules` 路径改为 `/alarms/alarm-filters/${id}`。

- [ ] **Step 4: 新增 `toggleRule` 方法**

```typescript
async toggleRule(id: string): Promise<void> {
  await http.post(`/alarms/alarm-filters/${id}/toggle`);
},
```

- [ ] **Step 5: 更新前端 `AlarmRules/index.tsx` 和 `AlarmRuleDrawer.tsx` 页面组件**

检查页面组件是否使用了 `ruleType`/`severity`/`conditions`/`actions` 字段，适配新的映射逻辑。特别是：
- `AlarmRuleDrawer.tsx` 中的表单字段（`ruleType` 对应后端 `filter_type`，`conditions` 对应后端的 `alarm_codes`/`alarm_sources` 等）
- `index.tsx` 中规则列表的显示逻辑

- [ ] **Step 6: 运行类型检查**

Run: `cd /home/baicells/goomc/omcmb/webcode && npx tsc --noEmit`
Expected: 无新增错误

---

### Task 5: 补充 useAlarms hooks

**Files:**
- Modify: `omcmb/webcode/src/hooks/api/useAlarms.ts`

- [ ] **Step 1: 新增 `useMarkAlarmRead` hook**

```typescript
export function useMarkAlarmRead() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => api.markAlarmRead(id),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['alarms'] });
    },
  });
}
```

- [ ] **Step 2: 新增 `useToggleAlarmRule` hook**

```typescript
export function useToggleAlarmRule() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => api.toggleRule(id),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['alarms', 'rules'] });
    },
  });
}
```

- [ ] **Step 3: 运行类型检查**

Run: `cd /home/baicells/goomc/omcmb/webcode && npx tsc --noEmit`
Expected: 无新增错误

---

### Task 6: 后端补充缺失的查询参数绑定

**Files:**
- Modify: `omcgo/internal/alarm/handler.go`

- [ ] **Step 1: 在 `alarmQuery` 结构体中添加缺失的查询参数绑定**

```go
type alarmQuery struct {
    DeviceID    string `form:"device_id"`
    DeviceSN    string `form:"device_sn"`
    Carrier     string `form:"carrier"`
    Severity    string `form:"severity"`
    Status      string `form:"status"`
    AlarmType   string `form:"alarm_type"`
    IsRead      string `form:"is_read"`
    DeviceName  string `form:"device_name"`
    StartTime   string `form:"start_time"`
    EndTime     string `form:"end_time"`
    model.ListRequest
}
```

- [ ] **Step 2: 在 `ListActive` 中绑定新增字段到 `AlarmFilter`**

在 `ListActive` handler 中添加映射：

```go
if q.AlarmType != "" {
    filter.AlarmType = &q.AlarmType
}
if q.IsRead != "" {
    read := q.IsRead == "true"
    filter.IsRead = &read
}
if q.DeviceName != "" {
    filter.DeviceName = &q.DeviceName
}
```

- [ ] **Step 3: 同步更新 `ListHistory`，绑定 `AlarmType` 和 `Carrier`**

- [ ] **Step 4: 编译验证**

Run: `cd /home/baicells/goomc/omcgo && go build ./...`
Expected: 编译通过

---

### Task 7: 验证前后端联调

- [ ] **Step 1: 切换为真实 API 模式**

确认 `VITE_USE_MOCK` 环境变量不为 `true`，或前端 `.env` 文件中未设置该变量。

- [ ] **Step 2: 启动前端开发服务器**

Run: `cd /home/baicells/goomc/omcmb/webcode && npm run dev`

- [ ] **Step 3: 手动验证以下场景**

1. 当前告警列表正常加载，字段显示正确（设备名称、告警级别、状态等）
2. 告警详情页正常展示
3. 确认告警（单条和批量）操作成功
4. 清除告警操作成功
5. 告警规则页面正常加载（路径修复后不再 404）
6. 创建/编辑/删除告警规则操作成功
7. 告警统计数字正确
8. Header 告警角标显示正确

- [ ] **Step 4: 提交代码**

```bash
cd /home/baicells/goomc/omcgo && git add internal/alarm/handler.go && git commit -m "fix(alarm): 补充活动告警查询参数绑定 alarm_type/is_read/device_name"
cd /home/baicells/goomc/omcmb && git add src/services/api/alarmApi.ts src/hooks/api/useAlarms.ts src/types/alarm.ts && git commit -m "fix(alarm): 修复告警API与后端接口对接差异"
```
