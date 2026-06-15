// qa-614 #369：产品类型标识（productClass）载波变体归一化工具。
//
// 背景：真机上报的 productClass 带载波变体后缀 —— /SC（单载波）/DC（双载波）/CA（载波聚合），
// 三者指向同一物理产品（见 product-name-routing.xml）。固件 IMAGE（升级镜像）做"版本合一"时，
// 同族变体应收敛为同一共同基础标识（如 FAP/MLN/SC、FAP/MLN/DC、FAP/MLN/CA → FAP/MLN），
// 不应把载波变体当作并列选项让用户选三遍/被迫绑死某一变体。
//
// PATCH / FPGA 维持细分（业务确认 IMAGE 才版本合一），故收敛逻辑只在 IMAGE 上传消费点调用。

// 仅当结尾恰为 /SC | /DC | /CA 时剥离后缀；用精确锚定避免误伤 .../CA-cert 之类。
const CARRIER_VARIANT_SUFFIX = /\/(SC|DC|CA)$/;

/**
 * baseProductClass 返回产品类型标识的共同基础标识：
 * 结尾恰为 /SC、/DC、/CA 时剥去该后缀，否则原样返回。
 */
export function baseProductClass(pc: string): string {
  return pc.replace(CARRIER_VARIANT_SUFFIX, '');
}

/**
 * collapseImageProductClasses 把一组 productClass 收敛为去重后的共同基础标识列表，
 * 保持原始顺序（首次出现的基础标识保留位置）。供 IMAGE 上传"产品类型标识"下拉消费。
 */
export function collapseImageProductClasses(list: readonly string[]): string[] {
  const seen = new Set<string>();
  const result: string[] = [];
  for (const pc of list) {
    if (!pc) continue;
    const base = baseProductClass(pc);
    if (!seen.has(base)) {
      seen.add(base);
      result.push(base);
    }
  }
  return result;
}
