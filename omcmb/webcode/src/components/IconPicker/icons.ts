// 图标白名单：menus.icon 字段允许的取值，与 @ant-design/icons 的 export name 一一对应。
// 完整清单冻结在 docs/prd/system/menu-dynamic-loading.md §附录 A（v0.3）。
//
// 维护规则：
// - 禁止任意添加：新增图标必须先改 PRD 附录 A，再改本文件
// - 禁止 Filled / TwoTone（16px 渲染视觉不达标，参 PRD §1.3）
// - 总量上限 200，超过需重构为 lazy load
import {
  // ─── A.1 现有 NavMenu 必含（24）──────────────────────────────────
  DashboardOutlined, ClusterOutlined, AlertOutlined, SettingOutlined,
  LineChartOutlined, CodeOutlined, GlobalOutlined, SaveOutlined,
  CloudUploadOutlined, FolderOutlined, FileTextOutlined, ToolOutlined,
  BarChartOutlined, RadarChartOutlined, SafetyOutlined, AppstoreOutlined,
  GatewayOutlined, DeploymentUnitOutlined, WifiOutlined, ThunderboltOutlined,
  ApartmentOutlined, CloudServerOutlined, AimOutlined, ExperimentOutlined,
  // ─── A.2 用户/团队类（10）────────────────────────────────────────
  UserOutlined, TeamOutlined, UsergroupAddOutlined, UsergroupDeleteOutlined,
  UserAddOutlined, UserDeleteOutlined, IdcardOutlined, SolutionOutlined,
  ContactsOutlined, CrownOutlined,
  // ─── A.3 安全/权限类（8）────────────────────────────────────────
  LockOutlined, UnlockOutlined, KeyOutlined, SafetyCertificateOutlined,
  SecurityScanOutlined, AuditOutlined, EyeOutlined, EyeInvisibleOutlined,
  // ─── A.4 通信/网络类（10）───────────────────────────────────────
  ApiOutlined, LinkOutlined, ShareAltOutlined, BranchesOutlined,
  ForkOutlined, NodeIndexOutlined, SwapOutlined, SyncOutlined,
  RetweetOutlined, BlockOutlined,
  // ─── A.5 存储/数据类（10）───────────────────────────────────────
  DatabaseOutlined, HddOutlined, ContainerOutlined, InboxOutlined,
  BookOutlined, ReadOutlined, ProfileOutlined, SnippetsOutlined,
  CodepenOutlined, BoxPlotOutlined,
  // ─── A.6 文件/操作类（15）───────────────────────────────────────
  EditOutlined, DeleteOutlined, CopyOutlined, ScissorOutlined,
  FileAddOutlined, FileSearchOutlined, FilePdfOutlined, FileExcelOutlined,
  FileImageOutlined, DownloadOutlined, UploadOutlined, SearchOutlined,
  ReloadOutlined, FilterOutlined, SortAscendingOutlined,
  // ─── A.7 通知/状态类（10）───────────────────────────────────────
  BellOutlined, NotificationOutlined, MessageOutlined, MailOutlined,
  SoundOutlined, FlagOutlined, TagOutlined, StarOutlined,
  HeartOutlined, ClockCircleOutlined,
  // ─── A.8 反馈/进度类（5）────────────────────────────────────────
  CheckCircleOutlined, CloseCircleOutlined, InfoCircleOutlined,
  WarningOutlined, QuestionCircleOutlined,
  // ─── A.9 业务/场景类（8）────────────────────────────────────────
  HomeOutlined, ShopOutlined, BankOutlined, RocketOutlined,
  BulbOutlined, FireOutlined, BugOutlined, MedicineBoxOutlined, RobotOutlined,
  // ─── A.10 兼容现网 seed 数据（16）—— 出现在 seed/000057/000059 已用图标
  AreaChartOutlined, BgColorsOutlined, CalendarOutlined, ConsoleSqlOutlined,
  ControlOutlined, FieldNumberOutlined, FileDoneOutlined, FileZipOutlined,
  HistoryOutlined, MenuOutlined, PartitionOutlined, RollbackOutlined,
  ScheduleOutlined, UndoOutlined, UnorderedListOutlined, UpCircleOutlined,
} from '@ant-design/icons';
import type { ComponentType } from 'react';

