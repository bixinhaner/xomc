# MML 用户私有模板 CRUD 完整化 + 用户级唯一性

> 状态：**待审核**
> 提出时间：2026-05-20
> 关联：`mml-custom-command-rename-and-fk-plan-20260520.md`（owner_user_id FK 升级三阶段）
> 涉及代码：`internal/mml/{model,handler,service,pg_repository,repository}.go` + `global/errors.go` + `omcmb/webcode/src/pages/mml/Console/components/{CommandTree,AddTemplateModal}.tsx` + `omcmb/frontend-core/src/{hooks/api/useMML,services/api/mmlApi}.ts` + i18n

---

## 1. 背景

`mml_custom_command` 表的 R-5 私有模板可见性已实现（T-0090-c），但完整 CRUD 仍有
三处缺口：

| 缺口 | 表现 | 来源 |
|---|---|---|
| **G1** Update 无权限校验 | `service.UpdateCustomCommand` 不校验调用者是否为创建者 → 任何登录用户可改任何模板 | 历史核查 §6 #1 |
| **G2** Delete 私有路径无校验 | `service.DeleteCustomCommand` 只挡"删别人的公有"；"删别人的私有"放行 | 历史核查 §6 #2 |
| **G3** 前端无 Edit/Delete UI | CommandTree 仅暴露 `+` 添加和叶子点击；无右键 / hover 编辑 / 删除入口 | 历史核查 §6 #5 |

同时，业务侧新增需求：**用户添加私有命令时，命令名称需要做用户级别唯一，每个用户的
私有命令名称不能重复**。

本方案一次性闭合 G1 + G2 + G3 + 用户级唯一性。

---

## 2. 决策清单

| # | 决策点 | 选定方案 | 理由 |
|---|---|---|---|
| D1 | 唯一性作用域 | **仅 private** `(owner_user_id, command_name) WHERE command_scope='private'` | 用户原话："私有命令名称不能重复"；public 跨用户允许同名（不同用户的"重启脚本"互不影响） |
| D2 | 唯一性载体 | DB 部分唯一索引 + service 层 pre-check | DB 索引兜底防 race；service pre-check 给友好的 409 错误而非裸 unique_violation |
| D3 | 大小写 | **大小写敏感**（exact match） | 与 PG 默认 collation 一致；不引入业务逻辑分歧 |
| D4 | NULL owner_user_id 历史行 | 不纳入索引（`WHERE owner_user_id IS NOT NULL`），保持兼容 | 000134 backfill 未命中的脏数据继续可用；新增行/已 backfill 行受约束 |
| D5 | Edit/Delete UI 暴露条件 | 仅当 `commandScope==='private'` 且 `(creator===self || isSuperAdmin)` 时显示 | 公有模板编辑/删除走 D21（admin + 创建者），本期不实现，保持现状不退化 |
| D6 | Edit 是否允许改 scope（private ⇄ public） | **不允许**，scope 在创建时固定 | 避免"私有 → 公有"流转引出二次合规审查；要换 scope 用"复制为公共" |
| D7 | Edit 是否允许改 command_name | **允许**，但触发 D1 唯一性校验 | 用户可能拼写错误想改名；保留灵活性 |
| D8 | Update/Delete 鉴权语义 | `(owner == currentUserID) OR isSuperAdmin` | 与 List 可见性的 RBAC group-share 不同 — 编辑/删除是写操作，仅创建者或超管 |
| D9 | 越权错误码 | 403 `ErrCodeForbidden`（已存在） | 复用，不新增 |
| D10 | 重名错误码 | 409 `ErrCodeTemplateNameDuplicated`（新增） | 与 HTTP 409 Conflict 语义一致 |
| D11 | 现网已存在的同名私有行 | 迁移前置 dedup：保留 `created_at` 最早一条，其他自动 rename `名字 (重名-N)` | 0 数据丢失；用户可后续手动重命名 |
| D12 | UI 入口形态 | CommandTree 叶子节点 hover 时显示右侧 `[✏️] [🗑️]` 图标；Tree 不挑节点高度，icon 用 antd 12px size | 与既有"+" 按钮风格一致；不引入右键菜单（移动端不友好） |
| D13 | AddTemplateModal 编辑模式 | 复用同一组件，新增 `editingTemplate?: MMLCustomCommand` prop；标题切换为"编辑私有模板"；scope 字段在编辑时隐藏 | 一处维护，避免代码重复 |

---

## 3. 数据模型变更

### 3.1 Migration 000135

文件：`omcgo/migrations/000135_mml_custom_command_private_name_unique.sql`

