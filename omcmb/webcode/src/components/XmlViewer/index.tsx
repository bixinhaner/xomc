/**
 * XmlViewer — 在 modal / drawer / cell 中展示 SOAP/XML 字符串，含：
 *  - 自动缩进（通过 @core/utils/xmlFormat）
 *  - 简单语法着色：标签名 / 属性名 / 属性值 / 注释 / 声明 / cdata
 *  - 一键复制原始 XML（注意：复制的是 props.xml 原文，不是已美化的版本，
 *    用户拿去诊断时保留与 device_tasks.result 完全一致的字节）
 *
 * 不引第三方 highlight 库（react-syntax-highlighter / prismjs），用 CSS 颜色 +
 * 极简正则切词覆盖 99% CWMP / SOAP 场景，bundle 0 增长。
 */
import React, { useCallback, useMemo } from 'react';
import { Button, Empty, Tooltip, message } from 'antd';
import { CopyOutlined } from '@ant-design/icons';
import { formatXml } from '@core/utils/xmlFormat';
import { useIsDark } from '@/hooks/useThemeToken';
import { useT } from '@/hooks/useT';

export interface XmlViewerProps {
  xml: string | null | undefined;
  maxHeight?: number | string;
  /** 隐藏顶部工具栏（如调用方已经在外层挂了复制按钮）*/
  hideToolbar?: boolean;
  style?: React.CSSProperties;
}

const LIGHT_COLORS = {
  bg: '#fafafa',
  border: '#e5e7eb',
  text: '#1f2937',
  tag: '#1d4ed8',
  attrName: '#9333ea',
  attrValue: '#15803d',
  decl: '#6b7280',
  comment: '#9ca3af',
  cdata: '#dc2626',
};

const DARK_COLORS = {
  bg: '#0d1117',
  border: '#30363d',
  text: '#c9d1d9',
  tag: '#79c0ff',
  attrName: '#d2a8ff',
  attrValue: '#a5d6ff',
  decl: '#8b949e',
  comment: '#6e7681',
  cdata: '#ff7b72',
};

export default function XmlViewer({
  xml,
  maxHeight = 480,
  hideToolbar,
  style,
}: XmlViewerProps) {
  const t = useT();
  const isDark = useIsDark();
  const colors = isDark ? DARK_COLORS : LIGHT_COLORS;

  const formatted = useMemo(() => formatXml(xml ?? ''), [xml]);

  const handleCopy = useCallback(async () => {
    if (!xml) return;
    try {
      await navigator.clipboard.writeText(xml);
      void message.success(t('common.copiedToClipboard'));
    } catch {
      void message.error(t('common.copyFailed'));
    }
  }, [xml, t]);

  if (!xml || !formatted) {
    return <Empty description={t('mml.noExecutionResult')} image={Empty.PRESENTED_IMAGE_SIMPLE} />;
  }

  return (
    <div
      style={{
        border: `1px solid ${colors.border}`,
        borderRadius: 6,
        background: colors.bg,
        overflow: 'hidden',
        ...style,
      }}
    >
      {!hideToolbar && (
        <div
          style={{
            display: 'flex',
            justifyContent: 'flex-end',
            padding: '4px 8px',
            borderBottom: `1px solid ${colors.border}`,
            background: isDark ? '#161b22' : '#f3f4f6',
          }}
        >
          <Tooltip title={t('common.copy')}>
            <Button size="small" icon={<CopyOutlined />} onClick={handleCopy}>
              {t('common.copy')}
            </Button>
          </Tooltip>
        </div>
      )}
      <pre
        style={{
          margin: 0,
          padding: 12,
          maxHeight,
          overflow: 'auto',
          fontFamily:
            "'JetBrains Mono', 'SFMono-Regular', Consolas, 'Liberation Mono', Menlo, Courier, monospace",
          fontSize: 12,
          lineHeight: 1.6,
          color: colors.text,
          whiteSpace: 'pre',
        }}
      >
        {formatted.split('\n').map((line, idx) => (
          <div key={idx}>{renderLine(line, colors)}</div>
        ))}
      </pre>
    </div>
  );
}

/**
 * 把一行 XML 文本切成带颜色的 React 片段。容忍多个 tag 出现在同一行（inline
 * text 的 `<v>1</v>` 场景）；不依赖 DOMParser，避免畸形 XML 抛错破坏页面。
 */
