/**
 * biz_code → i18n key 映射表
 *
 * Issue #730: 登录错误信息国际化
 * - 前端按 biz_code 查语料，后端 msg 作 fallback
 * - 用词统一：「禁用」→「锁定」
 */
export const BIZ_CODE_I18N_MAP: Record<number, string> = {
  2034: 'product.standardParams.pathExists',
  2601: 'geofence.validation.nameInvalid',
  2602: 'geofence.validation.nameDuplicate',
  2603: 'geofence.lifecycle.activeJobsBlockArchive',
  2604: 'geofence.lifecycle.previewExpired',
  // 登录相关错误码（7000-7099）
  7000: 'login.error.invalidCredentials',  // 登录凭据无效
  7001: 'login.error.userNotFound',        // 用户不存在
  7002: 'login.error.accountLocked',       // 账号被锁定（长期未登录/管理员禁用）
  7003: 'login.error.invalidCredentials',  // 解密失败（与 7000 共用语料）
  7004: 'login.error.plaintextDisabled',   // 明文密码禁用
  7005: 'login.error.wrongPassword',       // 密码错误
  7010: 'login.captcha.required',          // 需要验证码（已有语料）
  7011: 'login.captcha.invalid',           // 验证码错误（已有语料）
  7012: 'login.error.accountTempLocked',   // 暴力破解临时锁定
  7013: 'login.error.accountExpired',      // 账号过期
  7014: 'login.error.ipRateLimited',       // IP 限流
  7099: 'login.error.unknown',             // 未知错误
}

/**
 * 根据 biz_code 获取 i18n key
 * @returns i18n key，未找到返回 undefined（调用方应 fallback 到后端 msg 或默认语料）
 */
export function getI18nKeyByBizCode(bizCode: number | undefined): string | undefined {
  if (bizCode === undefined) return undefined
  return BIZ_CODE_I18N_MAP[bizCode]
}
