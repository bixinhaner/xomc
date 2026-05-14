import { describe, it, expect, beforeEach } from 'vitest';
import {
  renderStatementLocal,
  renderStatementsLocal,
  useMmlConsoleStore,
} from '../mmlConsoleStore';
import type { Statement, SubFieldDef } from '../../types/mmlConsole';

// ============================================================
// T-0123-P2-a Console store 单测
//
// 重点 1：renderStatementLocal 是 store 双向同步的本地 render 实现，
//        必须与后端 internal/mml/mml_renderer.go::RenderStatement 行为
//        一致；这里覆盖 LST/MOD/ADD/RMV 4 op + sort_order + 特殊字符。
// 重点 2：store actions 触发后 mmlText 应跟随更新，syncSource 标记正确。
// ============================================================

function makeSubField(overrides: Partial<SubFieldDef> = {}): SubFieldDef {
  return {
    id: 'sf-' + Math.random(),
    commandId: 'cmd-1',
    paramId: 'p-1',
    mmlCode: 'FIELD',
    label: 'Field',
    labelI18n: {},
    tr069Path: 'Device.X',
    valueType: 'string',
    accessType: 'READ_WRITE',
    isObject: false,
    supportsAdd: false,
    supportsDelete: false,
    changeApplies: 'Immediate',
    constraintText: '',
    constraintTextI18n: {},
    defaultSelected: false,
    isRequired: false,
    sortOrder: 0,
    ...overrides,
  };
}

function makeStmt(overrides: Partial<Statement> = {}): Statement {
  return {
    uid: 'u-' + Math.random(),
    commandId: 'cmd-1',
    commandCode: 'LST_DEVICE_INFO',
    logicalCode: 'DEVICE_INFO',
    operationType: 'LST',
    logicalNameI18n: {},
    subFields: [],
    selectedSubFieldIds: [],
    values: {},
    unknownCodes: [],
    ...overrides,
  };
}

// ============================================================
// renderStatementLocal — 4 op × 边界
// ============================================================

describe('renderStatementLocal · LST', () => {
  it('空 selected → 裸 op', () => {
    const s = makeStmt({ operationType: 'LST', selectedSubFieldIds: [] });
    expect(renderStatementLocal(s)).toBe('LST DEVICE_INFO');
  });

  it('单选 → :lstId={MMLCODE}', () => {
    const sf = makeSubField({ id: 'sf1', mmlCode: 'MODEL', sortOrder: 1 });
    const s = makeStmt({
      operationType: 'LST',
      subFields: [sf],
      selectedSubFieldIds: ['sf1'],
    });
    expect(renderStatementLocal(s)).toBe('LST DEVICE_INFO:lstId={MODEL}');
  });

  it('多选按 sortOrder 排序', () => {
    const sfA = makeSubField({ id: 'a', mmlCode: 'ZZZ', sortOrder: 1 });
    const sfB = makeSubField({ id: 'b', mmlCode: 'AAA', sortOrder: 2 });
    const sfC = makeSubField({ id: 'c', mmlCode: 'MMM', sortOrder: 3 });
    const s = makeStmt({
      operationType: 'LST',
      subFields: [sfA, sfB, sfC],
      selectedSubFieldIds: ['c', 'a', 'b'], // 乱序入参
    });
    expect(renderStatementLocal(s)).toBe('LST DEVICE_INFO:lstId={ZZZ,AAA,MMM}');
  });

  it('同 sortOrder 按 mml_code 字典序', () => {
    const sfA = makeSubField({ id: 'a', mmlCode: 'BBB', sortOrder: 5 });
    const sfB = makeSubField({ id: 'b', mmlCode: 'AAA', sortOrder: 5 });
    const s = makeStmt({
      operationType: 'LST',
      subFields: [sfA, sfB],
      selectedSubFieldIds: ['a', 'b'],
    });
    expect(renderStatementLocal(s)).toBe('LST DEVICE_INFO:lstId={AAA,BBB}');
  });
});

