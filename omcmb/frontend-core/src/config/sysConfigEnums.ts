// 系统配置中「枚举型」key 的注册表（共享层，KV 编辑器据此渲染下拉框）。
//
// 背景：系统配置是通用键值编辑器，默认把非 bool 值渲染成文本输入框。
// 某些 key（如设备名称同步策略 nameSyncMode）语义上是「固定几个候选值」的枚举，
// 手敲字符串既易错也不友好。此注册表声明这类 key 的候选值与显示文案（i18n key），
// 编辑器命中注册表时渲染为下拉选择，未命中的 key 保持原有输入框行为不变。
//
// 定义放共享层单点维护；具体下拉控件由各皮肤（v2 shadcn / v3 HUD）各自渲染。

export interface SysConfigEnumOption {
  /** 存储值（写入 sys_configs.value） */
  value: string;
  /** 显示文案的 i18n key */
  labelKey: string;
}

export interface SysConfigEnumSpec {
  /** 候选值列表 */
  options: SysConfigEnumOption[];
}

// key 形如 "<category>.<key>"，与通用编辑器按 category 拉取后的单个 key 拼接匹配。
export const SYS_CONFIG_ENUMS: Record<string, SysConfigEnumSpec> = {
  // 设备名称同步策略（Issue #758）：三值枚举（已删 off），与 v1 三选一单选组、后端 loadConfig 一致。
  'device.nameSyncMode': {
    options: [
      { value: 'auto_lmt_to_omc', labelKey: 'system.device.nameSync.mode.autoLmtToOmc' },
      { value: 'auto_omc_to_lmt', labelKey: 'system.device.nameSync.mode.autoOmcToLmt' },
      { value: 'prompt', labelKey: 'system.device.nameSync.mode.prompt' },
    ],
  },
};

/** 查询某个 (category, key) 是否为枚举型，返回其候选值规格；非枚举返回 undefined。 */
export function getSysConfigEnum(
  category: string,
  key: string,
): SysConfigEnumSpec | undefined {
  return SYS_CONFIG_ENUMS[`${category}.${key}`];
}
