import { Badge, Tooltip } from 'antd';
import type { CSSProperties, ReactNode } from 'react';

import type {
  UnifiedFileTransferDeviceItem,
  UnifiedFileTransferTask,
} from '@core/types/unifiedFileTransfer';

// 翻译函数签名 —— useT 返回的 (id, values?) => string，简化为窄类型供工具函数使用。
type Translate = (id: string, values?: Record<string, string | number>) => string;

// renderEllipsisCell — 长文本列单行截断 + Tooltip 兜底。任务名称 / 设备名 /
// 固件版本 / 文件名 等业务字段都可能超过列宽，统一封装避免每列写重复 markup。
//
// 实现要点：用 <div> 块级容器 + CSS（overflow:hidden, text-overflow:ellipsis,
// white-space:nowrap）而不是 Antd Text ellipsis。原因：Text ellipsis 渲染为
// <span>（inline），被 Tooltip wrapper（也是 inline）包后，maxWidth/width 都
// 不生效；只能依赖 Text 内部 display:inline-block 但实测在 Table fixed layout
// 下仍会触发换行。改用 block div 自然撑满 td 宽度，css ellipsis 100% 可靠。
//
// 用法：
//   render: (_, record) => renderEllipsisCell(record.taskName)
//   render: (_, record) => renderEllipsisCell(record.taskName, { strong: true })
//   render: (_, record) => renderEllipsisCell(record.taskName, {
//     subtitle: record.deviceName,  // 副行也跟着 ellipsis + tooltip
//   })
export function renderEllipsisCell(
  value: string | undefined | null,
  opts?: {
    strong?: boolean;
    secondary?: boolean;
    subtitle?: string | undefined | null;
    /** 业务渲染失败时的 fallback，比如 '-'。默认 '-'。 */
    placeholder?: ReactNode;
  },
): ReactNode {
  const display = (value ?? '').toString().trim();
  const placeholder = opts?.placeholder ?? '-';
  const sub = (opts?.subtitle ?? '').toString().trim();
  if (!display && !sub) return placeholder;

  const baseCellStyle: CSSProperties = {
    overflow: 'hidden',
    textOverflow: 'ellipsis',
    whiteSpace: 'nowrap',
    // block 让 maxWidth/width 100% 跟着 td 宽度走，避免 Tooltip wrapper inline 时
    // maxWidth 失效导致换行。
    display: 'block',
    width: '100%',
  };

  const mainNode = display ? (
    <Tooltip title={display} placement="topLeft" mouseEnterDelay={0.2}>
      <div
        style={{
          ...baseCellStyle,
          fontWeight: opts?.strong ? 600 : undefined,
          color: opts?.secondary ? 'rgba(0,0,0,0.45)' : undefined,
        }}
      >
        {display}
      </div>
    </Tooltip>
  ) : null;

  if (!sub) return mainNode ?? placeholder;

  return (
    <div style={{ minWidth: 0 }}>
      {mainNode}
      <Tooltip title={sub} placement="topLeft" mouseEnterDelay={0.2}>
        <div style={{ ...baseCellStyle, color: 'rgba(0,0,0,0.45)', fontSize: 12, marginTop: 2 }}>
          {sub}
        </div>
      </Tooltip>
    </div>
  );
}

export function renderTaskStatus(status: UnifiedFileTransferTask['status'], t: Translate) {
  switch (status) {
    case 'in_progress':
      return <Badge status="processing" text={t('ufte.status.inProgress')} />;
    case 'suspended':
      return <Badge status="warning" text={t('ufte.status.suspended')} />;
    case 'ended':
      return <Badge status="success" text={t('ufte.status.ended')} />;
    default:
      return <Badge status="default" text={t('ufte.status.pending')} />;
  }
}

type BadgeStatus = 'success' | 'processing' | 'default' | 'error' | 'warning';

function renderNoWrapDeviceStatus(status: BadgeStatus, text: ReactNode) {
  return (
    <span
      data-testid="ufte-device-status"
      style={{ display: 'inline-flex', alignItems: 'center', whiteSpace: 'nowrap' }}
    >
      <Badge status={status} text={text} />
    </span>
  );
}

export function renderDeviceStatus(status: UnifiedFileTransferDeviceItem['status'], t: Translate) {
  switch (status) {
    case 'downloading':
      // 升级类 Download RPC：CPE 正在从 ACS 拉镜像 / 补丁文件
      return renderNoWrapDeviceStatus('processing', t('ufte.status.downloading'));
    case 'rollback_checking':
      return renderNoWrapDeviceStatus('processing', t('ufte.status.rollbackChecking'));
    case 'rolling_back':
      return renderNoWrapDeviceStatus('processing', t('ufte.status.rollingBack'));
    case 'uploading':
      // 备份 / 日志采集 Upload RPC：Upload 命令已派发，等 UploadResponse + CPE 通过 HTTP PUT
      // 把文件上传到 ACS（这两步在 TR-069 上紧贴，ACS 侧合并到同一段展示）
      return renderNoWrapDeviceStatus('processing', t('ufte.status.uploading'));
    case 'awaiting_tc':
      // 备份 / 日志采集 Upload RPC：文件已落到 ACS MinIO（backup_restore_file 已 upsert），
      // 等 CPE 主动发 TransferComplete SOAP 来结束传输事务
      return renderNoWrapDeviceStatus('processing', t('ufte.status.awaitingTc'));
    case 'verifying':
      return renderNoWrapDeviceStatus('processing', t('ufte.status.verifying'));
    // 设备子任务的 'suspended' 既可能是"用户挂起创建"也可能是"设备离线等待"，
    // 后者占比更高（设备 inform 间隔 5 min，挂起→开始时常碰到设备短暂掉线）。
    // 合并文案为"已挂起 / 待上线"，避免用户以为操作未生效。详见
    // docs/project/backup-display-fix-20260520.md F11。
    case 'suspended':
      return renderNoWrapDeviceStatus('warning', t('ufte.status.suspendedOrOffline'));
    case 'ended':
      return renderNoWrapDeviceStatus('success', t('ufte.status.completed'));
    case 'failed':
      return renderNoWrapDeviceStatus('error', t('ufte.status.failed'));
    default:
      return renderNoWrapDeviceStatus('default', t('ufte.status.pending'));
  }
}
