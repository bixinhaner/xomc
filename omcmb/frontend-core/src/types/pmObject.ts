/**
 * T-0193 / issue #543 — 设备下钻到小区/PLMN 多制式（4G/5G/GSM） object_ldn 友好名共享工具。
 *
 * 设计：~/Documents/notes/PM功能设计/pm-cell-plmn-drilldown-design.md
 *      ~/Documents/notes/543-PM-砍跨层级配对-多制式唯一键-实施计划.md（阶段 A）
 *      ~/Documents/notes/PM测量对象唯一键-多制式分析-20260618.md（真实样本+维度表）
 *
 * 三制式 object_ldn 格式：
 *   - 4G / LTE：`Cellid=66` / `Cellid=66,PLMN=46001`
 *   - 5G / NR：`Type=Cell,Mode=SA,gNBID=350251605,NrCGI=15153,CUID=1[,PLMNID=00101]`
 *               扩展段：`,NSSAI=0/1/2` `,SCLICEGROUP=0/1/2` `,DUID=1`
 *   - GSM：`Uid=4002-1`
 *
 * 解析规则与后端 `internal/pm/metrics/object_ldn.go::ParseObjectLDN` 一致（单一真值源同步实现）。
 * 5G/GSM 行刻意不复用 cellId/plmn 字段（避免与 LTE 语义混淆，由后端 aggregator 决定配对策略）。
 */

/** object_ldn 拆解结果（按制式补缺段空）。 */
export interface ObjectLdnFields {
  /** 制式判定：'lte' / 'nr' / 'gsm' / 'unknown'。 */
  tech: 'lte' | 'nr' | 'gsm' | 'unknown';
  objectType?: string;
  // 4G
  cellId?: string;
  plmn?: string;
  // 5G
  gnbId?: string;
  nrCgi?: string;
  cuId?: string;
  duId?: string;
  plmnId?: string;
  nssai?: string;
  sliceGroup?: string;
  // GSM
  uid?: string;
}

/** 列小区接口返回的一项（前端视角，已 camelCase）。 */
export interface MetricObject {
  /** 完整 object_ldn 字符串，白名单/过滤用它（如 `Cellid=111172245,PLMN=46068`）。 */
  objectLdn: string;
  /** 拆解出的小区 ID（4G 友好名用），缺段时 undefined。 */
  cellId?: string;
  /** 拆解出的 PLMN（4G 友好名用），缺段时 undefined。 */
  plmn?: string;
}

/** 后端 wire 形态（http.ts 自动 camelCase，这里仅作类型声明备查）。 */
export interface BackendMetricObject {
  object_ldn: string;
  cell_id?: string;
  plmn?: string;
}

// 提取单段值（`Key=Value`，Value 到下一个逗号或串尾）。Cellid 大小写不敏感（兼容旧测试）。
function extractField(objectLdn: string | null | undefined, key: string, ci = false): string | undefined {
  if (!objectLdn) return undefined;
  // 用 \b 边界避免 PLMN 与 PLMNID 互窜
  const flag = ci ? 'i' : '';
  const re = new RegExp(`\\b${key}=([^,]+)`, flag);
  const m = re.exec(objectLdn);
  const v = m?.[1]?.trim();
  return v ? v : undefined;
}

/** 从 object_ldn 字符串拆出 4G 小区 ID（`Cellid=…`，大小写不敏感）。缺则 undefined。 */
export function parseCellId(objectLdn: string | null | undefined): string | undefined {
  return extractField(objectLdn, 'Cellid', true);
}

/** 从 object_ldn 字符串拆出 4G PLMN（`PLMN=…`）。缺则 undefined。 */
export function parsePlmn(objectLdn: string | null | undefined): string | undefined {
  return extractField(objectLdn, 'PLMN');
}

