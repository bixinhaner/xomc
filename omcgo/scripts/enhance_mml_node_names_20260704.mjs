import { createRequire } from 'node:module';
import fs from 'node:fs';
import path from 'node:path';

const require = createRequire('/Users/a1/Desktop/vscode/goomc/omcmb/');
const XLSX = require('xlsx');

const ROOT = '/Users/a1/Desktop/vscode/goomc';
const INPUT = path.join(
  ROOT,
  'omcgo/data/model-library/standard-params-to-mml-tree-analysis-20260703.xlsx',
);
const CATALOG = path.join(
  ROOT,
  'omcgo/datamodels/mml-catalog/cmcc-tdlte-v2.3.json',
);
const SEED = path.join(
  ROOT,
  'omcgo/internal/config/parammodel/mmlstandardloader/seeds/cmcc_tdlte_v23.json',
);
const OUTPUT = path.join(
  ROOT,
  'omcgo/data/model-library/standard-params-to-mml-tree-analysis-20260703-node-names-20260704.xlsx',
);
const SUMMARY_JSON = path.join(
  ROOT,
  'omcgo/data/model-library/standard-params-to-mml-tree-analysis-20260703-node-names-20260704.summary.json',
);

const INSTANCE = '{i}';
const ROOT_SEGMENTS = new Set(['Device', 'InternetGatewayDevice']);
const DISPLAY_SKIP_SEGMENTS = new Set([
  'Device',
  'InternetGatewayDevice',
  'Services',
  'FAPService',
  'CellConfig',
]);

function normalizeObjectPath(value) {
  let s = String(value || '').trim();
  if (!s) return '';
  s = s.replace(/\*$/u, '');
  if (!s.endsWith('.')) s += '.';
  return s;
}

function pathSegments(objectPath) {
  return normalizeObjectPath(objectPath)
    .replace(/\.$/u, '')
    .split('.')
    .filter(Boolean);
}

function businessSegments(objectPath) {
  return pathSegments(objectPath).filter(
    (seg) => !ROOT_SEGMENTS.has(seg) && seg !== INSTANCE,
  );
}

function displaySegments(objectPath) {
  const parts = pathSegments(objectPath).filter(
    (seg) => !DISPLAY_SKIP_SEGMENTS.has(seg) && seg !== INSTANCE,
  );
  return parts.length > 0 ? parts : businessSegments(objectPath);
}

function endsWithInstance(objectPath) {
  const parts = pathSegments(objectPath);
  return parts[parts.length - 1] === INSTANCE;
}

function canonicalNoInstance(objectPath) {
  return pathSegments(objectPath)
    .filter((seg) => seg !== INSTANCE)
    .join('.');
}

function addName(map, key, zh, source) {
  if (!key || !zh) return;
  const item = map.get(key) || { names: new Set(), sources: new Set() };
  item.names.add(zh);
  item.sources.add(source);
  map.set(key, item);
}

function buildKnownNameMaps() {
  const exact = new Map();
  const noInstance = new Map();

  const catalog = JSON.parse(fs.readFileSync(CATALOG, 'utf8'));
  for (const group of catalog.groups || []) {
    for (const command of group.commands || []) {
      const zh = command.logicalNameI18n?.['zh-CN'];
      const objectPath = normalizeObjectPath(command.groupCodeObject || '');
      addName(exact, objectPath, zh, 'catalog_exact');
      addName(noInstance, canonicalNoInstance(objectPath), zh, 'catalog_no_instance');
    }
  }

  const seed = JSON.parse(fs.readFileSync(SEED, 'utf8'));
  for (const group of seed.groups || []) {
    for (const command of group.commands || []) {
      const zh = command.name;
      const objectPath = normalizeObjectPath(command.object_path || '');
      addName(exact, objectPath, zh, 'seed_exact');
      addName(noInstance, canonicalNoInstance(objectPath), zh, 'seed_no_instance');
    }
  }

  return { exact, noInstance };
}

function lookupKnownZh(objectPath, maps) {
  const exactItem = maps.exact.get(normalizeObjectPath(objectPath));
  if (exactItem && exactItem.names.size === 1) {
    return {
      zh: [...exactItem.names][0],
      source: [...exactItem.sources].join('|'),
    };
  }

  const noInstanceItem = maps.noInstance.get(canonicalNoInstance(objectPath));
  if (noInstanceItem && noInstanceItem.names.size === 1) {
    return {
      zh: [...noInstanceItem.names][0],
      source: [...noInstanceItem.sources].join('|'),
    };
  }

  return { zh: '', source: '' };
}

function leafName(objectPath) {
  const parts = businessSegments(objectPath);
  return parts.length > 0 ? parts[parts.length - 1] : '';
}