describe('renderStatementLocal · MOD', () => {
  it('空 values → 裸 op', () => {
    const s = makeStmt({ operationType: 'MOD', commandCode: 'MOD_DEVICE_INFO' });
    expect(renderStatementLocal(s)).toBe('MOD DEVICE_INFO');
  });

  it('单字段 → :K=V', () => {
    const sf = makeSubField({ id: 'sf1', mmlCode: 'Name' });
    const s = makeStmt({
      operationType: 'MOD',
      subFields: [sf],
      values: { Name: 'alice' },
    });
    expect(renderStatementLocal(s)).toBe('MOD DEVICE_INFO:Name=alice');
  });

  it('多字段按 sortOrder 排序', () => {
    const sfA = makeSubField({ id: 'a', mmlCode: 'Name', sortOrder: 1 });
    const sfB = makeSubField({ id: 'b', mmlCode: 'Email', sortOrder: 2 });
    const s = makeStmt({
      operationType: 'MOD',
      subFields: [sfA, sfB],
      values: { Email: 'a@b', Name: 'alice' },
    });
    expect(renderStatementLocal(s)).toBe('MOD DEVICE_INFO:Name=alice,Email=a@b');
  });

  it('未在 subFields 的 key 按字典序追加在末尾', () => {
    const sfA = makeSubField({ id: 'a', mmlCode: 'Name', sortOrder: 1 });
    const s = makeStmt({
      operationType: 'MOD',
      subFields: [sfA],
      values: { Name: 'alice', Zeta: 'last', Alpha: 'first' },
    });
    expect(renderStatementLocal(s)).toBe('MOD DEVICE_INFO:Name=alice,Alpha=first,Zeta=last');
  });

  it('特殊字符值双引号包裹', () => {
    const sf = makeSubField({ id: 'sf1', mmlCode: 'Desc' });
    const s = makeStmt({
      operationType: 'MOD',
      subFields: [sf],
      values: { Desc: 'has, comma' },
    });
    expect(renderStatementLocal(s)).toBe('MOD DEVICE_INFO:Desc="has, comma"');
  });

  it('值含双引号转义为 \\"', () => {
    const sf = makeSubField({ id: 'sf1', mmlCode: 'X' });
    const s = makeStmt({
      operationType: 'MOD',
      subFields: [sf],
      values: { X: 'say "hi"' },
    });
    expect(renderStatementLocal(s)).toBe('MOD DEVICE_INFO:X="say \\"hi\\""');
  });
});

describe('renderStatementLocal · ADD', () => {
  it('与 MOD 同语法', () => {
    const sf = makeSubField({ id: 'sf1', mmlCode: 'Name' });
    const s = makeStmt({
      operationType: 'ADD',
      commandCode: 'ADD_USER',
      logicalCode: 'USER',
      subFields: [sf],
      values: { Name: 'bob' },
    });
    expect(renderStatementLocal(s)).toBe('ADD USER:Name=bob');
  });
});

describe('renderStatementLocal · RMV', () => {
  it('无 index → 裸 op', () => {
    const s = makeStmt({
      operationType: 'RMV',
      commandCode: 'RMV_USER',
      logicalCode: 'USER',
    });
    expect(renderStatementLocal(s)).toBe('RMV USER');
  });

  it('有 index → :Index=N', () => {
    const s = makeStmt({
      operationType: 'RMV',
      logicalCode: 'USER',
      rmvInstanceIndex: 5,
    });
    expect(renderStatementLocal(s)).toBe('RMV USER:Index=5');
  });

  it('index=0 也应输出（合法实例号）', () => {
    const s = makeStmt({
      operationType: 'RMV',
      logicalCode: 'USER',
      rmvInstanceIndex: 0,
    });
    expect(renderStatementLocal(s)).toBe('RMV USER:Index=0');
  });
});

// ============================================================
// renderStatementsLocal — 多 statement 拼接
// ============================================================

describe('renderStatementsLocal', () => {
  it('空数组 → 空字符串', () => {
    expect(renderStatementsLocal([])).toBe('');
  });

  it('多条用 ; 分隔 + 末尾 ;', () => {
    const s1 = makeStmt({ logicalCode: 'A', operationType: 'LST' });
    const s2 = makeStmt({ logicalCode: 'B', operationType: 'MOD' });
    expect(renderStatementsLocal([s1, s2])).toBe('LST A;MOD B;');
  });
});

