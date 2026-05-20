# 设备分组同步 + 设备列表筛选体验完善

> 状态：**待审核**
> 提出时间：2026-05-20
> 涉及代码：`omcmb/webcode/src/pages/device/{DeviceList,DeviceGrouping}/**` + `omcmb/webcode/src/components/FilterBar/**` + `omcmb/frontend-core/src/{hooks/api/useSystem,services/api/deviceApi}.ts` + `omcgo/internal/{topology,device}/**` + `omcgo/migrations/`

---

## 0. 用户原始诉求（6 项）

1. 编辑设备分组、改匹配规则到包含 "E2E-Updated" → 设备列表里名字含 "E2E-Updated" 的设备**未自动入组**
2. 分析根因 — 是否字段未对齐 / 异步未实现？此前明确要求过：分组匹配规则变化时异步重算成员
3. "导出" 按钮放在筛选条件 "搜索" 按钮**右侧**
4. 筛选展开 / 收起：展开是两行，除第一个搜索框外其它下拉**固定宽度**；目前设备分组下拉变得很长
5. 第一个搜索框输入 "E2E,007219" → 搜索**不生效**；需支持英文逗号分隔多关键字（"E2E" 匹配 name、"007219" 匹配 SN）
6. 筛选项数据源混乱：
   - 在线状态应该**只有在线 / 离线**，目前还有 "同步中" "同步失败"
   - 设备分组下拉应从 `device/group` 页面同一份数据源拉
   - 其它下拉若无固有来源，应统一走**系统管理-字典管理** (system/data-dictionary) 维护
   - 完成项目里现有"筛选项"清点 + 字典初始化

---

## 1. 现状核查结论

| 维度 | 当前实现 | 与诉求差距 |
|---|---|---|
| 后端分组匹配 schema | ✅ `device_groups.{matching_mode, name_rule_list, lac_list, tac_list, serial_number_list}`，`device_group_members(source_type, source_rule_id)` 区分手工 vs 规则 | 完整 |
| 后端 UpdateGroup → 异步同步 | ✅ `topology/service.go:265-284` 改字段后调 `fireGroupMatch(group)` → `GroupMatchEngine.MatchGroup()` | 完整 |
| 后端实时 / 定时同步 | ✅ `device.registered` 事件 + @hourly `ReEvaluateAll()` cron | 完整 |
| **前端 L2 分组编辑 — 加载** | ❌ `openEditLevel2` (`useGroupActions.tsx:152`) **不回填**现有 matchingMode/nameFilters → 用户看到的是"空白"表单 | 致命 bug |
| **前端 L2 分组编辑 — 保存** | ❌ `handleSaveEditLevel2` (`useGroupActions.tsx:272-275`) 仅提交 `{ name }`，**丢全部匹配规则** | 致命 bug — 用户诉求 1/2 的根因 |
| 前端 `UpdateGroupArgs` type | ❌ 仅 `{ name, parent_id, remark }`，无 matching rule 字段 → 类型层就过滤了 | bug |
| 后端 BuildSearchOR | ✅ `device/search.go:41` 按英文逗号分裂、最多 50 关键字、生成 sq.Or ILIKE 链 | 完整 |
| 前端 searchText → search 参数 | ✅ `deviceApi.ts:310` `if (params.searchText) query.search = params.searchText` | 完整 |
| 前端 DeviceList `handleSearch` | ✅ 已切英文逗号、前端截 50 | 完整（用户报"不生效"待复现 — 可能是缓存或具体字段集偏差） |
| 导出按钮位置 | ❌ 在 `ListPageLayout.extra`（页面右上） | 应在 FilterBar 搜索按钮右侧 |
| FilterBar 是否支持 extra 区 | ✅ `FilterBarProps.extra: React.ReactNode` 已存在 | 直接接入 |
| `groupId` 字段 width | ❌ 未设 → 默认 flex:1（变长） | 加 `width: 160` |
| connStatus 选项 | ❌ 硬编码 4 项含"同步中"(3)/"同步失败"(2) | 收敛到 online(1)/offline(0) |
| groupId 选项 | ❌ `options: []` 占位 | 接 `useDeviceGroups()` |
| 字典系统 | ✅ 后端 `sys_dictionaries` + `sys_dictionary_details` + Service/Handler/Repo 完整；前端 `useDictionary(code)` 也有 | 缺**种子数据**与 Filter 接线 |

