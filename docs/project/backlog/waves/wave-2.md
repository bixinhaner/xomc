# Wave 2 · 收尾冲刺（已完结 2026-06-22，12/13 ✅）

> 从 `docs/project/backlog.md` §3 Wave 整改执行队列拆出（2026-05-07，纯搬运）。
> 历史快照，不再更新；仅 W2.A.3（短信凭据，T-0014 deps T-0009 外部）未达成。

#### 🟡 Wave 2 · 收尾冲刺（W3-W8，2026-05-12 ~ 2026-06-22）

**Block A · F04 通知三通道（W3-W4）**

| 序 | Task | 标题 | 章程 |
|----|------|------|------|
| A.1 | T-0007 | F04 邮件通道（EmailDispatcher + dispatchEmail + DI + migration 000040） | ✅ W2.A.1 2026-04-28 |
| A.2 | T-0014 | F04 短信通道（依赖 T-0009） | W2.A.3 |
| A.3 | T-0011 | F04 Webhook 通道（retry/dead-letter/HMAC + FilterEngine 接 AlarmEngine.Process） | ✅ W2.A.2 2026-04-28 |
| A.4 | T-0009 | 短信凭据申请（外部动作） | — |
| A.5 | T-0043 | 通知模板 + 历史记录（API + UI） | W2.A.4 |
| A.6 | T-0044 | notification/ 测试覆盖率 ≥ 70% | W2.A.5 |

**Block B · 测试债清零（W5-W6）**

| 序 | Task | 标题 | 章程 |
|----|------|------|------|
| B.1 | T-0045 | task/ 测试覆盖率 21.3% → 75.3% (CWMP/Reboot/Completion 三关键测试 PASS) | ✅ W2.B.1 2026-04-28 |
| B.2 | T-0046 | events/ 测试 0% → 87.6% + EventService facade（39 测） | ✅ W2.B.2 2026-04-28 |
| B.3 | T-0047 | core/ 覆盖率 43.2% → 55.7% (carrier/cmcc/ctcc/cucc 0% → 91-97%) | ✅ W2.B.3 2026-04-28 |
| B.4a | T-0048 | mr/ thin service facade + 11 测 | ✅ W2.B.4 2026-04-28 |
| B.4b | T-0049 | syslog/ thin service facade + 15 测 | ✅ W2.B.4 2026-04-28 |
| B.4c | T-0050 | provision/ facade 包既有 engine/orchestrator + 13 测 | ✅ W2.B.4 2026-04-28 |
| B.4d | T-0051 | interop/ facade 包既有 runner/validator + 12 测 | ✅ W2.B.4 2026-04-28 |

**Block C · 前端整改（W7）**

| 序 | Task | 标题 | 章程 |
|----|------|------|------|
| C.1 | T-0052 | frontend-core hooks/services 对齐（24 → 28 hooks, gap 5 → 1） | ✅ W2.C.1 2026-04-28 |
| C.2 | T-0053 | 前端 any 清零（20 → 0） | ✅ W2.C.2 (a) 2026-04-28 |
| C.3 | T-0054 | DeviceGrouping 拆分（max 650 → 303 行 / 6 子组件 + 6 hooks） | ✅ W2.C.2 (b) 2026-04-28 |
| C.4 | T-0055 | 前端 vitest 0% → lines 66.66% / statements 54.7% | ✅ W2.C.3 2026-04-28 |

**Block D · E2E ≥ 100（W8）**：T-0006 累计目标 @≥100（章程 W2.D.1）

**Wave 2 退出（2026-06-22 对峙）**：≥ 7/10 ✅ → 进 Wave 3

---

← 返回 [`docs/project/backlog.md`](../../backlog.md)