function tailName(objectPath, depth) {
  const parts = displaySegments(objectPath);
  if (parts.length === 0) return '';
  return parts.slice(Math.max(0, parts.length - depth)).join(' ');
}

function displayNameCandidates(objectPath, maxDepth = 4) {
  const parts = displaySegments(objectPath);
  if (parts.length === 0) return [];
  const names = [];
  const isInstance = endsWithInstance(objectPath);
  const pushBase = (base) => {
    if (!base) return;
    names.push(base);
    if (isInstance) {
      names.push(`${base} Instance`);
    } else {
      names.push(`${base} Object`);
    }
  };
  const max = Math.min(maxDepth, parts.length);
  for (let depth = 1; depth <= max; depth += 1) {
    const base = parts.slice(Math.max(0, parts.length - depth)).join(' ');
    pushBase(base);
  }
  if (parts.length > maxDepth) {
    pushBase(parts.join(' '));
  }

  const fullParts = businessSegments(objectPath);
  const fullMax = Math.min(maxDepth + 2, fullParts.length);
  for (let depth = 1; depth <= fullMax; depth += 1) {
    const base = fullParts.slice(Math.max(0, fullParts.length - depth)).join(' ');
    pushBase(base);
  }
  if (fullParts.length > fullMax) {
    pushBase(fullParts.join(' '));
  }
  return [...new Set(names.filter(Boolean))];
}

function toCommandSuffix(name) {
  return String(name || '')
    .replace(/([a-z0-9])([A-Z])/gu, '$1_$2')
    .replace(/([A-Z]+)([A-Z][a-z])/gu, '$1_$2')
    .replace(/[^A-Za-z0-9]+/gu, '_')
    .replace(/^_+|_+$/gu, '')
    .replace(/_+/gu, '_')
    .toUpperCase();
}

function suggestedCommands(candidateCommands, suggestedName) {
  const suffix = toCommandSuffix(suggestedName);
  if (!suffix) return '';
  const ops = String(candidateCommands || '')
    .split(';')
    .map((part) => part.trim().split(/\s+/u)[0])
    .filter(Boolean);
  if (ops.length === 0) return '';
  return ops.map((op) => `${op} ${suffix}`).join(';');
}

function uniqueObjectRows(rows) {
  const byKey = new Map();
  for (const row of rows) {
    const objectPath = normalizeObjectPath(row.target_object_path);
    if (!objectPath) continue;
    const key = `${row.target_chapter_code || ''}|${objectPath}`;
    if (!byKey.has(key)) byKey.set(key, row);
  }
  return [...byKey.values()];
}

function buildSuggestions(candidateRows, maps) {
  const objectRows = uniqueObjectRows(candidateRows);
  const leafGroups = new Map();
  for (const row of objectRows) {
    const chapter = row.target_chapter_code || '';
    const leaf = leafName(row.target_object_path) || row.target_node_name || '';
    const key = `${chapter}|${leaf}`;
    if (!leafGroups.has(key)) leafGroups.set(key, []);
    leafGroups.get(key).push(row);
  }

  const suggestions = new Map();
  for (const row of objectRows) {
    const objectPath = normalizeObjectPath(row.target_object_path);
    const chapter = row.target_chapter_code || '';
    const leaf = leafName(objectPath) || String(row.target_node_name || '');
    const group = leafGroups.get(`${chapter}|${leaf}`) || [row];

    let suggestedName = leaf;
    let nameSource = 'path_last_non_instance_segment';
    let conflict = '';
    let note = '';

    if (String(row.target_node_name || '') === INSTANCE) {
      note = `原 target_node_name 为 ${INSTANCE}，改取最后一个非 ${INSTANCE} 的业务段`;
    }

    if (group.length > 1) {
      conflict = `same_chapter_duplicate_leaf:${leaf}`;
      const objects = group.map((r) => normalizeObjectPath(r.target_object_path));
      for (const currentName of displayNameCandidates(objectPath)) {
        const counts = new Map();
        for (const other of objects) {
          const candidate = displayNameCandidates(other).find((name) => name === currentName);
          if (candidate) counts.set(candidate, (counts.get(candidate) || 0) + 1);
        }
        if ((counts.get(currentName) || 0) === 1) {
          suggestedName = currentName;
          nameSource = 'path_tail_disambiguated';
          break;
        }
      }
      if (suggestedName === leaf) {
        suggestedName = displayNameCandidates(objectPath, 6)[0] || canonicalNoInstance(objectPath);
        nameSource = 'path_tail_disambiguated_fallback';
      }
    }

    const zh = lookupKnownZh(objectPath, maps);
    const key = `${chapter}|${objectPath}`;
    suggestions.set(key, {
      suggested_node_name: suggestedName,
      suggested_command_suffix: toCommandSuffix(suggestedName),
      suggested_candidate_commands: suggestedCommands(row.candidate_commands, suggestedName),
      suggested_display_name_zh: zh.zh,
      suggested_node_name_source: nameSource,
      suggested_display_name_zh_source: zh.source,
      suggested_name_conflict: conflict,
      suggested_name_note: note,
    });
  }
  return suggestions;
}