**根因总结（针对诉求 1+2）**：
- 后端异步同步链路完整。
- 前端 L2 编辑入口 `openEditLevel2` 不回填 + `handleSaveEditLevel2` 只发 name。
- 结果：用户以为改了规则，实际**根本没提交规则字段**给后端，service 收到 `req.NameRuleList == nil` 跳过更新 → 旧规则保留 → `fireGroupMatch` 用旧规则跑了一遍 → 设备不变。

---

## 2. 决策清单

| # | 决策点 | 选定方案 |
|---|---|---|
| D1 | L2 编辑 "加载" 阶段，从 group 对象哪些字段反推 UI 状态？ | grp.matchingMode → form.matchingMode；grp.nameRuleList → editLevel2NameFilters.replace([...])；grp.lacList/tacList → form.tacRag = serializeRangeString(list) |
| D2 | L2 编辑保存语义 | 全量替换 — 用户在 UI 上看到的就是最终落库的；不做增量合并避免歧义 |
| D3 | serialNumber 匹配模式前端 | 本期**不开放**（后端已支持，但前端未来再补开关 + 文本框） |
| D4 | 在线状态收敛 | UI 仅展示 online(1) / offline(0)；DB 仍可写其他值（如 syncing 用于内部状态），但筛选不暴露 |
| D5 | groupId 多选 vs 单选 | 沿用 `multi-select`（多选） |
| D6 | 字典 code 命名 | 全小写下划线分隔：`network_type`、`op_state`；保留 `product_type` 已有 |
| D7 | 字典 not-loaded fallback | useDictionary 加载中 → options=[]，下拉灰显 / 占位提示 |
| D8 | 多关键字搜索的字段集 | 沿用后端 `device_info_pg_repository.go:273` 的字段集（serial_number / device_name / ip_address / mac / eci / pci）— 用户提的 "name + SN" 已被包含；不再额外缩小 |
| D9 | 导出按钮放置 | 走 FilterBar `extra` 槽，渲染顺序：搜索 / 重置 / 展开 — `<extra>` — 即导出在最右 |

---

## 3. 实现项（按依赖关系排序）

### R1 — 修复 L2 分组编辑（前端 / **核心 bug**）

**文件**：`omcmb/webcode/src/pages/device/DeviceGrouping/useGroupActions.tsx`

#### R1.1 扩展 type

```typescript
interface UpdateGroupArgs {
  id: string;
  data: {
    name?: string;
    parent_id?: string;
    remark?: string;
    matching_mode?: string;
    name_rule_list?: NameFilterItem[];
    lac_list?: number[];
    tac_list?: number[];
  };
}
```

#### R1.2 `openEditLevel2` 回填

```typescript
const openEditLevel2 = useCallback((groupId: string) => {
  const grp = groups.find((g) => g.id === groupId);
  if (!grp) return;
  setEditLevel2GroupId(groupId);
  editLevel2Form.resetFields();
  editLevel2Form.setFieldsValue({
    name: grp.name,
    matchingMode: (grp.matchingMode as AddChildFormValues['matchingMode']) || 'deviceName',
    tacRag: serializeRangeForUI(grp.matchingMode, grp.lacList, grp.tacList),
  });
  // 回填 name filters
  if (grp.matchingMode === 'deviceName' && Array.isArray(grp.nameRuleList) && grp.nameRuleList.length > 0) {
    editLevel2NameFilters.replace(grp.nameRuleList);
  } else {
    editLevel2NameFilters.reset();
  }
  setEditLevel2DrawerOpen(true);
}, [editLevel2Form, editLevel2NameFilters, groups]);
```