```sql
-- +goose Up

-- Step 1: 预 dedup — 同 owner_user_id + command_name 私有行若有重复，
--         按 created_at 升序保留第一条；其余自动 rename 为 "原名 (重名-N)"
-- +goose StatementBegin
DO $$
DECLARE
    dup RECORD;
BEGIN
    FOR dup IN
        SELECT id, command_name, owner_user_id, created_at,
               ROW_NUMBER() OVER (PARTITION BY owner_user_id, command_name ORDER BY created_at) AS rn
        FROM mml_custom_command
        WHERE command_scope = 'private' AND owner_user_id IS NOT NULL
    LOOP
        IF dup.rn > 1 THEN
            UPDATE mml_custom_command
               SET command_name = dup.command_name || ' (重名-' || dup.rn || ')'
             WHERE id = dup.id;
        END IF;
    END LOOP;
END $$;
-- +goose StatementEnd

-- Step 2: 部分唯一索引 — 仅约束 private + 已绑定 owner 的行
CREATE UNIQUE INDEX IF NOT EXISTS uq_mml_custom_command_private_name_per_owner
    ON mml_custom_command(owner_user_id, command_name)
    WHERE command_scope = 'private' AND owner_user_id IS NOT NULL;

COMMENT ON INDEX uq_mml_custom_command_private_name_per_owner IS
    '用户级唯一性：每个用户的私有模板 command_name 不重复（public 跨用户允许同名；owner_user_id NULL 的历史脏数据豁免）';

-- +goose Down
DROP INDEX IF EXISTS uq_mml_custom_command_private_name_per_owner;
```

### 3.2 影响面

- 写路径：Create / Update 任何会写入 `(owner_user_id, command_name) where private` 的操作都会经过此索引检查
- 索引大小：每 私有模板行约 32B（uuid + name 平均 50 字符），10^4 行 ≈ 数 MB，可忽略
- 老 catalog admin 行（owner_user_id 为 NULL）不受约束 — 与"用户级唯一"语义吻合（系统级公共条目不属于任何用户的私有命名空间）

---

## 4. 后端契约

### 4.1 错误码

`omcgo/global/errors.go` 新增：

```go
ErrCodeTemplateNameDuplicated = 17008  // 用户已有同名私有模板
```

HTTP 状态映射：→ `http.StatusConflict` (409)

### 4.2 Repository 接口扩展

```go
// CustomCommandRepository
type CustomCommandRepository interface {
    Create(ctx, *MMLCustomCommand) error
    GetByID(ctx, uuid.UUID) (*MMLCustomCommand, error)
    Update(ctx, *MMLCustomCommand) error
    Delete(ctx, uuid.UUID) error
    List(ctx, CustomCommandFilter) (*model.ListResponse[MMLCustomCommand], error)

    // NEW: 同 owner 私有命名空间内是否已存在该 name；excludeID 用于 Update 排除自身。
    //      只在 (command_scope='private' AND owner_user_id=ownerID) 子集内匹配。
    NameExistsForPrivate(ctx context.Context, ownerID uuid.UUID, name string, excludeID *uuid.UUID) (bool, error)
}
```

SQL：

```sql
SELECT EXISTS(
    SELECT 1 FROM mml_custom_command
    WHERE owner_user_id = $1
      AND command_name  = $2
      AND command_scope = 'private'
      AND ($3::uuid IS NULL OR id <> $3)
)
```

### 4.3 Service 行为

| 方法 | 新签名 | 关键校验顺序 |
|---|---|---|
| `CreateCustomCommand(ctx, cmd)` | 不变 | ① scope=='private' 且 OwnerUserID 非空 → NameExistsForPrivate ？ 409 ② Create |
| `UpdateCustomCommand(ctx, id, cmd, currentUserID, isSuperAdmin)` | **签名变更** | ① GetByID ② ownership 校验（非 owner 且非 super → 403） ③ 若改名且 scope=='private'，NameExistsForPrivate ④ Update |
| `DeleteCustomCommand(ctx, id, currentUserID, isSuperAdmin)` | **签名变更**（去 username 参数） | ① GetByID ② ownership 校验（同上）③ Delete |
| `CloneCustomCommand(ctx, id, currentUser, ownerID)` | 不变（000134 已就绪） | ① GetByID ② new clone：name = source.name + " (副本)"，scope='private'，owner=current ③ Create（**会触发唯一性校验**：若副本名已存在则 409，用户改名后重试） |

**ownership 校验函数**：

