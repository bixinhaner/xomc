import type {
  AnchorHTMLAttributes,
  HTMLAttributes,
  ImgHTMLAttributes,
  ReactNode,
  TableHTMLAttributes,
} from 'react';
import ReactMarkdown from 'react-markdown';
import remarkGfm from 'remark-gfm';

interface AgentMarkdownProps {
  content: string;
  className?: string;
}

export function AgentMarkdown({ content, className }: AgentMarkdownProps) {
  const normalizedContent = normalizeAgentMarkdown(content);
  return (
    <div className={className ? `agent-render-markdown ${className}` : 'agent-render-markdown'}>
      <ReactMarkdown
        remarkPlugins={[remarkGfm]}
        components={{
          a: AgentMarkdownLink,
          table: AgentMarkdownTable,
          code: AgentMarkdownCode,
          img: AgentMarkdownImage,
          pre: AgentMarkdownPre,
        }}
      >
        {normalizedContent}
      </ReactMarkdown>
    </div>
  );
}

export function normalizeAgentMarkdown(content: string): string {
  return content
    .split(/(```[\s\S]*?```)/g)
    .map((segment) => (segment.startsWith('```') ? segment : normalizeMarkdownText(segment)))
    .join('');
}

function normalizeMarkdownText(content: string): string {
  let next = content.replace(/\r\n?/g, '\n');

  next = next.replace(/^(\s{0,3})(\d{1,2})\.([^\s\d])/gm, '$1$2. $3');
  next = next.replace(/^(\s{0,3})([-*+])([^\s])/gm, '$1$2 $3');
  next = next.replace(/([：:；;。])\s*(\d{1,2})\.([^\s\d])/g, '$1\n\n$2. $3');
  next = next.replace(/([\u4e00-\u9fffA-Za-z）)，,])\s+(\d{1,2})\.([\u4e00-\u9fffA-Za-z（(])/g, '$1\n$2. $3');
  next = next.replace(/([\u4e00-\u9fff）)，,])(\d{1,2})\.([\u4e00-\u9fffA-Za-z（(])/g, '$1\n$2. $3');
  next = next.replace(/([。；;])\s*([-*+])([^\s])/g, '$1\n$2 $3');

  return next;
}

function AgentMarkdownLink(props: AnchorHTMLAttributes<HTMLAnchorElement>) {
  const { href, className, children, ...rest } = props;
  const normalized = typeof href === 'string' ? href.trim() : '';
  if (!/^https?:\/\//i.test(normalized)) {
    return <span className={className}>{children}</span>;
  }
  return (
    <a className={className} href={normalized} target="_blank" rel="noreferrer" {...rest}>
      {children}
    </a>
  );
}

function AgentMarkdownTable(props: TableHTMLAttributes<HTMLTableElement>) {
  const { className, children, ...rest } = props;
  return (
    <div className="agent-render-table-scroll">
      <table className={className ? `agent-render-table ${className}` : 'agent-render-table'} {...rest}>
        {children}
      </table>
    </div>
  );
}

function AgentMarkdownCode(props: HTMLAttributes<HTMLElement> & { children?: ReactNode }) {
  const { className, children, ...rest } = props;
  if (className) {
    return (
      <code className={className} {...rest}>
        {children}
      </code>
    );
  }
  return (
    <code className="agent-render-inline-code" {...rest}>
      {children}
    </code>
  );
}

function AgentMarkdownImage(props: ImgHTMLAttributes<HTMLImageElement>) {
  const { src, alt, className, ...rest } = props;
  const normalizedSrc = typeof src === 'string' ? src.trim() : '';
  if (!isSafeImageSource(normalizedSrc)) {
    return <span className="agent-render-image-missing">{alt || 'Image unavailable'}</span>;
  }
  return (
    <span className="agent-render-image-card">
      <img
        {...rest}
        className={className ? `agent-render-image ${className}` : 'agent-render-image'}
        src={normalizedSrc}
        alt={alt ?? ''}
        loading="lazy"
      />
    </span>
  );
}

function AgentMarkdownPre(props: HTMLAttributes<HTMLPreElement>) {
  const { className, children, ...rest } = props;
  return (
    <pre className={className ? `agent-render-pre ${className}` : 'agent-render-pre'} {...rest}>
      {children}
    </pre>
  );
}

function isSafeImageSource(value: string): boolean {
  if (!value) return false;
  if (/^(https?:|data:image\/|blob:)/i.test(value)) return true;
  return /^\/(?:api|v1|assets)\//i.test(value);
}