/** 拆 5G gNBID。 */
export function parseGnbId(objectLdn: string | null | undefined): string | undefined {
  return extractField(objectLdn, 'gNBID');
}
/** 拆 5G NrCGI。 */
export function parseNrCgi(objectLdn: string | null | undefined): string | undefined {
  return extractField(objectLdn, 'NrCGI');
}
/** 拆 5G CUID。 */
export function parseCuId(objectLdn: string | null | undefined): string | undefined {
  return extractField(objectLdn, 'CUID');
}
/** 拆 5G DUID。 */
export function parseDuId(objectLdn: string | null | undefined): string | undefined {
  return extractField(objectLdn, 'DUID');
}
/** 拆 5G PLMNID（与 4G PLMN 字段名不同）。 */
export function parsePlmnId(objectLdn: string | null | undefined): string | undefined {
  return extractField(objectLdn, 'PLMNID');
}
/** 拆 5G NSSAI（值含斜杠如 `0/1/2`）。 */
export function parseNssai(objectLdn: string | null | undefined): string | undefined {
  return extractField(objectLdn, 'NSSAI');
}
/** 拆 5G 切片组 SCLICEGROUP。 */
export function parseSliceGroup(objectLdn: string | null | undefined): string | undefined {
  return extractField(objectLdn, 'SCLICEGROUP');
}
/** 拆 GSM Uid（值含连字符如 `4002-1`）。 */
export function parseUid(objectLdn: string | null | undefined): string | undefined {
  return extractField(objectLdn, 'Uid');
}

/** 拆 object_ldn 的 Type 段（如 `Type=gNB` / `Type=Cell`）。 */
export function parseObjectType(objectLdn: string | null | undefined): string | undefined {
  return extractField(objectLdn, 'Type');
}

/** 判断 object_ldn 是否包含某个维度字段名；只看字段名，不要求字段值固定。 */
export function hasObjectLdnField(objectLdn: string | null | undefined, key: string, ci = false): boolean {
  if (!objectLdn) return false;
  const flag = ci ? 'i' : '';
  return new RegExp(`\\b${key}=`, flag).test(objectLdn);
}

/**
 * 解析 object_ldn 全字段 + 制式判定。
 * 与后端 ParseObjectLDN 单一真值源同步实现，制式判定优先级 gNBID/NrCGI > Uid > Cellid。
 */
export function parseObjectLdn(objectLdn: string | null | undefined): ObjectLdnFields {
  const cellId = parseCellId(objectLdn);
  const plmn = parsePlmn(objectLdn);
  const gnbId = parseGnbId(objectLdn);
  const nrCgi = parseNrCgi(objectLdn);
  const cuId = parseCuId(objectLdn);
  const duId = parseDuId(objectLdn);
  const plmnId = parsePlmnId(objectLdn);
  const nssai = parseNssai(objectLdn);
  const sliceGroup = parseSliceGroup(objectLdn);
  const uid = parseUid(objectLdn);
  const objectType = parseObjectType(objectLdn);

  let tech: ObjectLdnFields['tech'];
  if (gnbId || nrCgi) {
    tech = 'nr';
  } else if (uid) {
    tech = 'gsm';
  } else if (cellId || plmn) {
    // 任一 LTE 段命中即认为是 4G 行（兼容历史合成串 `PLMN=46068`，无 Cellid 也能格式化）
    tech = 'lte';
  } else {
    tech = 'unknown';
  }

  // 5G/GSM 行不复用 cellId/plmn 字段（与后端语义一致：避免误触跨层级配对）
  const safeCellId = tech === 'lte' ? cellId : undefined;
  const safePlmn = tech === 'lte' ? plmn : undefined;

  return {
    tech,
    objectType,
    cellId: safeCellId,
    plmn: safePlmn,
    gnbId,
    nrCgi,
    cuId,
    duId,
    plmnId,
    nssai,
    sliceGroup,
    uid,
  };
}

