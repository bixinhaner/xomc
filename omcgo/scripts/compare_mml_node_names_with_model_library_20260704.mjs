import { createRequire } from 'node:module';
import fs from 'node:fs';
import path from 'node:path';

const require = createRequire('/Users/a1/Desktop/vscode/goomc/omcmb/');
const XLSX = require('xlsx');

const ROOT = '/Users/a1/Desktop/vscode/goomc';
const CANDIDATE_XLSX = path.join(
  ROOT,
  'omcgo/data/model-library/standard-params-to-mml-tree-analysis-20260703-node-names-20260704.xlsx',
);
const MODEL_LIBRARY_XLSX = path.join(
  ROOT,
  'omcgo/data/model-library/cmcc-southbound-model-library-path-rows-dedup-category-path-20260703.xlsx',
);
const OUTPUT_XLSX = path.join(
  ROOT,
  'omcgo/data/model-library/standard-params-to-mml-tree-node-name-model-library-comparison-20260704.xlsx',
);
const SUMMARY_JSON = path.join(
  ROOT,
  'omcgo/data/model-library/standard-params-to-mml-tree-node-name-model-library-comparison-20260704.summary.json',
);

const GENERIC_OBJECT_CATEGORY = '多实例Object个数读写属性';
const ROOT_ALIAS = new Map([['InternetGatewayDevice', 'Device']]);

function toText(value) {
  return String(value ?? '').trim();
}

function unique(values) {
  return [...new Set(values.map(toText).filter(Boolean))];
}

function joinUnique(values, limit = 8) {
  const items = unique(values);
  const shown = items.slice(0, limit).join('; ');
  return items.length > limit ? `${shown}; ...(+${items.length - limit})` : shown;
}

function pathSegments(value, aliasRoot = true) {
  let s = toText(value);
  if (!s) return [];
  s = s.replace(/\*$/u, '').replace(/\.+$/u, '');
  const parts = s.split('.').filter(Boolean);
  return parts.map((seg, index) => {
    if (/^\d+$/u.test(seg)) return '{i}';
    if (aliasRoot && index === 0 && ROOT_ALIAS.has(seg)) return ROOT_ALIAS.get(seg);
    return seg;
  });
}

function normalizedPath(value, aliasRoot = true) {
  return pathSegments(value, aliasRoot).join('.');
}

function pathVariants(value) {
  const base = normalizedPath(value);
  const variants = new Set([base]);
  // TD-LTE model sheets often omit the CellConfig instance level while the
  // standard tree keeps it. Treat that instance as optional for naming evidence.
  if (base.includes('.CellConfig.{i}.')) {
    variants.add(base.replace(/\.CellConfig\.\{i\}\./gu, '.CellConfig.'));
  }
  return [...variants].filter(Boolean);
}

function isPrefixPath(modelPathNorm, objectPathNorm) {
  return modelPathNorm === objectPathNorm || modelPathNorm.startsWith(`${objectPathNorm}.`);
}

function variantsPrefixMatch(modelVariants, objectVariants) {
  return modelVariants.some((modelPath) =>
    objectVariants.some((objectPath) => isPrefixPath(modelPath, objectPath)),
  );
}

function variantsExactMatch(modelVariants, objectVariants) {
  return modelVariants.some((modelPath) => objectVariants.includes(modelPath));
}

function cleanZhDisplayName(value) {
  let s = toText(value);
  if (!s) return '';
  s = s.replace(/节点$/u, '');
  s = s.replace(/参数管理$/u, '');
  s = s.replace(/配置管理$/u, '');
  s = s.replace(/参数$/u, '');
  s = s.replace(/配置$/u, '');
  s = s.replace(/\s+/gu, ' ').trim();
  return s || toText(value);
}

function countBy(rows, getter) {
  const counts = new Map();
  for (const row of rows) {
    const key = toText(getter(row));
    if (!key) continue;
    counts.set(key, (counts.get(key) || 0) + 1);
  }
  return [...counts.entries()]
    .map(([value, count]) => ({ value, count }))
    .sort((a, b) => b.count - a.count || a.value.localeCompare(b.value, 'zh-CN'));
}