function renderLine(line: string, colors: typeof LIGHT_COLORS): React.ReactNode {
  if (!line) return ' '; // 空行占位
  const parts: React.ReactNode[] = [];
  let i = 0;
  let key = 0;
  while (i < line.length) {
    const ch = line[i];
    if (ch !== '<') {
      // 文本节点：吃到下一个 '<'
      const next = line.indexOf('<', i);
      const end = next === -1 ? line.length : next;
      parts.push(line.slice(i, end));
      i = end;
      continue;
    }
    // 标签起始：定位结束 '>'
    let endIdx = -1;
    // 声明 / 注释 / CDATA 整段处理
    if (line.startsWith('<!--', i)) {
      endIdx = line.indexOf('-->', i);
      if (endIdx !== -1) endIdx += 2;
    } else if (line.startsWith('<![CDATA[', i)) {
      endIdx = line.indexOf(']]>', i);
      if (endIdx !== -1) endIdx += 2;
    } else if (line.startsWith('<?', i)) {
      endIdx = line.indexOf('?>', i);
      if (endIdx !== -1) endIdx += 1;
    } else {
      endIdx = line.indexOf('>', i);
    }
    if (endIdx === -1) {
      // 不闭合 → 原样输出剩余部分，避免无限循环
      parts.push(line.slice(i));
      break;
    }
    const tag = line.slice(i, endIdx + 1);
    parts.push(<React.Fragment key={key++}>{renderTag(tag, colors)}</React.Fragment>);
    i = endIdx + 1;
  }
  return parts;
}

function renderTag(tag: string, colors: typeof LIGHT_COLORS): React.ReactNode {
  if (tag.startsWith('<!--')) {
    return <span style={{ color: colors.comment, fontStyle: 'italic' }}>{tag}</span>;
  }
  if (tag.startsWith('<![CDATA[')) {
    return <span style={{ color: colors.cdata }}>{tag}</span>;
  }
  if (tag.startsWith('<?') || tag.startsWith('<!')) {
    return <span style={{ color: colors.decl }}>{tag}</span>;
  }
  // 普通开 / 闭 / 自闭合
  // 结构：<[/]Name[ attr="v" ...][/]>
  const m = /^<(\/)?\s*([^\s/>]+)([^>]*?)(\/?)>$/.exec(tag);
  if (!m) {
    return <span>{tag}</span>;
  }
  const [, slash, name, attrs, selfClose] = m;
  return (
    <>
      <span style={{ color: colors.decl }}>&lt;{slash ?? ''}</span>
      <span style={{ color: colors.tag }}>{name}</span>
      {attrs ? renderAttrs(attrs, colors) : null}
      <span style={{ color: colors.decl }}>{selfClose ?? ''}&gt;</span>
    </>
  );
}

function renderAttrs(attrs: string, colors: typeof LIGHT_COLORS): React.ReactNode {
  // attrs 形如：' xmlns:soap="..." cwmp:id="123"'  ；按 name="value" / name='value' 切
  const parts: React.ReactNode[] = [];
  const re = /(\s+)([^\s=]+)(?:\s*=\s*("([^"]*)"|'([^']*)'))?/g;
  let lastIndex = 0;
  let key = 0;
  let m: RegExpExecArray | null;
  while ((m = re.exec(attrs)) !== null) {
    if (m.index > lastIndex) {
      parts.push(attrs.slice(lastIndex, m.index));
    }
    const [, ws, name, full, dq, sq] = m;
    parts.push(ws);
    parts.push(
      <span key={key++} style={{ color: colors.attrName }}>
        {name}
      </span>,
    );
    if (full) {
      const quote = dq !== undefined ? '"' : "'";
      const value = dq !== undefined ? dq : (sq ?? '');
      parts.push(<span key={key++} style={{ color: colors.decl }}>=</span>);
      parts.push(
        <span key={key++} style={{ color: colors.attrValue }}>
          {quote}
          {value}
          {quote}
        </span>,
      );
    }
    lastIndex = re.lastIndex;
  }
  if (lastIndex < attrs.length) {
    parts.push(attrs.slice(lastIndex));
  }
  return parts;
}
