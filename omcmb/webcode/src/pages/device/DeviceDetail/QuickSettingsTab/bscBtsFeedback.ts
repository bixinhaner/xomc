/**
 * 共享常量：BSC 顶部"新增 BTS / 删除 BTS"的 feedback store group id。
 * QuickSettingsTab 顶部 Tag 与 BscBtsAddModal 都用这个 key 写/读 lastAction,
 * 让两侧看到的就是同一份"上次操作"状态(与 Trx 行级 Tag 同样的体验)。
 */
export const BSC_BTS_FEEDBACK_GROUP_ID = 'bsc-bts-instance';
