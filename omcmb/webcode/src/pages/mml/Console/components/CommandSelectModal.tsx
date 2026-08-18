import { useMemo, useState } from 'react';
import { Button, Empty, Input, Modal, Space, Spin, Tag, Tooltip, Tree, Typography } from 'antd';
import type { TreeDataNode } from 'antd';
import { RightOutlined } from '@ant-design/icons';
import {
  useCommandSubFields,
  useCustomCommandPaths,
  useGroupTree,
  useUnsupportedPaths,
} from '@core/hooks/api/useMmlConsole';
import type { MMLCustomCommand } from '@core/types/mml';
import { useI18nText } from '@/hooks/useI18nText';
import { useT } from '@/hooks/useT';
import {
  CUSTOM_KEY_PREFIX,
  CUSTOM_ROOT_KEY,
  buildCustomizedSubtree,
  parseCustomLeafId,
  useCustomCommands,
} from '../../components/customizedSubtree';
import type { CommandItem } from '../types';
import { COMMAND_MODAL_BODY_HEIGHT, opColor } from '../constants';
import {
  commandUsesPathSelection,
  getDefaultSelectedPathKeys,
  getOrderedSelectedPathKeys,
  getSelectableCommandPaths,
} from '../pathSelection';
import {
  customCommandPathDefsToParamPaths,
  customCommandParamPaths,
  flattenGroupTree,
  mapCommandItem,
  mapCustomCommandItem,
  subFieldsToParamPaths,
} from '../adapters';
import CommandPathSelector from './CommandPathSelector';

const { Text } = Typography;

interface CommandSelectModalProps {
  open: boolean;
  value: CommandItem | null;
  selectedPathKeys: string[];
  onCancel: () => void;
  onConfirm: (command: CommandItem, selectedPathKeys: string[]) => void;
  /** 「指定参数」快捷入口：跳过命令选择，直接进入「配置参数」弹框的「指定参数」标签（裸路径专家模式）。 */
  onGotoRawParams: () => void;
  /**
   * 当前已选设备 SN（取首个）。传入后按该设备 product_class → paramModel 过滤参数 PATH，
   * 设备模型不支持的 path 不再展示（选择命令右侧预览 + 配置参数勾选列表均生效）。
   */
  deviceSn?: string;
  /** 当前所选产品类型。优先用于命令树 / 参数列表过滤，口径与 param_mappings 执行校验一致。 */
  productClass?: string;
  /**
   * 当前所选产品 ID（设备列表强制同一产品，见 DeviceSelectModal）。用于拉「产品不支持 path
   * 自学习表」，按命令读/写类型过滤：LST/DSP 隐藏 read 不支持的；MOD/ADD/RMV 隐藏 write 不支持的。
   */
  productId?: string;
}

/**
 * 命令选择弹框（设计 §3.3.2 + §3.12）—— 左侧两层命令树（分组 → 命令）+ 搜索，
 * 右侧选中命令预览（操作类型 / 命令码 / 参数 PATH）。
 * 数据来源：`useGroupTree` 拉层级命令树拍平为二级；选中命令后 `useCommandSubFields`
 * 拉该命令的参数路径（决定结果表格列 + 可写项）。
 *
 * 自定义命令：`useCustomCommands` 拉管理员自定义命令（mml_custom_command），合并为一个扁平
 * 「自定义命令(Customized)」分组（只读，无新增/编辑）；选中后不调 sub-fields 端点，直接用
 * 其 paramPaths 作为参数路径。命令名为单语中文，分组名随 locale 中英切换。
 */