function dedupeModelRows(rows) {
  const seen = new Set();
  const deduped = [];
  for (const row of rows) {
    const key = [
      row._path_norm,
      row.entry_type,
      row.category,
      row.name_zh,
      row.param_name,
      row.source_file,
      row.source_sheet,
      row.source_row,
    ]
      .map(toText)
      .join('|');
    if (seen.has(key)) continue;
    seen.add(key);
    deduped.push(row);
  }
  return deduped;
}

function pickEvidenceRows(exactObjects, paramRows, otherRows) {
  return [
    ...exactObjects.slice(0, 3),
    ...paramRows.slice(0, 7),
    ...otherRows.slice(0, 2),
  ].slice(0, 10);
}

function makeRecommendation(candidate, modelRows) {
  const exactObjects = modelRows.filter(
    (row) =>
      row.entry_type === 'object' && variantsExactMatch(row._path_variants, candidate._object_variants),
  );
  const paramRows = modelRows.filter((row) => row.entry_type === 'parameter');
  const otherRows = modelRows.filter((row) => row.entry_type !== 'parameter' && row.entry_type !== 'object');
  const objectNameCounts = countBy(exactObjects, (row) => row.name_zh);
  const categoryRows = paramRows.filter((row) => toText(row.category) && row.category !== GENERIC_OBJECT_CATEGORY);
  const categoryCounts = countBy(categoryRows, (row) => row.category);
  const paramNameZh = joinUnique(paramRows.map((row) => row.name_zh), 12);
  const sourceFiles = joinUnique(modelRows.map((row) => row.source_file), 8);
  const topCategory = categoryCounts[0] || { value: '', count: 0 };
  const categoryTotal = categoryRows.length;
  const categoryShare = categoryTotal > 0 ? topCategory.count / categoryTotal : 0;
  const existingZh = toText(candidate.suggested_display_name_zh);

  let recommendedZh = '';
  let source = '';
  let confidence = '低';
  let reason = '';
  let needsReview = 'Y';

  if (objectNameCounts.length > 0) {
    recommendedZh = cleanZhDisplayName(objectNameCounts[0].value);
    source = '模型库对象中文名';
    confidence = '高';
    needsReview = 'N';
    reason = `模型库存在对象行：${objectNameCounts[0].value}`;
  } else if (topCategory.value && categoryShare >= 0.75 && topCategory.count >= 3) {
    recommendedZh = cleanZhDisplayName(topCategory.value);
    source = '模型库主分类';
    confidence = '中';
    needsReview = 'N';
    reason = `参数主分类占比 ${Math.round(categoryShare * 100)}%，可作为中文显示名`;
  } else if (topCategory.value && categoryShare >= 0.5 && topCategory.count >= 2) {
    recommendedZh = cleanZhDisplayName(topCategory.value);
    source = '模型库分类候选';
    confidence = '中';
    needsReview = 'Y';
    reason = `参数分类存在但不够集中，占比 ${Math.round(categoryShare * 100)}%，建议人工确认`;
  } else if (existingZh) {
    recommendedZh = cleanZhDisplayName(existingZh);
    source = '现有MML目录中文名';
    confidence = '中';
    needsReview = 'Y';
    reason = '模型库未给出明确对象/主分类，沿用已有目录中文名';
  } else if (modelRows.length > 0) {
    source = '模型库参数证据不足';
    reason = '已匹配模型库 path，但没有稳定对象中文名或主分类';
  } else {
    source = 'path派生';
    reason = '模型库未匹配到该对象 path，英文节点名只能按标准 path 派生';
  }

  return {
    exactObjects,
    paramRows,
    otherRows,
    objectNameCounts,
    categoryCounts,
    topCategory,
    categoryTotal,
    categoryShare,
    paramNameZh,
    sourceFiles,
    recommendedZh,
    source,
    confidence,
    reason,
    needsReview,
  };
}

function sheetFromRows(rows, headers) {
  const data = rows.map((row) => {
    const output = {};
    for (const header of headers) output[header] = row[header] ?? '';
    return output;
  });
  const ws = XLSX.utils.json_to_sheet(data, { header: headers });
  ws['!cols'] = headers.map((header) => {
    const max = Math.max(
      header.length,
      ...data.slice(0, 500).map((row) => toText(row[header]).length),
    );
    return { wch: Math.max(10, Math.min(72, max + 2)) };
  });
  ws['!autofilter'] = { ref: ws['!ref'] };
  return ws;
}