```go
// isOwnerOrSuper 兼容 Phase 1（owner_user_id 可能为 NULL）：
//   - 优先比对 owner_user_id (uuid)；非空且不等 → 拒绝
//   - 兼容兜底：owner_user_id 为 NULL 时回退比对 creator (username)
//   - super_admin 始终放行
func isOwnerOrSuper(cmd *MMLCustomCommand, currentUserID uuid.UUID, currentUsername string, isSuperAdmin bool) bool {
    if isSuperAdmin {
        return true
    }
    if cmd.OwnerUserID != nil && *cmd.OwnerUserID != uuid.Nil {
        return *cmd.OwnerUserID == currentUserID
    }
    // legacy NULL owner_user_id 行（000134 backfill 未命中）兜底走 username
    return cmd.Creator != "" && cmd.Creator == currentUsername
}
```

### 4.4 Handler 改动

| Handler | 新增提取 | 调用 |
|---|---|---|
| `CreateTemplate` | `user_id` (uuid) | 注入到 `MMLCustomCommand.OwnerUserID`（已规划于 mml-custom-command-rename-and-fk-plan §3.1） |
| `UpdateTemplate` | `user_id` + `is_super_admin` | `service.UpdateCustomCommand(ctx, id, tmpl, userID, isSuper)` |
| `DeleteTemplate` | `user_id` + `is_super_admin` | `service.DeleteCustomCommand(ctx, id, userID, isSuper)`（删 `username` 参数） |
| `CloneTemplate` | `user_id` | `service.CloneCustomCommand(ctx, id, username, &userID)` |

### 4.5 API 响应体

`PUT /api/v1/mml/templates/:id` 错误体（统一 `commonerrors.AbortWithError`）：

```jsonc
// 409 Conflict — 同 owner 已有同名私有模板
{
  "code": 17008,
  "message": "您已有同名的私有模板，请换个名字",
  "details": { "name": "重启脚本" }
}

// 403 Forbidden — 非创建者且非超管
{
  "code": 1003,    // ErrForbidden 既有
  "message": "您没有权限修改此模板"
}
```

---

## 5. 前端 UX

### 5.1 CommandTree 叶子 hover 操作

`omcmb/webcode/src/pages/mml/Console/components/CommandTree.tsx`

```typescript
// renderLeaf 在 buildCustomTreeData 内：
const canEdit = isSuperAdmin || cc.creator === currentUsername;
// 仅在 private 且具备权限时挂图标；public 暂不暴露编辑入口（D5）
const showActions = canEdit && cc.commandScope === 'private';

const renderLeaf = (cc: MMLCustomCommand): TreeDataNode => ({
  key: `${CUSTOM_KEY_PREFIX}${cc.id}`,
  title: (
    <span className="custom-leaf">
      {renderOpLeafTitle(cc.operationType, cc.commandName)}
      {showActions && (
        <span className="custom-leaf-actions">
          <EditOutlined onClick={(e) => { e.stopPropagation(); setEditingTemplate(cc); }} />
          <Popconfirm
            title={t('mml.template.deleteConfirm', { name: cc.commandName })}
            onConfirm={(e) => { e?.stopPropagation(); deleteMutation.mutate(cc.id); }}
          >
            <DeleteOutlined onClick={(e) => e.stopPropagation()} />
          </Popconfirm>
        </span>
      )}
    </span>
  ),
  isLeaf: true,
});
```

CSS：`.custom-leaf-actions` 默认 `opacity: 0`，`.custom-leaf:hover .custom-leaf-actions { opacity: 1 }`，过渡 0.15s。

### 5.2 AddTemplateModal 编辑模式

`omcmb/webcode/src/pages/mml/Console/components/AddTemplateModal.tsx`

```typescript
interface AddTemplateModalProps {
  open: boolean;
  scope: 'public' | 'private';
  editingTemplate?: MMLCustomCommand | null;  // NEW
  onClose: () => void;
  onSuccess: () => void;
}

// 内部派生 isEdit
const isEdit = Boolean(editingTemplate);
const title = isEdit ? t('mml.template.editTitle') : t('mml.template.addTitle');

// 初始值
useEffect(() => {
  if (open) {
    if (editingTemplate) {
      form.setFieldsValue({
        templateName: editingTemplate.commandName,
        commandCode: editingTemplate.commandCode,
        operationType: editingTemplate.operationType,
        modifyValues: serializeParams(editingTemplate.parameters),
        description: editingTemplate.description ?? '',
      });
    } else {
      form.resetFields();
    }
  }
}, [open, editingTemplate]);

// 提交分支
if (isEdit) {
  await updateMutation.mutateAsync({ id: editingTemplate.id, data: payload });
} else {
  await createMutation.mutateAsync(payload);
}

// scope 在编辑模式隐藏（D6）
```

### 5.3 错误处理

```typescript
const handleError = (err: unknown) => {
  if (err instanceof AxiosError && err.response) {
    const { code, message: msg } = err.response.data ?? {};
    if (code === 17008) {
      void message.error(t('mml.template.error.nameDuplicated'));
      return;
    }
    if (code === 1003) {
      void message.error(t('mml.template.error.notOwner'));
      return;
    }
  }
  void message.error(t('common.operationFailed'));
};
```

