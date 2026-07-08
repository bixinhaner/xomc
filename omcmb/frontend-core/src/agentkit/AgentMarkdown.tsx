import type { AnchorHTMLAttributes, HTMLAttributes, ReactNode, TableHTMLAttributes } from 'react';
import ReactMarkdown from 'react-markdown';
import remarkGfm from 'remark-gfm';

interface AgentMarkdownProps {
  content: string;
  className?: string;
}

export function AgentMarkdown({ content, className }: AgentMarkdownProps) {
  return (
    <div className={className}>
      <ReactMarkdown
        remarkPlugins={[remarkGfm]}
        components={{
          a: AgentMarkdownLink,
          table: AgentMarkdownTable,
          code: AgentMarkdownCode,
        }}
      >
        {content}
      </ReactMarkdown>
    </div>
  );
}

function AgentMarkdownLink(props: AnchorHTMLAttributes<HTMLAnchorElement>) {
  const { href, children, ...rest } = props;
  const normalized = typeof href === 'string' ? href.trim() : '';
  if (!/^https?:\/\//i.test(normalized)) {
    return <span>{children}</span>;
  }
  return (
    <a href={normalized} target="_blank" rel="noreferrer" {...rest}>
      {children}
    </a>
  );
}

function AgentMarkdownTable(props: TableHTMLAttributes<HTMLTableElement>) {
  const { children, ...rest } = props;
  return (
    <div style={{ maxWidth: '100%', overflowX: 'auto' }}>
      <table {...rest}>{children}</table>
    </div>
  );
}

function AgentMarkdownCode(props: HTMLAttributes<HTMLElement> & { children?: ReactNode }) {
  const { children, ...rest } = props;
  return <code {...rest}>{children}</code>;
}