/**
 * object_ldn → 可读友好名（多制式）。
 *
 *   - 4G： `小区{cellId}` 或 `小区{cellId} · PLMN{plmn}`
 *   - 5G： `gNB{gnbId} · 小区{nrCgi}[/{cuId|duId}] · PLMN{plmnId}[ · 切片{nssai}][ · 切片组{sliceGroup}]`
 *           （任一段缺则丢段不丢友好名；gNBID 单独出现且无 NrCGI → 仅 `gNB{gnbId}`）
 *   - GSM：`小区 Uid{uid}`
 *   - 无法识别：回退原串（保留可见信息），空则空串。
 *
 * 纯展示用，不做 i18n（小区/PLMN/gNB/Uid/切片是网管通用术语）。
 */
export function formatObjectLdn(objectLdn: string | null | undefined, locale: 'zh-CN' | 'en-US' = 'zh-CN'): string {
  const raw = (objectLdn ?? '').trim();
  if (!raw) return '';

  const f = parseObjectLdn(raw);
  const parts: string[] = [];
  const cellLabel = locale === 'zh-CN' ? '小区' : 'Cell';
  const sliceLabel = locale === 'zh-CN' ? '切片' : 'Slice';
  const sliceGroupLabel = locale === 'zh-CN' ? '切片组' : 'Slice Group';

  switch (f.tech) {
    case 'lte': {
      if (f.cellId) parts.push(`${cellLabel}${f.cellId}`);
      if (f.plmn) parts.push(`PLMN${f.plmn}`);
      break;
    }
    case 'nr': {
      if (f.gnbId) parts.push(`gNB${f.gnbId}`);
      // 小区段：NrCGI 是核心；CUID/DUID 作为后缀（任一）
      if (f.nrCgi) {
        const suffix = f.cuId ?? f.duId;
        parts.push(suffix ? `${cellLabel}${f.nrCgi}/${suffix}` : `${cellLabel}${f.nrCgi}`);
      }
      if (f.plmnId) parts.push(`PLMN${f.plmnId}`);
      if (f.nssai) parts.push(`${sliceLabel}${f.nssai}`);
      if (f.sliceGroup) parts.push(`${sliceGroupLabel}${f.sliceGroup}`);
      break;
    }
    case 'gsm': {
      // 形如 "小区 Uid4002-1"
      if (f.uid) parts.push(`${cellLabel} Uid${f.uid}`);
      break;
    }
    case 'unknown':
    default:
      // 不进 parts，下面统一回退
      break;
  }

  if (parts.length > 0) return parts.join(' · ');
  // 制式判定后仍无可读段（例如 LTE 行只有 PLMN= 无 Cellid=）退化兜底：用旧 4G 风格
  if (f.plmn && !f.cellId && f.tech !== 'unknown') {
    return `PLMN${f.plmn}`;
  }
  // 完全无法拆解 → 回退原串
  return raw;
}

/** 设备尾号（SN 末 6 位，不足则全量）— 多设备分线时线名前缀用。 */
export function deviceSnTail(sn: string | null | undefined, tailLen = 6): string {
  const s = (sn ?? '').trim();
  if (!s) return '';
  return s.length > tailLen ? s.slice(-tailLen) : s;
}

/**
 * 设备列表分线的系列名：「设备尾号 · 友好名」。
 *   - 有 object_ldn：`{尾号} · {友好名}`
 *   - 无 object_ldn（缺小区信息，兜底单线）：仅 `{尾号}`（退化为按设备）。
 *   - 连尾号都缺：回退友好名或固定占位由调用方处理。
 */
export function buildDeviceSeriesName(
  sn: string | null | undefined,
  objectLdn: string | null | undefined,
): string {
  const tail = deviceSnTail(sn);
  const raw = (objectLdn ?? '').trim();
  const displayName = raw && parseObjectLdn(raw).tech === 'nr' ? raw : formatObjectLdn(raw);
  if (tail && displayName) return `${tail} · ${displayName}`;
  if (tail) return tail;
  return displayName;
}