function addSheet(wb, name, rows, headers) {
  wb.SheetNames.push(name);
  wb.Sheets[name] = sheetFromRows(rows, headers);
}

const candidateWb = XLSX.readFile(CANDIDATE_XLSX);
const modelWb = XLSX.readFile(MODEL_LIBRARY_XLSX);
const candidateRows = XLSX.utils.sheet_to_json(candidateWb.Sheets['新增参数节点候选'], { defval: '' });
const modelRowsRaw = XLSX.utils.sheet_to_json(modelWb.Sheets[modelWb.SheetNames[0]], { defval: '' });

const modelRows = modelRowsRaw.map((row) => ({
  ...row,
  _path_norm: normalizedPath(row.path),
  _path_norm_no_alias: normalizedPath(row.path, false),
  _path_variants: pathVariants(row.path),
}));

const comparisonRows = [];
const evidenceRows = [];
const noModelMatchRows = [];
const evidenceCandidateKeys = new Set();

for (const candidate of candidateRows) {
  const objectNorm = normalizedPath(candidate.target_object_path);
  const objectNormNoAlias = normalizedPath(candidate.target_object_path, false);
  const objectVariants = pathVariants(candidate.target_object_path);
  const uniqueCandidateKey = `${candidate.target_chapter_code || ''}|${objectNorm}`;
  const candidateWithNorm = {
    ...candidate,
    _object_norm: objectNorm,
    _object_norm_no_alias: objectNormNoAlias,
    _object_variants: objectVariants,
  };
  const matches = dedupeModelRows(
    modelRows.filter((row) => variantsPrefixMatch(row._path_variants, objectVariants)),
  );
  const recommendation = makeRecommendation(candidateWithNorm, matches);
  const exactObjectNames = joinUnique(recommendation.objectNameCounts.map((item) => item.value), 8);
  const categoryCandidates = recommendation.categoryCounts
    .slice(0, 8)
    .map((item) => `${item.value}(${item.count})`)
    .join('; ');
  const exactObjectSourceFiles = joinUnique(recommendation.exactObjects.map((row) => row.source_file), 8);
  const modelMatchStatus =
    matches.length === 0
      ? '未匹配'
      : recommendation.exactObjects.length > 0
        ? '对象行匹配'
        : recommendation.paramRows.length > 0
          ? '参数前缀匹配'
          : '其他匹配';

  const outputRow = {
    target_chapter_code: candidate.target_chapter_code,
    target_chapter_name: candidate.target_chapter_name,
    target_object_path: candidate.target_object_path,
    normalized_object_path: objectNorm,
    path_count: candidate.path_count,
    candidate_commands: candidate.candidate_commands,
    old_target_node_name: candidate.target_node_name,
    path_suggested_node_name: candidate.suggested_node_name,
    path_suggested_command_suffix: candidate.suggested_command_suffix,
    path_suggested_candidate_commands: candidate.suggested_candidate_commands,
    path_suggested_display_name_zh: candidate.suggested_display_name_zh,
    model_match_status: modelMatchStatus,
    model_match_count: matches.length,
    model_exact_object_count: recommendation.exactObjects.length,
    model_param_count: recommendation.paramRows.length,
    model_exact_object_name_zh: exactObjectNames,
    model_exact_object_source_files: exactObjectSourceFiles,
    model_top_category: recommendation.topCategory.value,
    model_top_category_count: recommendation.topCategory.count || '',
    model_top_category_share: recommendation.categoryTotal
      ? Number(recommendation.categoryShare.toFixed(4))
      : '',
    model_category_candidates: categoryCandidates,
    model_sample_param_names_zh: recommendation.paramNameZh,
    model_source_files: recommendation.sourceFiles,
    recommended_node_name_en: candidate.suggested_node_name,
    recommended_command_suffix: candidate.suggested_command_suffix,
    recommended_candidate_commands: candidate.suggested_candidate_commands,
    recommended_display_name_zh: recommendation.recommendedZh,
    recommended_name_source: recommendation.source,
    recommended_confidence: recommendation.confidence,
    need_manual_review: recommendation.needsReview,
    recommended_name_reason: recommendation.reason,
  };
  comparisonRows.push(outputRow);
  if (matches.length === 0) noModelMatchRows.push(outputRow);

  if (!evidenceCandidateKeys.has(uniqueCandidateKey)) {
    evidenceCandidateKeys.add(uniqueCandidateKey);
    for (const row of pickEvidenceRows(recommendation.exactObjects, recommendation.paramRows, recommendation.otherRows)) {
      evidenceRows.push({
        target_chapter_code: candidate.target_chapter_code,
        target_object_path: candidate.target_object_path,
        recommended_node_name_en: candidate.suggested_node_name,
        recommended_display_name_zh: recommendation.recommendedZh,
        match_type:
          row.entry_type === 'object' && variantsExactMatch(row._path_variants, objectVariants)
            ? 'exact_object'
            : row.entry_type === 'parameter'
              ? 'child_parameter'
              : 'matched_row',
        model_code: row.model_code,
        source_file: row.source_file,
        source_sheet: row.source_sheet,
        source_row: row.source_row,
        entry_type: row.entry_type,
        category: row.category,
        model_path: row.path,
        normalized_model_path: row._path_norm,
        name_zh: row.name_zh,
        param_name: row.param_name,
        access: row.access,
        data_type: row.data_type,
        description: row.description,
        remark: row.remark,
      });
    }
  }
}