type IconComponent = ComponentType<{ style?: React.CSSProperties; className?: string }>;

// key 严格对齐 antd export name，作为 menus.icon 字段持久化值。
export const iconRegistry: Record<string, IconComponent> = {
  // A.1
  DashboardOutlined, ClusterOutlined, AlertOutlined, SettingOutlined,
  LineChartOutlined, CodeOutlined, GlobalOutlined, SaveOutlined,
  CloudUploadOutlined, FolderOutlined, FileTextOutlined, ToolOutlined,
  BarChartOutlined, RadarChartOutlined, SafetyOutlined, AppstoreOutlined,
  GatewayOutlined, DeploymentUnitOutlined, WifiOutlined, ThunderboltOutlined,
  ApartmentOutlined, CloudServerOutlined, AimOutlined, ExperimentOutlined,
  // A.2
  UserOutlined, TeamOutlined, UsergroupAddOutlined, UsergroupDeleteOutlined,
  UserAddOutlined, UserDeleteOutlined, IdcardOutlined, SolutionOutlined,
  ContactsOutlined, CrownOutlined,
  // A.3
  LockOutlined, UnlockOutlined, KeyOutlined, SafetyCertificateOutlined,
  SecurityScanOutlined, AuditOutlined, EyeOutlined, EyeInvisibleOutlined,
  // A.4
  ApiOutlined, LinkOutlined, ShareAltOutlined, BranchesOutlined,
  ForkOutlined, NodeIndexOutlined, SwapOutlined, SyncOutlined,
  RetweetOutlined, BlockOutlined,
  // A.5
  DatabaseOutlined, HddOutlined, ContainerOutlined, InboxOutlined,
  BookOutlined, ReadOutlined, ProfileOutlined, SnippetsOutlined,
  CodepenOutlined, BoxPlotOutlined,
  // A.6
  EditOutlined, DeleteOutlined, CopyOutlined, ScissorOutlined,
  FileAddOutlined, FileSearchOutlined, FilePdfOutlined, FileExcelOutlined,
  FileImageOutlined, DownloadOutlined, UploadOutlined, SearchOutlined,
  ReloadOutlined, FilterOutlined, SortAscendingOutlined,
  // A.7
  BellOutlined, NotificationOutlined, MessageOutlined, MailOutlined,
  SoundOutlined, FlagOutlined, TagOutlined, StarOutlined,
  HeartOutlined, ClockCircleOutlined,
  // A.8
  CheckCircleOutlined, CloseCircleOutlined, InfoCircleOutlined,
  WarningOutlined, QuestionCircleOutlined,
  // A.9
  HomeOutlined, ShopOutlined, BankOutlined, RocketOutlined,
  BulbOutlined, FireOutlined, BugOutlined, MedicineBoxOutlined, RobotOutlined,
  // A.10 兼容现网 seed
  AreaChartOutlined, BgColorsOutlined, CalendarOutlined, ConsoleSqlOutlined,
  ControlOutlined, FieldNumberOutlined, FileDoneOutlined, FileZipOutlined,
  HistoryOutlined, MenuOutlined, PartitionOutlined, RollbackOutlined,
  ScheduleOutlined, UndoOutlined, UnorderedListOutlined, UpCircleOutlined,
};

export const iconNames: string[] = Object.keys(iconRegistry);

// 渲染图标的统一入口：找不到时返回 null，调用方负责兜底。
// 同时供 IconPicker 和 NavMenu (P1) 使用。
export function resolveIcon(name: string | undefined | null): IconComponent | null {
  if (!name) return null;
  return iconRegistry[name] ?? null;
}