### 5.4 i18n keys

| key | zh-CN | en-US |
|---|---|---|
| `mml.template.addTitle` | 新建模板 | New Template |
| `mml.template.editTitle` | 编辑模板 | Edit Template |
| `mml.template.action.edit` | 编辑 | Edit |
| `mml.template.action.delete` | 删除 | Delete |
| `mml.template.deleteConfirm` | 确认删除模板「{name}」？删除后无法恢复 | Delete template "{name}"? This cannot be undone |
| `mml.template.error.nameDuplicated` | 您已有同名的私有模板，请换个名字 | A private template with this name already exists |
| `mml.template.error.notOwner` | 您没有权限修改 / 删除此模板 | You don't have permission to modify / delete this template |
| `mml.template.deleted` | 模板已删除 | Template deleted |
| `mml.template.updated` | 模板已更新 | Template updated |

---

## 6. 测试

### 6.1 单元测试

- `service_test.go`：
  - `TestService_CreateCustomCommand_PrivateNameDuplicate` — 同 owner 同名 → 409
  - `TestService_CreateCustomCommand_PrivateNameAcrossOwners_Allowed` — owner A / owner B 同名 OK
  - `TestService_CreateCustomCommand_PublicSameNameAllowed` — public scope 不受约束
  - `TestService_UpdateCustomCommand_NonOwner_Forbidden`
  - `TestService_UpdateCustomCommand_OwnerSelf_OK`
  - `TestService_UpdateCustomCommand_SuperAdmin_BypassOwner`
  - `TestService_UpdateCustomCommand_RenameToSelf_Allowed` — 没改名也通过校验
  - `TestService_UpdateCustomCommand_RenameToOtherExisting_409`
  - `TestService_DeleteCustomCommand_PrivateNonOwner_Forbidden`（**修复 G2**）
  - `TestService_DeleteCustomCommand_PrivateOwner_OK`
  - `TestService_DeleteCustomCommand_SuperAdmin_OK`

### 6.2 E2E

`scripts/e2e_verify.sh` 新增：

```bash
claim "mml templates: PUT 越权 → 403"
claim "mml templates: DELETE 越权 → 403"
claim "mml templates: POST 私有同名 → 409"
claim "mml templates: PUT 改名撞同 owner 已有私有名 → 409"
```

---

## 7. 兼容与回滚

| 维度 | 行为 |
|---|---|
| 现网已有同名私有行 | Migration §3.1 Step 1 自动 rename，保留所有数据 |
| owner_user_id NULL 行 | 不受新索引约束（partial WHERE）；现状不变 |
| 旧 API 客户端调 DELETE 不带 user_id | Phase 1 `RequireAuth` 中间件保证 user_id 一定存在；缺失视为 401 |
| 回滚 | `goose down` 仅 DROP 索引；rename 过的 command_name 不自动还原（用户可手动恢复） |
| Update 签名变更 | 调用者仅 handler 一处；同 PR 内同步更新；service mock 也需更新 |
| Delete 签名变更 | 同上 |

---

## 8. 实施顺序（本 PR）

1. ✅ 本设计文档落地
2. Migration 000135 — 唯一索引 + dedup
3. `global/errors.go` — `ErrCodeTemplateNameDuplicated = 17008`
4. `internal/mml/model.go` — `MMLCustomCommand.OwnerUserID *uuid.UUID`（重做 000134 的 Go side dual-write）
5. `internal/mml/repository.go` 接口 + `pg_repository.go` 实现 `NameExistsForPrivate` + INSERT/scan owner_user_id
6. `internal/mml/service.go` — Create/Update/Delete 完整鉴权 + 唯一性
7. `internal/mml/handler.go` — Create/Update/Delete/Clone 透传 user_id + is_super_admin
8. 单测：`service_test.go` 增补 9 个 case
9. 前端：useMML.ts 确认（已存在）、AddTemplateModal edit mode、CommandTree hover icons + Popconfirm
10. i18n（zh-CN + en-US）
11. E2E 4 条
12. 校验：`go build` / `go test -race` / `npm typecheck` / `vitest`

---

## 9. 关联文档

- [`mml-custom-command-rename-and-fk-plan-20260520.md`](./mml-custom-command-rename-and-fk-plan-20260520.md) — owner_user_id FK 三阶段计划（本 PR 承载 Phase 2 的"读路径权限校验"部分）
- [`mml-console-cmcc-tdlte-v23-adjustment-plan-20260519.md`](./mml-console-cmcc-tdlte-v23-adjustment-plan-20260519.md) §6.5 — R-5 Customized 可见性原始需求