const seenUniqueRows = new Set();
const uniqueComparisonRows = comparisonRows.filter((row) => {
  const key = `${row.target_chapter_code || ''}|${row.normalized_object_path || ''}`;
  if (seenUniqueRows.has(key)) return false;
  seenUniqueRows.add(key);
  return true;
});
const uniqueNoModelMatchRows = uniqueComparisonRows.filter((row) => row.model_match_status === '未匹配');

const summaryCounts = {
  candidate_rows: comparisonRows.length,
  unique_node_candidates: uniqueComparisonRows.length,
  duplicate_candidate_rows: comparisonRows.length - uniqueComparisonRows.length,
  model_matched: uniqueComparisonRows.filter((row) => row.model_match_status !== '未匹配').length,
  exact_object_matched: uniqueComparisonRows.filter((row) => row.model_exact_object_count > 0).length,
  parameter_prefix_matched: uniqueComparisonRows.filter(
    (row) => row.model_exact_object_count === 0 && row.model_param_count > 0,
  ).length,
  no_model_match: uniqueComparisonRows.filter((row) => row.model_match_status === '未匹配').length,
  high_confidence: uniqueComparisonRows.filter((row) => row.recommended_confidence === '高').length,
  medium_confidence: uniqueComparisonRows.filter((row) => row.recommended_confidence === '中').length,
  low_confidence: uniqueComparisonRows.filter((row) => row.recommended_confidence === '低').length,
  need_manual_review: uniqueComparisonRows.filter((row) => row.need_manual_review === 'Y').length,
};

const bySource = countBy(uniqueComparisonRows, (row) => row.recommended_name_source).map((item) => ({
  metric: `命名来源：${item.value}`,
  value: item.count,
  note: '',
}));
const byStatus = countBy(uniqueComparisonRows, (row) => row.model_match_status).map((item) => ({
  metric: `模型库匹配状态：${item.value}`,
  value: item.count,
  note: '',
}));

const summaryRows = [
  { metric: '新增节点候选行数', value: summaryCounts.candidate_rows, note: '原表行数，包含同章节同 path 重复' },
  { metric: '去重后新增节点数', value: summaryCounts.unique_node_candidates, note: '按 target_chapter_code + normalized_object_path 去重' },
  { metric: '重复候选行数', value: summaryCounts.duplicate_candidate_rows, note: '' },
  { metric: '模型库 path 匹配数', value: summaryCounts.model_matched, note: '数字实例段已按 {i} 归一，InternetGatewayDevice 已按 Device 对齐' },
  { metric: '模型库对象行精确匹配数', value: summaryCounts.exact_object_matched, note: '优先采用对象行 name_zh' },
  { metric: '仅参数前缀匹配数', value: summaryCounts.parameter_prefix_matched, note: '无对象行时采用主分类/参数名辅助判断' },
  { metric: '模型库未匹配数', value: summaryCounts.no_model_match, note: '只保留 path 派生英文名，建议人工确认中文显示名' },
  { metric: '高置信命名数', value: summaryCounts.high_confidence, note: '来自模型库对象中文名' },
  { metric: '中置信命名数', value: summaryCounts.medium_confidence, note: '来自模型库主分类/现有目录中文名' },
  { metric: '低置信命名数', value: summaryCounts.low_confidence, note: '缺少模型库中文证据' },
  { metric: '需要人工复核数', value: summaryCounts.need_manual_review, note: '' },
  ...byStatus,
  ...bySource,
];

