// gen_menu_buttons.mjs — 为缺操作按钮的菜单批量补 4 按钮（查询/添加/修改/删除）。
//
// 用法：
//   1. 把当前 DB 里没按钮的 menu 列表 dump 到 /tmp/menus_no_buttons.txt：
//      docker exec docker-postgres-1 psql -U omcgo -d omcgo -t -A -F "|" \
//        -c "SELECT id, permission_key FROM menus WHERE type='menu' AND status='normal' \
//            AND NOT EXISTS (SELECT 1 FROM menus c WHERE c.parent_id = menus.id \
//            AND c.type='button' AND c.status='normal') ORDER BY permission_key;" \
//        > /tmp/menus_no_buttons.txt
//   2. node omcgo/scripts/gen_menu_buttons.mjs
//   3. 输出 SQL 在 /tmp/059.sql；按下一个迁移版本号挪到 omcgo/migrations/seed/。
//
// 设计：每个菜单插入 4 行，permission_key = <父 key>:<action>，幂等（WHERE NOT EXISTS）。
import { readFileSync, writeFileSync } from 'node:fs';

// 4 个标准按钮，与现有 device:list / system:user 等示例对齐。
const STD_BUTTONS = [
  { action: 'query', name: '查询', sort: 1 },
  { action: 'add',   name: '添加', sort: 2 },
  { action: 'edit',  name: '修改', sort: 3 },
  { action: 'delete', name: '删除', sort: 4 },
];

const lines = readFileSync('/tmp/menus_no_buttons.txt', 'utf-8').split('\n').filter(Boolean);
const out = [];
out.push('-- +goose Up');
out.push('-- 为 29 个尚无操作按钮的菜单补全标准 4 按钮（查询/添加/修改/删除）。');
out.push("-- 命名规范：name='查询'/'添加'/'修改'/'删除'，permission_key=<父 menu key>:<action>。");
out.push("-- 幂等：用 WHERE NOT EXISTS 跳过已存在按钮（按 parent_id + name 唯一）。");
out.push("-- 需要更细的按钮（如 导出 / 同步 / 强制下线 等）由 admin 在 /system/menus 页面手工追加。");
out.push('');

for (const line of lines) {
  const [id, key] = line.split('|');
  for (const b of STD_BUTTONS) {
    out.push(`INSERT INTO menus (parent_id, type, name, permission_key, sort_order, show_status, status)
SELECT '${id}', 'button', '${b.name}', '${key}:${b.action}', ${b.sort}, 'show', 'normal'
WHERE NOT EXISTS (
    SELECT 1 FROM menus
    WHERE parent_id = '${id}' AND name = '${b.name}' AND status = 'normal'
);`);
  }
  out.push('');
}

out.push('-- +goose Down');
out.push('-- 回滚：删除本次脚本插入的 4 类按钮（按 permission_key 后缀精确匹配，避免误删手工按钮）。');
out.push("DELETE FROM menus WHERE type = 'button' AND permission_key IN (");
const pks = [];
for (const line of lines) {
  const [, key] = line.split('|');
  for (const b of STD_BUTTONS) pks.push(`${key}:${b.action}`);
}
for (let i = 0; i < pks.length; i++) {
  out.push(`    '${pks[i]}'${i < pks.length - 1 ? ',' : ''}`);
}
out.push(');');

writeFileSync('/tmp/059.sql', out.join('\n') + '\n');
console.error(`generated ${lines.length} menus × 4 buttons = ${lines.length * 4} INSERT rows`);
