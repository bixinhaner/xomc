import { useMemo, useState } from 'react';
import { Empty, Input, Popover, Tooltip } from 'antd';
import { CloseCircleFilled, SearchOutlined } from '@ant-design/icons';
import { useT } from '@/hooks/useT';
import { iconNames, iconRegistry, resolveIcon } from './icons';
import styles from './IconPicker.module.css';

export interface IconPickerProps {
  /** menus.icon 字段值（antd icon export name，如 'DashboardOutlined'） */
  value?: string;
  /** 选择 / 清除回调；clear 时传 undefined */
  onChange?: (name: string | undefined) => void;
  disabled?: boolean;
  placeholder?: string;
}

/**
 * IconPicker —— 从 100 个 antd Outlined 图标白名单中选择一个。
 *
 * 设计约束（参 docs/prd/system/menu-dynamic-loading.md §1.3 / §4.3.1）：
 * - 所有图标按 16px 渲染（与 Sidebar 实际尺寸一致，所见即所得）
 * - 仅 Outlined 风格（16px 下视觉最稳）
 * - 不允许用户上传自定义 SVG，纯白名单
 */
export default function IconPicker({ value, onChange, disabled, placeholder }: IconPickerProps) {
  const t = useT();
  const [open, setOpen] = useState(false);
  const [keyword, setKeyword] = useState('');

  const SelectedIcon = resolveIcon(value);

  const filtered = useMemo(() => {
    const k = keyword.trim().toLowerCase().replace(/outlined$/, '');
    if (!k) return iconNames;
    return iconNames.filter((name) =>
      name.toLowerCase().replace(/outlined$/, '').includes(k),
    );
  }, [keyword]);

  const handleSelect = (name: string) => {
    onChange?.(name);
    setOpen(false);
    setKeyword('');
  };

  const handleClear: React.MouseEventHandler<HTMLSpanElement> = (e) => {
    e.stopPropagation();
    onChange?.(undefined);
  };

  const handleOpenChange = (next: boolean) => {
    if (disabled) return;
    setOpen(next);
    if (!next) setKeyword('');
  };

  const content = (
    <div className={styles.popover}>
      <Input
        prefix={<SearchOutlined />}
        placeholder={t('iconPicker.search')}
        value={keyword}
        onChange={(e) => setKeyword(e.target.value)}
        allowClear
        size="small"
        autoFocus
      />
      <div className={styles.gridWrapper}>
        {filtered.length === 0 ? (
          <Empty
            image={Empty.PRESENTED_IMAGE_SIMPLE}
            description={t('iconPicker.noMatch')}
            className={styles.empty}
          />
        ) : (
          <div className={styles.grid}>
            {filtered.map((name) => {
              const Icon = iconRegistry[name];
              const selected = name === value;
              return (
                <Tooltip key={name} title={name} placement="top" mouseEnterDelay={0.3}>
                  <button
                    type="button"
                    className={`${styles.cell} ${selected ? styles.cellSelected : ''}`}
                    onClick={() => handleSelect(name)}
                  >
                    <Icon style={{ fontSize: 16 }} />
                  </button>
                </Tooltip>
              );
            })}
          </div>
        )}
      </div>
      <div className={styles.footer}>
        <span>{t('iconPicker.totalCount', { count: iconNames.length })}</span>
        {filtered.length !== iconNames.length && (
          <span className={styles.matchCount}>
            {t('iconPicker.matchCount', { count: filtered.length })}
          </span>
        )}
      </div>
    </div>
  );

  return (
    <Popover
      content={content}
      trigger="click"
      open={disabled ? false : open}
      onOpenChange={handleOpenChange}
      placement="bottomLeft"
      destroyOnHidden
    >
      <Input
        readOnly
        disabled={disabled}
        placeholder={placeholder ?? t('iconPicker.placeholder')}
        value={value ?? ''}
        prefix={
          SelectedIcon ? (
            <SelectedIcon style={{ fontSize: 16 }} />
          ) : (
            <span className={styles.prefixPlaceholder} />
          )
        }
        suffix={
          value ? (
            <CloseCircleFilled
              onClick={handleClear}
              className={styles.clearIcon}
              aria-label={t('iconPicker.clear')}
            />
          ) : null
        }
        className={styles.trigger}
      />
    </Popover>
  );
}
