/**
 * UnitsDrawer — T-0180 P4 决策 D4:单位定义改为右侧抽屉 width=720,
 * 不再占用顶层 Tab 空间(单位是跨制式共享资源,放主流程入口不经济)。
 *
 * 内容主体直接复用既有 IndicatorUnitsTab 的 CRUD 列表,
 * 仅包装一层 Drawer 控制开关。
 */
import { Drawer } from 'antd';
import IndicatorUnitsTab from './IndicatorUnitsTab';
import { useT } from '@/hooks/useT';

interface Props {
  open: boolean;
  onClose: () => void;
}

export default function UnitsDrawer({ open, onClose }: Props) {
  const t = useT();
  return (
    <Drawer
      title={t('product.kpi.units.short')}
      placement="right"
      width={720}
      open={open}
      onClose={onClose}
      destroyOnHidden
    >
      <IndicatorUnitsTab />
    </Drawer>
  );
}