const rulesRows = [
  {
    item: 'path 归一化',
    rule: '删除末尾点号，数字实例段统一转为 {i}，并把 CellConfig.{i}. 作为可选层级辅助匹配',
    example: 'A5MeasureCtrl.1.Enable -> A5MeasureCtrl.{i}.Enable',
  },
  {
    item: '根节点别名',
    rule: 'InternetGatewayDevice 与 Device 视为同一根做匹配',
    example: 'InternetGatewayDevice.Services... -> Device.Services...',
  },
  {
    item: '英文节点名',
    rule: '沿用上一步 path 派生结果，确保不再使用 {i} 作为节点名',
    example: '...QOS.{i}. -> QOS Instance',
  },
  {
    item: '中文显示名优先级',
    rule: '模型库对象 name_zh > 模型库主分类 > 现有 MML 目录中文名 > 留空人工确认',
    example: 'RU实例节点 -> RU实例',
  },
  {
    item: '主分类采用条件',
    rule: '无对象行时，参数主分类占比 >= 75% 且数量 >= 3 可作为中置信中文名',
    example: 'A5事件测量控制参数 -> A5事件测量控制',
  },
];

const comparisonHeaders = [
  'target_chapter_code',
  'target_chapter_name',
  'target_object_path',
  'normalized_object_path',
  'path_count',
  'candidate_commands',
  'old_target_node_name',
  'path_suggested_node_name',
  'path_suggested_command_suffix',
  'path_suggested_candidate_commands',
  'path_suggested_display_name_zh',
  'model_match_status',
  'model_match_count',
  'model_exact_object_count',
  'model_param_count',
  'model_exact_object_name_zh',
  'model_exact_object_source_files',
  'model_top_category',
  'model_top_category_count',
  'model_top_category_share',
  'model_category_candidates',
  'model_sample_param_names_zh',
  'model_source_files',
  'recommended_node_name_en',
  'recommended_command_suffix',
  'recommended_candidate_commands',
  'recommended_display_name_zh',
  'recommended_name_source',
  'recommended_confidence',
  'need_manual_review',
  'recommended_name_reason',
];
const evidenceHeaders = [
  'target_chapter_code',
  'target_object_path',
  'recommended_node_name_en',
  'recommended_display_name_zh',
  'match_type',
  'model_code',
  'source_file',
  'source_sheet',
  'source_row',
  'entry_type',
  'category',
  'model_path',
  'normalized_model_path',
  'name_zh',
  'param_name',
  'access',
  'data_type',
  'description',
  'remark',
];

const wb = XLSX.utils.book_new();
addSheet(wb, '命名结论统计', summaryRows, ['metric', 'value', 'note']);
addSheet(wb, '去重后节点命名', uniqueComparisonRows, comparisonHeaders);
addSheet(wb, '新增节点命名对比', comparisonRows, comparisonHeaders);
addSheet(wb, '模型库证据明细', evidenceRows, evidenceHeaders);
addSheet(wb, '模型库未匹配', uniqueNoModelMatchRows, comparisonHeaders);
addSheet(wb, '命名规则', rulesRows, ['item', 'rule', 'example']);

XLSX.writeFile(wb, OUTPUT_XLSX, { bookType: 'xlsx' });
fs.writeFileSync(
  SUMMARY_JSON,
  `${JSON.stringify(
    {
      candidate_xlsx: CANDIDATE_XLSX,
      model_library_xlsx: MODEL_LIBRARY_XLSX,
      output_xlsx: OUTPUT_XLSX,
      generated_at: new Date().toISOString(),
      ...summaryCounts,
      evidence_rows: evidenceRows.length,
    },
    null,
    2,
  )}\n`,
);

console.log(
  JSON.stringify(
    {
      output_xlsx: OUTPUT_XLSX,
      summary_json: SUMMARY_JSON,
      ...summaryCounts,
      evidence_rows: evidenceRows.length,
    },
    null,
    2,
  ),
);