export default function CommandSelectModal({
  open,
  value,
  onCancel,
  onConfirm,
  onGotoRawParams,
  deviceSn,
  productClass,
  productId,
}: CommandSelectModalProps) {
  const { locale } = useI18nText();
  const t = useT();
  const [keyword, setKeyword] = useState('');
  const [selectedId, setSelectedId] = useState<string | undefined>();
  const [wasOpen, setWasOpen] = useState(false);
  const [draftPathKeys, setDraftPathKeys] = useState<string[]>([]);
  const [selectionEpoch, setSelectionEpoch] = useState(0);
  const [draftSelectionEpoch, setDraftSelectionEpoch] = useState<number | undefined>();
  // §需求 B1：默认所有命令分组折叠。expandedKeys 由用户手动展开累积；搜索时另行整树展开。
  const [expandedKeys, setExpandedKeys] = useState<string[]>([]);

  // 打开时回填（渲染阶段调整 state，避开 set-state-in-effect）。
  if (open !== wasOpen) {
    setWasOpen(open);
    if (open) {
      setKeyword('');
      setExpandedKeys([]); // 每次打开都重置为全部折叠
      // 自定义命令叶子 key 带 custom: 前缀，回填选中态时需还原前缀，否则匹配不到树节点。
      setSelectedId(value ? (value.isCustom ? `${CUSTOM_KEY_PREFIX}${value.id}` : value.id) : undefined);
      setSelectionEpoch((epoch) => epoch + 1);
      setDraftSelectionEpoch(undefined);
      setDraftPathKeys([]);
    }
  }

  // 命令树和右侧 sub-fields 使用同一个产品类型支持集合；设备 SN 仅作兼容兜底。
  const { data: treeNodes, isLoading: treeLoading } = useGroupTree(undefined, locale, productClass, deviceSn);
  const { commands: customCommands } = useCustomCommands(productId);

  // 产品参数模型过滤由 /mml/templates?product_id= 完成；这里继续叠加运行时
  // 不支持 path 过滤，确保模板在树上没有任何可执行 path 时直接消失。
  const { data: unsupportedPaths } = useUnsupportedPaths(productId);
  const unsupportedPathsPending = Boolean(productId) && unsupportedPaths === undefined;

  const customCommandsWithAvailablePaths = useMemo(() => {
    if (!unsupportedPaths) return customCommands;
    return customCommands.filter((cc) => {
      const isWrite = cc.operationType === 'MOD' || cc.operationType === 'ADD' || cc.operationType === 'RMV';
      const blocked = new Set(
        unsupportedPaths
          .filter((u) => (isWrite ? u.writeUnsupported : u.readUnsupported))
          .map((u) => u.path),
      );
      return customCommandParamPaths(cc).some((p) => !blocked.has(p.path));
    });
  }, [customCommands, unsupportedPaths]);

  // 自定义命令分组名（随 locale 中英切换，复用命令树「自定义命令 / Customized」语料）。
  const customGroupLabel = t('mml.console.commandTree.customized');

  // 拍平为「分组 → 命令」二级，并建 id → 条目索引（取 groupName / GroupTreeCommand）。
  const entries = useMemo(() => flattenGroupTree(treeNodes ?? []), [treeNodes]);
  const entryById = useMemo(() => {
    const m = new Map<string, (typeof entries)[number]>();
    entries.forEach((e) => m.set(e.command.id, e));
    return m;
  }, [entries]);
  const customById = useMemo(() => {
    const m = new Map<string, MMLCustomCommand>();
    customCommandsWithAvailablePaths.forEach((cc) => m.set(cc.id, cc));
    return m;
  }, [customCommandsWithAvailablePaths]);

  const filtered = useMemo(() => {
    const kw = keyword.trim().toLowerCase();
    if (!kw) return entries;
    return entries.filter((e) =>
      [
        e.command.commandCode,
        e.command.displayName,
        e.groupName,
        e.command.targetObject ?? '',
        ...(e.command.targetPaths ?? []),
      ].some((field) => field.toLowerCase().includes(kw)),
    );
  }, [entries, keyword]);

  const filteredCustoms = useMemo(() => {
    const kw = keyword.trim().toLowerCase();
    if (!kw) return customCommandsWithAvailablePaths;
    return customCommandsWithAvailablePaths.filter((cc) =>
      [
        cc.commandCode,
        cc.commandName,
        customGroupLabel,
        ...(cc.paramPaths ?? []),
      ].some((field) => field.toLowerCase().includes(kw)),
    );
  }, [customCommandsWithAvailablePaths, keyword, customGroupLabel]);

  const treeData: TreeDataNode[] = useMemo(() => {
    const byGroup = new Map<string, typeof entries>();
    filtered.forEach((e) => {
      const arr = byGroup.get(e.groupName) ?? [];
      arr.push(e);
      byGroup.set(e.groupName, arr);
    });
    const groups: TreeDataNode[] = Array.from(byGroup.entries()).map(([group, items]) => ({
      key: `group:${group}`,
      title: group,
      selectable: false,
      children: items.map((e) => {
        // 选中命令名加粗 + 主题色高亮，配合 Tree 选中底色给出明确的点击选中效果（§需求 3）。
        const isSelected = e.command.id === selectedId;
        return {
          key: e.command.id,
          title: (
            <Space size={6}>
              <Tag color={opColor(e.command.operationType)} style={{ marginInlineEnd: 0 }}>
                {e.command.operationType}
              </Tag>
              <span style={{ fontWeight: isSelected ? 600 : undefined, color: isSelected ? '#1677ff' : undefined }}>
                {e.command.displayName}
              </span>
            </Space>
          ),
        };
      }),
    }));

    // §需求 B2：自定义命令子树与 mml/admin/catalog 一致——
    //   私有模板 > 当前账号 > 私有命令；公有模板 > 公有命令；标签随中英切换。
    // 复用 buildCustomizedSubtree（不传 onAdd → 只读，无「+」），叶子可选中。
    // 搜索无自定义命中且无关键字时也保留骨架；纯标准命令搜索时不追加（避免空骨架噪声 + 保留「无匹配」空态）。
    if (!keyword.trim() || filteredCustoms.length > 0) {
      groups.push(
        buildCustomizedSubtree({
          customs: filteredCustoms,
          t,
          leafSelectable: true,
          renderLeafTitle: (cc) => {
            const isSelected = `${CUSTOM_KEY_PREFIX}${cc.id}` === selectedId;
            return (
              <Space size={6}>
                <Tag color={opColor(cc.operationType)} style={{ marginInlineEnd: 0 }}>
                  {cc.operationType}
                </Tag>
                <span style={{ fontWeight: isSelected ? 600 : undefined, color: isSelected ? '#1677ff' : undefined }}>
                  {cc.commandName}
                </span>
              </Space>
            );
          },
        }),
      );
    }
    return groups;
  }, [filtered, filteredCustoms, selectedId, keyword, t]);

  // §需求 B1：默认全部折叠（expandedKeys 初始为空）；搜索时整树展开以便看到命中项。
  const allTreeKeys = useMemo(() => {
    const out: string[] = [];
    const walk = (ns: TreeDataNode[]) => {
      ns.forEach((n) => {
        out.push(n.key as string);
        if (n.children) walk(n.children as TreeDataNode[]);
      });
    };
    walk(treeData);
    return out;
  }, [treeData]);
  const effectiveExpandedKeys = keyword.trim() ? allTreeKeys : expandedKeys;

  // 选中项可能是标准命令（裸 id）或自定义命令（custom:<id>）。
  const isCustomSelected = selectedId?.startsWith(CUSTOM_KEY_PREFIX) ?? false;
  const selectedCustomId = isCustomSelected && selectedId ? parseCustomLeafId(selectedId) : null;
  const selectedCustom = selectedCustomId ? customById.get(selectedCustomId) : undefined;
  const selectedStandardId = !isCustomSelected ? selectedId : undefined;
  const selectedEntry = selectedStandardId ? entryById.get(selectedStandardId) : undefined;

  // 标准命令的参数路径（自定义命令时 selectedStandardId 为空，hook 自动不发请求）。
  const { data: subFields, isFetching: subFieldsLoading } = useCommandSubFields(
    selectedStandardId,
    locale,
    deviceSn,
    productClass,
    open,
  );
  const { data: customPathDefs, isFetching: customPathsLoading } = useCustomCommandPaths(
    selectedCustomId ?? undefined,
  );

  // 该产品执行 path 不支持类故障记录的自学习表——按命令读/写类型过滤（标准 + 自定义共用）。
  const hiddenPaths = useMemo(() => {
    if (!unsupportedPaths || unsupportedPaths.length === 0) return new Set<string>();
    const op = selectedCustom?.operationType ?? selectedEntry?.command.operationType;
    const isWrite = op === 'MOD' || op === 'ADD' || op === 'RMV';
    return new Set(
      unsupportedPaths
        .filter((u) => (isWrite ? u.writeUnsupported : u.readUnsupported))
        .map((u) => u.path),
    );
  }, [unsupportedPaths, selectedCustom, selectedEntry]);

  const visibleSubFields = useMemo(
    () => (subFields ?? []).filter((sf) => !hiddenPaths.has(sf.tr069Path)),
    [subFields, hiddenPaths],
  );
  const customParamPaths = useMemo(
    () => {
      if (!selectedCustom) return [];
      // /mml/templates?product_id= 已把 selectedCustom.paramPaths 裁成当前产品支持集合；
      // 富化端点按命令返回标准元数据，必须与该集合取交集，不能让产品不支持的关联
      // Path 重新进入选择器和执行请求。
      const productSupportedPaths = new Set(
        selectedCustom.paramPaths.map((path) => path.trim()).filter(Boolean),
      );
      return customCommandPathDefsToParamPaths(customPathDefs ?? []).filter(
        (p) => productSupportedPaths.has(p.path) && !hiddenPaths.has(p.path),
      );
    },
    [selectedCustom, customPathDefs, hiddenPaths],
  );

  // ADD/RMV 以「目标对象路径」(target_object)下发 RPC(AddObject/DeleteObject)，无参数 PATH；
  // 仅标准命令带 target_object。有 target_object 即可「确定选择」，不受 paramPaths 为空限制。
  const selOp = selectedEntry?.command.operationType;
  const selTargetObject = selectedEntry?.command.targetObject?.trim() ?? '';
  const isAddRmvWithObject =
    !isCustomSelected && (selOp === 'ADD' || selOp === 'RMV') && selTargetObject !== '';

  // 统一的「当前选中命令的可执行参数路径」+ 加载态（右侧预览与确定按钮共用）。
  const paramPaths = isCustomSelected ? customParamPaths : subFieldsToParamPaths(visibleSubFields);
  const pathsLoading =
    (!isAddRmvWithObject && unsupportedPathsPending) ||
    (isCustomSelected ? customPathsLoading : subFieldsLoading);
  const hasSelection = !!selectedEntry || !!selectedCustom;
  const selectedOperation = selectedCustom?.operationType ?? selectedEntry?.command.operationType;
  const usesPathSelection = commandUsesPathSelection(selectedOperation);
  const selectablePaths = getSelectableCommandPaths(selectedOperation, paramPaths);
  const visiblePathCount = usesPathSelection ? selectablePaths.length : paramPaths.length;
  const defaultPathKeys = getDefaultSelectedPathKeys(selectablePaths);
  const effectiveDraftPathKeys =
    draftSelectionEpoch === selectionEpoch
      ? getOrderedSelectedPathKeys(selectablePaths, draftPathKeys)
      : defaultPathKeys;

  const handleDraftPathKeysChange = (pathKeys: string[]): void => {
    setDraftSelectionEpoch(selectionEpoch);
    setDraftPathKeys(pathKeys);
  };

  // §需求 3：LST/MOD 无可执行 PATH → 禁用；ADD/RMV 看 target_object。
  const okDisabled =
    !hasSelection ||
    pathsLoading ||
    (isAddRmvWithObject ? false : paramPaths.length === 0) ||
    (usesPathSelection && effectiveDraftPathKeys.length === 0);

  const handleOk = (): void => {
    if (pathsLoading) return;
    if (selectedCustom) {
      // 无可执行 path 不允许确认（§需求 3）；按钮已禁用，这里再兜底。
      if (customParamPaths.length === 0) return;
      onConfirm(
        mapCustomCommandItem(selectedCustom, customGroupLabel, customParamPaths),
        usesPathSelection ? effectiveDraftPathKeys : [],
      );
      return;
    }
    if (!selectedEntry || !subFields) return;
    // ADD/RMV 以 target_object 执行(允许空 paramPaths)；LST/MOD 需有可执行 PATH。
    if (!isAddRmvWithObject && paramPaths.length === 0) return;
    onConfirm(
      mapCommandItem(selectedEntry.groupName, selectedEntry.command, visibleSubFields),
      usesPathSelection ? effectiveDraftPathKeys : [],
    );
  };

  return (
    <Modal
      title={
        <Space size={12} align="center">
          <span>{t('mml.consoleV2.cmdSelect.title')}</span>
          {/* 「指定参数」入口：跳过命令选择，直接跳到「配置参数」的指定 PATH 模式（§需求 1/5） */}
          <Tooltip title={t('mml.consoleV2.cmdSelect.gotoRawTip')}>
            <Button type="link" size="small" style={{ padding: 0 }} onClick={onGotoRawParams}>
              {t('mml.consoleV2.cmdSelect.gotoRaw')}
              <RightOutlined style={{ fontSize: 11 }} />
            </Button>
          </Tooltip>
        </Space>
      }
      open={open}
      width={860}
      onCancel={onCancel}
      onOk={handleOk}
      okText={t('mml.consoleV2.cmdSelect.okText')}
      cancelText={t('common.cancel')}
      // §需求 3：LST/MOD 无可执行 PATH 时禁用「确定选择」；ADD/RMV 看 target_object（§需求 1）。
      okButtonProps={{ disabled: okDisabled }}
      destroyOnHidden
    >
      <div style={{ marginBottom: 12 }}>
        <Input.Search
          allowClear
          placeholder={t('mml.consoleV2.cmdSelect.searchPlaceholder')}
          style={{ width: 240 }}
          value={keyword}
          onChange={(e) => setKeyword(e.target.value)}
        />
      </div>
      <div style={{ display: 'flex', gap: 12, height: COMMAND_MODAL_BODY_HEIGHT }}>
        <div style={{ width: '42%', overflow: 'auto', borderRight: '1px solid #f0f0f0', paddingRight: 8 }}>
          {treeLoading ? (
            <div style={{ textAlign: 'center', marginTop: 120 }}>
              <Spin />
            </div>
          ) : treeData.length === 0 ? (
            <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description={t('mml.consoleV2.cmdSelect.noMatch')} />
          ) : (
            <Tree
              blockNode
              treeData={treeData}
              expandedKeys={effectiveExpandedKeys}
              onExpand={(keys) => setExpandedKeys(keys as string[])}
              selectedKeys={selectedId ? [selectedId] : []}
              onSelect={(keys) => {
                const k = keys[0] as string | undefined;
                // 仅命令叶子可选（分组 / 私有公有骨架节点 selectable=false 不会触发，这里再排除前缀兜底）。
                if (k && !k.startsWith('group:') && k !== CUSTOM_ROOT_KEY) {
                  if (k !== selectedId) {
                    setSelectedId(k);
                    setSelectionEpoch((epoch) => epoch + 1);
                    setDraftSelectionEpoch(undefined);
                    setDraftPathKeys([]);
                  }
                }
              }}
            />
          )}
        </div>
        <div style={{ flex: 1, overflow: 'auto' }}>
          {hasSelection ? (
            <Space orientation="vertical" size={10} style={{ width: '100%' }}>
              {pathsLoading ? (
                <Spin size="small" />
              ) : isAddRmvWithObject ? (
                <>
                  {/* §需求 1：ADD/RMV 无参数 PATH，展示执行 RPC 的「目标对象路径」提醒用户。 */}
                  <Text strong>{t('mml.consoleV2.cmdSelect.targetObjectPath')}</Text>
                  <Text type="secondary" style={{ fontSize: 12 }}>
                    {t('mml.consoleV2.cmdSelect.addRmvHint', {
                      op: selOp,
                      rpc: selOp === 'ADD' ? 'AddObject' : 'DeleteObject',
                    })}
                  </Text>
                  <Text code style={{ fontSize: 12, wordBreak: 'break-all' }}>
                    {selTargetObject}
                  </Text>
                </>
              ) : (
                <>
                  <Text strong>{t('mml.consoleV2.cmdSelect.paramPathCount', { count: visiblePathCount })}</Text>
                  {visiblePathCount === 0 ? (
                    <Text type="secondary">{t('mml.consoleV2.cmdSelect.noParamPath')}</Text>
                  ) : usesPathSelection ? (
                    <CommandPathSelector
                      paths={selectablePaths}
                      value={effectiveDraftPathKeys}
                      onChange={handleDraftPathKeysChange}
                    />
                  ) : (
                    <Space orientation="vertical" size={4} style={{ width: '100%' }}>
                      {paramPaths.map((p) => (
                        <div
                          key={p.path}
                          style={{ display: 'flex', alignItems: 'baseline', gap: 8, fontSize: 12, minWidth: 0 }}
                        >
                          {/* PATH 名称固定宽展示，过长截断 + hover Tooltip 显示完整（§3.10.2） */}
                          <Text style={{ flex: '0 0 96px' }} ellipsis={{ tooltip: p.label }}>
                            {p.label}
                          </Text>
                          <Text
                            code
                            style={{ flex: 1, minWidth: 0, fontSize: 12 }}
                            ellipsis={{ tooltip: p.path }}
                          >
                            {p.path}
                          </Text>
                        </div>
                      ))}
                    </Space>
                  )}
                </>
              )}
            </Space>
          ) : (
            <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description={t('mml.consoleV2.cmdSelect.selectToView')} />
          )}
        </div>
      </div>
    </Modal>
  );
}