function getSuggestion(row, suggestions) {
  const key = `${row.target_chapter_code || ''}|${normalizeObjectPath(row.target_object_path)}`;
  return suggestions.get(key) || {
    suggested_node_name: '',
    suggested_command_suffix: '',
    suggested_candidate_commands: '',
    suggested_display_name_zh: '',
    suggested_node_name_source: '',
    suggested_display_name_zh_source: '',
    suggested_name_conflict: '',
    suggested_name_note: '',
  };
}

function enhanceRows(rows, suggestions) {
  return rows.map((row) => ({
    ...row,
    ...getSuggestion(row, suggestions),
  }));
}

function withWidths(ws, rows) {
  if (rows.length === 0) return ws;
  const headers = Object.keys(rows[0]);
  ws['!cols'] = headers.map((header) => {
    const max = Math.max(
      String(header).length,
      ...rows.slice(0, 500).map((row) => String(row[header] ?? '').length),
    );
    return { wch: Math.max(10, Math.min(60, max + 2)) };
  });
  ws['!autofilter'] = { ref: ws['!ref'] };
  return ws;
}

function sheetFromRows(rows) {
  const ws = XLSX.utils.json_to_sheet(rows);
  return withWidths(ws, rows);
}

function appendSummaryRows(wb, stats) {
  const sheetName = '节点命名规则';
  const rows = [
    { item: '处理对象', value: '新增参数节点候选 / 标准参数明细 / 对象节点候选', note: '' },
    { item: '核心规则', value: `不使用 ${INSTANCE} 作为节点显示名`, note: `${INSTANCE} 只表示多实例占位符` },
    { item: '英文技术名', value: '取最后一个非 {i} 的业务段', note: '同章节重名时追加父级上下文直到唯一' },
    { item: '中文显示名', value: '优先匹配已有 catalog/seed 逻辑名', note: '未命中时留空，后续人工补中文' },
    { item: '建议命令后缀', value: 'suggested_command_suffix', note: '由 suggested_node_name 转大写下划线生成' },
    { item: '新增参数节点候选行数', value: stats.candidateRows, note: '' },
    { item: '原 target_node_name 为 {i} 的候选行数', value: stats.oldInstanceNameRows, note: '' },
    { item: '建议名仍为空的候选行数', value: stats.emptySuggestedRows, note: '' },
    { item: '同章节需要消歧的候选行数', value: stats.conflictRows, note: '' },
    { item: '匹配到中文显示名的候选行数', value: stats.zhMatchedRows, note: '' },
  ];
  const ws = XLSX.utils.json_to_sheet(rows);
  ws['!cols'] = [{ wch: 28 }, { wch: 42 }, { wch: 54 }];
  wb.SheetNames.push(sheetName);
  wb.Sheets[sheetName] = ws;
}

const maps = buildKnownNameMaps();
const wb = XLSX.readFile(INPUT);
const candidateRows = XLSX.utils.sheet_to_json(wb.Sheets['新增参数节点候选'], { defval: '' });
const suggestions = buildSuggestions(candidateRows, maps);

const enhancedCandidateRows = enhanceRows(candidateRows, suggestions);
wb.Sheets['新增参数节点候选'] = sheetFromRows(enhancedCandidateRows);

for (const sheetName of ['对象节点候选', '标准参数明细']) {
  if (wb.Sheets[sheetName]) {
    const rows = XLSX.utils.sheet_to_json(wb.Sheets[sheetName], { defval: '' });
    wb.Sheets[sheetName] = sheetFromRows(enhanceRows(rows, suggestions));
  }
}

const stats = {
  input: INPUT,
  output: OUTPUT,
  candidateRows: enhancedCandidateRows.length,
  oldInstanceNameRows: enhancedCandidateRows.filter((r) => r.target_node_name === INSTANCE).length,
  emptySuggestedRows: enhancedCandidateRows.filter((r) => !r.suggested_node_name).length,
  conflictRows: enhancedCandidateRows.filter((r) => r.suggested_name_conflict).length,
  zhMatchedRows: enhancedCandidateRows.filter((r) => r.suggested_display_name_zh).length,
  generatedAt: new Date().toISOString(),
};

if (!wb.SheetNames.includes('节点命名规则')) appendSummaryRows(wb, stats);

XLSX.writeFile(wb, OUTPUT, { bookType: 'xlsx' });
fs.writeFileSync(SUMMARY_JSON, `${JSON.stringify(stats, null, 2)}\n`);
console.log(JSON.stringify(stats, null, 2));