（需要 `useNameFilters` 暴露一个 `replace(filters: NameFilterItem[])` 方法，若没有就新增）

#### R1.3 `handleSaveEditLevel2` 全量提交

```typescript
const handleSaveEditLevel2 = useCallback(async () => {
  try {
    const values = await editLevel2Form.validateFields();
    if (!editLevel2GroupId) return;

    let matching_mode: string | undefined;
    let name_rule_list: NameFilterItem[] | undefined;
    let lac_list: number[] | undefined;
    let tac_list: number[] | undefined;

    if (values.matchingMode === 'deviceName') {
      matching_mode = 'deviceName';
      name_rule_list = editLevel2NameFilters.filters.filter((f) => f.value && f.value.trim() !== '');
      // 显式清空 lac/tac，让后端 = null
      lac_list = [];
      tac_list = [];
    } else if (values.matchingMode === 'lac') {
      matching_mode = 'lac';
      lac_list = parseRangeString(values.tacRag || '');
      name_rule_list = [];
      tac_list = [];
    } else if (values.matchingMode === 'tac') {
      matching_mode = 'tac';
      tac_list = parseRangeString(values.tacRag || '');
      name_rule_list = [];
      lac_list = [];
    }

    await updateGroupMutation.mutateAsync({
      id: editLevel2GroupId,
      data: {
        name: values.name,
        matching_mode,
        name_rule_list,
        lac_list,
        tac_list,
      },
    });
    void message.success(t('common.success'));
    setEditLevel2DrawerOpen(false);
    void refetchGroups();
  } catch (err) {
    handleSaveError(err);
  }
}, [editLevel2Form, editLevel2GroupId, editLevel2NameFilters.filters, updateGroupMutation, message, t, refetchGroups, handleSaveError]);
```

**验收**：编辑 L2 分组的匹配名规则改为含 `*E2E-Updated*`，点保存 → 2 秒内设备列表里名含 "E2E-Updated" 的设备出现在该分组下。

### R2 — 后端异步同步验证（已存在，加 E2E 断言）

后端无需改动。在 `e2e_verify.sh` 加一条断言：

```bash
claim "device groups: UPDATE 后 settle 触发 group_match_engine 异步重算（路由 200/401）"
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X PUT \
    "$API/device/groups/$W2D_BAD_UUID" -H "$W2D_AUTH" -H "Content-Type: application/json" \
    -d '{"matching_mode":"deviceName","name_rule_list":[{"operator":"contains","value":"E2E-Updated"}]}')
check_status_in "device-group-update: PUT 路由已挂" "200 400 404 401" "$HTTP_CODE"
```

**注**：完整设备入组验证需 seed 数据 + 异步等待，列入 P2 集成测试，本期不阻塞。

### R3 — 导出按钮位置

**文件**：`omcmb/webcode/src/pages/device/DeviceList/index.tsx`

把 `ListPageLayout.extra` 中的导出按钮**剪切**到 `<FilterBar ... extra={<ExportButton .../>} />`。

由于 FilterBar 已暴露 `extra: React.ReactNode` 槽（`FilterBar/index.tsx:40`），仅作两步：
1. 抽出 ExportButton 子组件（已有逻辑：菜单 / 导出全部 / 导出选中）
2. 把它作为 `<FilterBar extra={...}>` 的 prop 传入

**验收**：DOM 上 `<button>搜索</button> ... <button>导出</button>` 同行，导出按钮靠右。

### R4 — groupId 下拉宽度对齐

**文件**：`omcmb/webcode/src/pages/device/DeviceList/index.tsx` FILTER_FIELDS

```typescript
{
  name: 'groupId',
  label: t('device.groupName'),
  type: 'multi-select',
  width: 160,         // ← 新增，与其它下拉对齐
  options: [],        // R6b 接管
}
```

