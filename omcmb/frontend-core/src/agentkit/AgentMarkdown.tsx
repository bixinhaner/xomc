import type {
  AnchorHTMLAttributes,
  HTMLAttributes,
  ImgHTMLAttributes,
  ReactNode,
  TableHTMLAttributes,
} from 'react';
import { isValidElement, useEffect, useId, useMemo, useRef, useState } from 'react';
import ReactMarkdown from 'react-markdown';
import rehypeKatex from 'rehype-katex';
import remarkGfm from 'remark-gfm';
import remarkMath from 'remark-math';
import 'katex/dist/katex.min.css';

interface AgentMarkdownProps {
  content: string;
  className?: string;
}

interface MermaidModule {
  initialize(config: Record<string, unknown>): void;
  render(id: string, text: string): Promise<{ svg: string; bindFunctions?: ((element: Element) => void) | undefined }>;
}

let mermaidModulePromise: Promise<MermaidModule> | null = null;
let mermaidInitialized = false;
let mermaidRenderSequence = 0;

export function AgentMarkdown({ content, className }: AgentMarkdownProps) {
  return (
    <div className={className ? `agent-render-markdown ${className}` : 'agent-render-markdown'}>
      <ReactMarkdown
        remarkPlugins={[remarkGfm, remarkMath]}
        rehypePlugins={[rehypeKatex]}
        components={{
          a: AgentMarkdownLink,
          table: AgentMarkdownTable,
          code: AgentMarkdownCode,
          img: AgentMarkdownImage,
          pre: AgentMarkdownPre,
        }}
      >
        {content}
      </ReactMarkdown>
    </div>
  );
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
  const mermaidCode = extractMermaidCodeFromPreChildren(children);
  if (mermaidCode) return <AgentMermaidBlock code={mermaidCode} />;
  return (
    <pre className={className ? `agent-render-pre ${className}` : 'agent-render-pre'} {...rest}>
      {children}
    </pre>
  );
}

function AgentMermaidBlock(props: { code: string }) {
  const canvasRef = useRef<HTMLDivElement | null>(null);
  const bindFunctionsRef = useRef<((element: Element) => void) | undefined>(undefined);
  const reactId = useId();
  const blockId = useMemo(() => `agent-mermaid-${reactId}`, [reactId]);
  const [state, setState] = useState<{
    status: 'loading' | 'ready' | 'error';
    svg: string;
    error: string;
  }>({ status: 'loading', svg: '', error: '' });

  useEffect(() => {
    let active = true;
    bindFunctionsRef.current = undefined;
    setState({ status: 'loading', svg: '', error: '' });

    if (typeof window === 'undefined') {
      return () => {
        active = false;
      };
    }

    void loadMermaid()
      .then(async (mermaid) => {
        const result = await mermaid.render(nextMermaidRenderId(blockId), props.code);
        if (!active) return;
        bindFunctionsRef.current = result.bindFunctions;
        setState({ status: 'ready', svg: result.svg, error: '' });
      })
      .catch((error: unknown) => {
        if (!active) return;
        setState({
          status: 'error',
          svg: '',
          error: error instanceof Error ? error.message : 'Mermaid render failed.',
        });
      });

    return () => {
      active = false;
    };
  }, [blockId, props.code]);

  useEffect(() => {
    if (state.status !== 'ready' || !canvasRef.current || !bindFunctionsRef.current) return;
    bindFunctionsRef.current(canvasRef.current);
  }, [state.status, state.svg]);

  if (state.status === 'error') {
    return (
      <div className="agent-render-mermaid agent-render-mermaid-error" role="img" aria-label="Mermaid diagram failed">
        <p>Mermaid render failed</p>
        <pre>{state.error}</pre>
      </div>
    );
  }

  return (
    <div className="agent-render-mermaid" data-status={state.status}>
      {state.status === 'ready' ? (
        <div
          ref={canvasRef}
          className="agent-render-mermaid-canvas"
          role="img"
          aria-label="Mermaid diagram"
          dangerouslySetInnerHTML={{ __html: state.svg }}
        />
      ) : (
        <div className="agent-render-mermaid-loading">Rendering diagram...</div>
      )}
    </div>
  );
}

function extractMermaidCodeFromPreChildren(children: ReactNode): string | null {
  const codeNode = Array.isArray(children) ? children[0] : children;
  if (!isValidElement(codeNode)) return null;
  const props = codeNode.props as { className?: unknown; children?: ReactNode };
  const className = typeof props.className === 'string' ? props.className : '';
  if (!/\blanguage-mermaid\b/.test(className)) return null;
  const code = flattenNodeText(props.children).replace(/\n$/, '');
  return code.trim() ? code : null;
}

function flattenNodeText(value: ReactNode): string {
  if (value === null || value === undefined || typeof value === 'boolean') return '';
  if (typeof value === 'string' || typeof value === 'number') return String(value);
  if (Array.isArray(value)) return value.map((item) => flattenNodeText(item)).join('');
  if (isValidElement(value)) {
    return flattenNodeText((value.props as { children?: ReactNode }).children);
  }
  return '';
}

async function loadMermaid(): Promise<MermaidModule> {
  if (!mermaidModulePromise) {
    mermaidModulePromise = import('mermaid').then((module) => {
      const resolved = (module.default ?? module) as MermaidModule;
      if (!mermaidInitialized) {
        resolved.initialize({
          startOnLoad: false,
          securityLevel: 'strict',
          theme: 'neutral',
          suppressErrorRendering: true,
        });
        mermaidInitialized = true;
      }
      return resolved;
    });
  }
  return mermaidModulePromise;
}

function nextMermaidRenderId(prefix: string): string {
  mermaidRenderSequence += 1;
  return `${prefix.replace(/[^a-zA-Z0-9_-]/g, '')}-${mermaidRenderSequence}`;
}

function isSafeImageSource(value: string): boolean {
  if (!value) return false;
  if (/^(https?:|data:image\/|blob:)/i.test(value)) return true;
  return /^\/(?:api|v1|assets)\//i.test(value);
}