// ============================================================
// useMmlConsoleStore — 关键 actions
// ============================================================

describe('useMmlConsoleStore actions', () => {
  beforeEach(() => {
    useMmlConsoleStore.getState().reset();
  });

  it('appendStatement 后 mmlText 自动更新 + syncSource=ui', () => {
    const s = makeStmt({ logicalCode: 'DEVICE_INFO', operationType: 'LST' });
    useMmlConsoleStore.getState().appendStatement(s);
    const state = useMmlConsoleStore.getState();
    expect(state.statements).toHaveLength(1);
    expect(state.mmlText).toBe('LST DEVICE_INFO;');
    expect(state.syncSource).toBe('ui');
    expect(state.activeStatementUid).toBe(s.uid);
  });

  it('removeStatement 后 mmlText 收缩', () => {
    const s1 = makeStmt({ uid: 'u1', logicalCode: 'A' });
    const s2 = makeStmt({ uid: 'u2', logicalCode: 'B' });
    useMmlConsoleStore.getState().appendStatement(s1);
    useMmlConsoleStore.getState().appendStatement(s2);
    expect(useMmlConsoleStore.getState().mmlText).toBe('LST A;LST B;');

    useMmlConsoleStore.getState().removeStatement('u1');
    expect(useMmlConsoleStore.getState().mmlText).toBe('LST B;');
  });

  it('toggleSubField 切换选中 + mmlText 同步', () => {
    const sf = makeSubField({ id: 'sf1', mmlCode: 'X' });
    const s = makeStmt({
      uid: 'u1',
      logicalCode: 'CMD',
      operationType: 'LST',
      subFields: [sf],
    });
    useMmlConsoleStore.getState().appendStatement(s);

    useMmlConsoleStore.getState().toggleSubField('u1', 'sf1');
    expect(useMmlConsoleStore.getState().mmlText).toBe('LST CMD:lstId={X};');

    useMmlConsoleStore.getState().toggleSubField('u1', 'sf1');
    expect(useMmlConsoleStore.getState().mmlText).toBe('LST CMD;');
  });

  it('setValue 写入后 mmlText 同步；空字符串删 key', () => {
    const sf = makeSubField({ id: 'sf1', mmlCode: 'Name' });
    const s = makeStmt({
      uid: 'u1',
      operationType: 'MOD',
      logicalCode: 'USER',
      subFields: [sf],
    });
    useMmlConsoleStore.getState().appendStatement(s);

    useMmlConsoleStore.getState().setValue('u1', 'Name', 'alice');
    expect(useMmlConsoleStore.getState().mmlText).toBe('MOD USER:Name=alice;');

    useMmlConsoleStore.getState().setValue('u1', 'Name', '');
    expect(useMmlConsoleStore.getState().mmlText).toBe('MOD USER;');
  });

  it('setMmlText 标记 syncSource=text 不动 statements', () => {
    const s = makeStmt({ uid: 'u1', logicalCode: 'DEVICE_INFO' });
    useMmlConsoleStore.getState().appendStatement(s);
    const beforeStmts = useMmlConsoleStore.getState().statements;

    useMmlConsoleStore.getState().setMmlText('LST OTHER;');
    const state = useMmlConsoleStore.getState();
    expect(state.mmlText).toBe('LST OTHER;');
    expect(state.syncSource).toBe('text');
    expect(state.statements).toBe(beforeStmts); // 引用未变 — 未触发 parse
  });

  it('reset 清空 statements + mmlText + 取消 debounce timer', () => {
    const s = makeStmt({ logicalCode: 'A' });
    useMmlConsoleStore.getState().appendStatement(s);
    useMmlConsoleStore.getState().reset();
    const state = useMmlConsoleStore.getState();
    expect(state.statements).toEqual([]);
    expect(state.mmlText).toBe('');
    expect(state.syncSource).toBe('none');
  });
});
