#!/usr/bin/env node
// =============================================================================
// skin-parity.mjs —— 三皮肤路由/菜单对齐守卫（v1 = 唯一标准）
// =============================================================================
// 背景：webcode(v1, Antd5) 是主皮肤/唯一标准；webcode-v2(shadcn) / webcode-v3(HUD)
// 必须与 v1 的「可路由路径集合」「可见菜单项集合」完全一致，只允许外观不同。
// 本守卫从 v1 源码抽出 canonical spec，断言 v2/v3 一致，不一致即 exit 1。
//
// 校验两个不可变量（machine-checked）：
//   1) 路由集合：skin 的全部声明 path == v1 routes.tsx 的全部声明 path
//   2) 可见菜单：skin 的非 hidden path == v1 navConfig.ts 的可见（非注释）path
// 分组/排版（v1 的扁平 NavGroup vs v2/v3 的 section→module）属各皮肤 shell 范式，
// 不做字节级强制——但「显示哪些叶子项」必须一致（上面第 2 条）。
//
// 用法： node omcmb/scripts/skin-parity.mjs        （从 omcmb/ 或仓库根均可）
// =============================================================================
import { readFileSync } from 'node:fs';
import { fileURLToPath } from 'node:url';
import { dirname, resolve } from 'node:path';

const HERE = dirname(fileURLToPath(import.meta.url));
const OMCMB = resolve(HERE, '..');

const V1_ROUTES = resolve(OMCMB, 'webcode/src/router/routes.tsx');
const V1_NAV = resolve(OMCMB, 'webcode/src/components/Layout/Sidebar/navConfig.ts');
const SKINS = {
  'webcode-v2': resolve(OMCMB, 'webcode-v2/src/router/navConfig.tsx'),
  'webcode-v3': resolve(OMCMB, 'webcode-v3/src/router/navConfig.tsx'),
};

// 非业务路由：登录、403、通配、父路由 —— 不参与对齐
const IGNORED = new Set(['/login', '/403', '403', '*', '/', '/*']);

const PATH_RE = /path:\s*['"]([^'"]+)['"]/;

function norm(p) {
  if (!p) return p;
  return p.startsWith('/') ? p : `/${p}`;
}
function isLineComment(line) {
  return /^\s*\/\//.test(line);
}

// v1 canonical：routes.tsx 里所有声明的 path（含隐藏但可达的）
function v1CanonicalRoutes() {
  const set = new Set();
  for (const line of readFileSync(V1_ROUTES, 'utf8').split('\n')) {
    const m = line.match(PATH_RE);
    if (!m) continue;
    const p = norm(m[1]);
    if (!IGNORED.has(p)) set.add(p);
  }
  return set;
}

// v1 可见菜单：navConfig.ts 中非注释行的 path（注释掉的 = 隐藏）
function v1VisibleRoutes() {
  const set = new Set();
  for (const line of readFileSync(V1_NAV, 'utf8').split('\n')) {
    if (isLineComment(line)) continue;
    const m = line.match(PATH_RE);
    if (!m) continue;
    const p = norm(m[1]);
    if (!IGNORED.has(p)) set.add(p);
  }
  return set;
}

// skin (v2/v3)：每个 RouteDef 一行，hidden:true 表示不在菜单显示
function skinRoutes(file) {
  const all = new Set();
  const visible = new Set();
  for (const line of readFileSync(file, 'utf8').split('\n')) {
    if (isLineComment(line)) continue;
    const m = line.match(PATH_RE);
    if (!m) continue;
    const p = norm(m[1]);
    if (IGNORED.has(p)) continue;
    all.add(p);
    if (!/hidden:\s*true/.test(line)) visible.add(p);
  }
  return { all, visible };
}

function diff(a, b) {
  // 返回 {extra: 在 a 不在 b, missing: 在 b 不在 a}
  const extra = [...a].filter((x) => !b.has(x)).sort();
  const missing = [...b].filter((x) => !a.has(x)).sort();
  return { extra, missing };
}

function printList(title, items) {
  if (!items.length) return;
  console.log(`  ${title} (${items.length}):`);
  for (const it of items) console.log(`    ${it}`);
}

const canonical = v1CanonicalRoutes();
const visible = v1VisibleRoutes();
console.log(`v1 标准：${canonical.size} 条可路由 path，${visible.size} 个可见菜单项\n`);

let failed = false;
for (const [skin, file] of Object.entries(SKINS)) {
  const { all, visible: vis } = skinRoutes(file);
  const r = diff(all, canonical);       // 路由集合 vs v1
  const v = diff(vis, visible);         // 可见菜单 vs v1
  const ok = !r.extra.length && !r.missing.length && !v.extra.length && !v.missing.length;
  console.log(`── ${skin} ${ok ? '✓ 对齐' : '✗ 不一致'} （${all.size} 路由 / ${vis.size} 可见）`);
  if (!ok) {
    failed = true;
    printList('多余路由（v1 没有，需删）', r.extra);
    printList('缺失路由（v1 有，需补）', r.missing);
    printList('多显菜单（v1 隐藏/无，需隐藏或删）', v.extra);
    printList('少显菜单（v1 可见，需显示）', v.missing);
  }
  console.log('');
}

if (failed) {
  console.error('✗ skin-parity 失败：v2/v3 与 v1 标准不一致（见上）。');
  process.exit(1);
}
console.log('✓ skin-parity 通过：v2/v3 路由与可见菜单均与 v1 一致。');