### R5 — 多关键字搜索端到端调试

**核查清单**：
1. ✅ 前端 `handleSearch` 切分逗号 → 仍以逗号 join 回传至 URL state
2. ✅ `deviceApi.getList` 透传 `searchText → search` 参数
3. ✅ 后端 `BuildSearchOR` 切分 → ILIKE OR 链
4. ❓ 前端 `FilterBar` 是否对 input 类型字段做了 trim / 强制单值？— 需复核

**操作**：用 curl 直接验证后端 ↔ 前端调用是否一致：

```bash
curl "$API/device/list?search=E2E,007219&page=1&page_size=10" -H "$W2D_AUTH"
```

若返 200 且包含双重匹配设备 → 后端 OK，前端组装无问题；若不包含 → 后端字段集不覆盖期望字段。

**若发现端到端真 bug**：补 fix（最可能是 URL encoding 或 trim 问题）。

### R6a — connStatus 收敛

**文件**：`omcmb/webcode/src/pages/device/DeviceList/index.tsx`

```typescript
options: [
  { label: t('filter.conn.normal'), value: '1' },       // 在线
  { label: t('filter.conn.disconnected'), value: '0' }, // 离线
],
```

删除 `value:'2'`（同步失败）与 `value:'3'`（同步中）。i18n 文案不动，对应字符串保留以兼容历史搜索 URL。

### R6b — groupId 接入 useDeviceGroups

**文件**：`omcmb/webcode/src/pages/device/DeviceList/index.tsx`

```typescript
const { data: groupList = [] } = useDeviceGroups();

const FILTER_FIELDS: FilterField[] = useMemo(() => [
  // ...
  {
    name: 'groupId',
    label: t('device.groupName'),
    type: 'multi-select',
    width: 160,
    options: groupList.flatMap((g) => [
      { label: g.name, value: g.id },
      ...(g.children ?? []).map((c) => ({ label: `${g.name} / ${c.name}`, value: c.id })),
    ]),
  },
  // ...
], [t, groupList]);
```

### R6c — 字典种子 + Filter 接入

#### R6c.1 Migration 000136

```sql
-- seed network_type / op_state / conn_status 字典数据
INSERT INTO sys_dictionaries (dict_code, dict_name, status, ...) VALUES
  ('network_type', '设备网络制式', 1, ...),
  ('op_state',     '设备激活状态', 1, ...),
  ('conn_status',  '设备在线状态', 1, ...);

INSERT INTO sys_dictionary_details (dict_id, label, value, sort_order, status) VALUES
  -- network_type
  ((SELECT id FROM sys_dictionaries WHERE dict_code='network_type'), 'eNB', 'eNB', 1, 1),
  ((SELECT id FROM sys_dictionaries WHERE dict_code='network_type'), 'gNB', 'gNB', 2, 1),
  -- op_state
  ((SELECT id FROM sys_dictionaries WHERE dict_code='op_state'), '激活', 'active', 1, 1),
  ((SELECT id FROM sys_dictionaries WHERE dict_code='op_state'), '未激活', 'inactive', 2, 1),
  -- conn_status (只放在线/离线，前端用本字典自动收敛)
  ((SELECT id FROM sys_dictionaries WHERE dict_code='conn_status'), '在线', '1', 1, 1),
  ((SELECT id FROM sys_dictionaries WHERE dict_code='conn_status'), '离线', '0', 2, 1)
ON CONFLICT DO NOTHING;
```

（实际 INSERT 列依目标表 schema 调整，需先读 dictionary 表 DDL）

#### R6c.2 Filter 接入字典

```typescript
const { data: networkTypeDict } = useDictionary('network_type');
const { data: opStateDict } = useDictionary('op_state');
const { data: connStatusDict } = useDictionary('conn_status');

const FILTER_FIELDS = useMemo(() => [
  {
    name: 'connStatus',
    label: t('device.connStatus'),
    type: 'multi-select',
    width: 160,
    options: (connStatusDict?.sysDictionaryDetails ?? []).map((d) => ({ label: d.label, value: d.value })),
  },
  {
    name: 'opState',
    label: t('device.opState'),
    type: 'multi-select',
    width: 160,
    options: (opStateDict?.sysDictionaryDetails ?? []).map((d) => ({ label: d.label, value: d.value })),
  },
  {
    name: 'networkType',
    label: t('device.networkType'),
    type: 'multi-select',
    width: 160,
    options: (networkTypeDict?.sysDictionaryDetails ?? []).map((d) => ({ label: d.label, value: d.value })),
  },
  // ...
], [t, connStatusDict, opStateDict, networkTypeDict, ...]);
```

### R6d — 当前筛选条件清单（数据源审计）

| 字段 | 数据源（**改造后**） |
|---|---|
| searchText | 文本输入（无下拉） |
| connStatus | 字典 `conn_status` |
| opState | 字典 `op_state` |
| networkType | 字典 `network_type` |
| productModel | 字典 `product_type`（已有） |
| groupId | `useDeviceGroups()` API |
| modelName / softwareVersion / firmwareVersion | 暂保留 `options:[]` 或 P2 接 device 聚合 API |

---

## 4. 实施顺序

1. ✅ 本设计文档落地
2. R1 修 L2 编辑（致命 bug，优先）
3. R6b groupId 动态选项 + R4 宽度对齐（同一个 useMemo，一并改）
4. R6a connStatus 收敛 + i18n（如需新增 label）
5. R3 导出按钮挪到 FilterBar
6. R6c 字典种子 migration（000136）+ Filter 接入
7. R5 端到端搜索验证 — 若 OK 不动；若 bug 补修
8. R2 E2E 断言 1 条
9. 校验：go build / typecheck / vitest

---

## 5. 风险与回滚

| 风险 | 缓解 |
|---|---|
| `useNameFilters.replace()` 方法不存在 | 若未暴露，本期一并新增；hook 内部 setFilters(newList) 即可 |
| L2 编辑切换 matchingMode 时 UI 字段联动 — 旧 fixture 可能匹配 deviceName 但目前 form 默认 'deviceName' | 已在 R1.2 显式根据 grp.matchingMode 设值 |
| 字典 seed 与现存 product_type / op_state 重复 | INSERT ... ON CONFLICT DO NOTHING；先读现存数据避免重复 |
| 字典加载失败 → 筛选下拉空 | useDictionary 已带 loading；UI 显示骨架 / 占位 |
| 设备分组列表巨大（> 1000 个）→ 下拉选项炸 | 短期 OK；后期改 server-side 搜索的 Select.Search mode |

---

## 6. 测试

| 类型 | 用例 |
|---|---|
| 前端单测 | `useGroupActions.handleSaveEditLevel2` 提交 payload 含 matching_mode/name_rule_list（mock updateGroupMutation） |
| 前端单测 | `useGroupActions.openEditLevel2` 回填 form + nameFilters（断言 setFieldsValue / replace 调用） |
| E2E | 改 L2 分组规则 → API 200 |
| E2E | GET /device/list?search=E2E,007219 → 200 |
| Vitest | DeviceList FILTER_FIELDS 在 connStatus 仅 2 项（online/offline） |
| Vitest | DeviceList FILTER_FIELDS 在 groupId 来自 useDeviceGroups 返回 |
| 手测 | UI 导出按钮在搜索按钮右侧 |

---

## 7. 关联资料

- 后端分组匹配引擎：`omcgo/internal/topology/group_match_engine.go`
- 多关键字搜索 helper：`omcgo/internal/device/search.go`
- 字典系统：`omcgo/internal/admin/dictionary_*.go` + frontend `useSystem.ts:399`
- FilterBar：`omcmb/webcode/src/components/FilterBar/index.tsx`
