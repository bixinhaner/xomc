/**
 * xmlFormat — 把 SOAP / XML 字符串重排为带缩进的可读格式。
 *
 * 用途：MML 任务记录的"执行结果"列原始是单行 SOAP envelope（CWMP GPV/SPV
 * Response 等），1KB+ 无换行 → 用户无法阅读。本工具用纯字符串扫描完成缩进，
 * 不引第三方依赖，对标主流 XML beautifier。
 *
 * 行为：
 * - 自闭合标签 `<x/>`、声明 `<?xml ?>`、注释 `<!-- -->`、CDATA `<![CDATA[ ]]>`
 *   按整体单元处理，缩进等级不变
 * - 开标签 `<x>` 后缩进 + 1，闭标签 `</x>` 前缩进 - 1
 * - 单行文本节点（开闭标签夹一段非标签内容）保留在同一行，避免 `<v>1</v>`
 *   被拆成 3 行
 * - 输入若为空 / 完全不像 XML / 解析出错，返回原值（永不抛错，让 UI 兜底）
 *
 * 边界：本函数面向"展示美化"而非"语法校验"。畸形 XML 不会被纠正——只是按
 * 当前 token 序列尽力缩进；用户能复制原始串照样调试。
 */

const INDENT = '  ';

export function formatXml(input: string, options?: { indent?: string }): string {
  if (!input) return '';
  const raw = input.trim();
  if (!raw) return '';
  // 非 XML 形态（一段纯文本 / JSON）直接原样返回，不强行加缩进
  if (!raw.startsWith('<')) return raw;

  const indent = options?.indent ?? INDENT;
  const tokens = tokenize(raw);
  if (tokens.length === 0) return raw;

  const out: string[] = [];
  let depth = 0;
  for (let i = 0; i < tokens.length; i += 1) {
    const tok = tokens[i];
    if (tok.kind === 'text') {
      // 文本节点已在 tokenize 阶段过滤掉纯空白；保留到 inline 时另作处理
      continue;
    }
    if (tok.kind === 'close') {
      depth = Math.max(0, depth - 1);
      // inline 优化：上一个输出行就是同名 open，且中间只有一段文本 → 拼一行
      const last = out[out.length - 1];
      const inline = tryInlineClose(last, tok.value);
      if (inline !== null) {
        out[out.length - 1] = inline;
        continue;
      }
      out.push(indent.repeat(depth) + tok.value);
      continue;
    }
    // open / self / decl / comment / cdata
    out.push(indent.repeat(depth) + tok.value);
    if (tok.kind === 'open') depth += 1;
    // 紧跟一段非空白 text → 暂存到行尾，留给下一轮 inline 判定
    const next = tokens[i + 1];
    if (
      tok.kind === 'open' &&
      next?.kind === 'text' &&
      next.value.length > 0
    ) {
      out[out.length - 1] += escapeForRender(next.value);
      // 跳过 text，并在下一轮处理 close 时做 inline
      i += 1;
    }
  }
  return out.join('\n');
}

interface XmlToken {
  kind: 'open' | 'close' | 'self' | 'decl' | 'comment' | 'cdata' | 'text';
  value: string;
}

function tokenize(src: string): XmlToken[] {
  const tokens: XmlToken[] = [];
  let i = 0;
  while (i < src.length) {
    if (src[i] !== '<') {
      // 文本节点：吃到下一个 '<'
      const next = src.indexOf('<', i);
      const end = next === -1 ? src.length : next;
      const text = src.slice(i, end);
      const trimmed = text.replace(/\s+/g, ' ').trim();
      if (trimmed.length > 0) {
        tokens.push({ kind: 'text', value: trimmed });
      }
      i = end;
      continue;
    }
    // 各类标签
    if (src.startsWith('<?', i)) {
      const end = src.indexOf('?>', i);
      if (end === -1) return [{ kind: 'text', value: src }];
      tokens.push({ kind: 'decl', value: src.slice(i, end + 2) });
      i = end + 2;
      continue;
    }
    if (src.startsWith('<!--', i)) {
      const end = src.indexOf('-->', i);
      if (end === -1) return [{ kind: 'text', value: src }];
      tokens.push({ kind: 'comment', value: src.slice(i, end + 3) });
      i = end + 3;
      continue;
    }
    if (src.startsWith('<![CDATA[', i)) {
      const end = src.indexOf(']]>', i);
      if (end === -1) return [{ kind: 'text', value: src }];
      tokens.push({ kind: 'cdata', value: src.slice(i, end + 3) });
      i = end + 3;
      continue;
    }
    if (src.startsWith('<!', i)) {
      // <!DOCTYPE ...>
      const end = src.indexOf('>', i);
      if (end === -1) return [{ kind: 'text', value: src }];
      tokens.push({ kind: 'decl', value: src.slice(i, end + 1) });
      i = end + 1;
      continue;
    }
    // 常规标签：< ... >  ; 注意属性里可能含 '>'? XML 不允许未转义的 '>'，按
    // 首个 '>' 收尾即可（CWMP / SOAP 一向规范）
    const end = src.indexOf('>', i);
    if (end === -1) return [{ kind: 'text', value: src }];
    const tag = src.slice(i, end + 1);
    if (tag.startsWith('</')) {
      tokens.push({ kind: 'close', value: tag });
    } else if (tag.endsWith('/>')) {
      tokens.push({ kind: 'self', value: tag });
    } else {
      tokens.push({ kind: 'open', value: tag });
    }
    i = end + 1;
  }
  return tokens;
}

function tryInlineClose(lastLine: string | undefined, closeTag: string): string | null {
  if (!lastLine) return null;
  // closeTag 形如 </Name>  → 取 Name
  const m = /^<\/\s*([^\s>]+)\s*>$/.exec(closeTag);
  if (!m) return null;
  const name = m[1];
  // 上一行必须以 <Name ...> 开头（同名 open），且后面有非空内容（即 inline text）
  const openRe = new RegExp(`^(\\s*)<${escapeRegExp(name)}([\\s>][^]*)$`);
  const lm = openRe.exec(lastLine);
  if (!lm) return null;
  // 若上一行已经被 inline（含 '>' 后接非空内容），拼接 close
  const trailing = lm[2];
  // trailing 中第一个 '>' 后必须不是再开新标签
  const gtIdx = trailing.indexOf('>');
  if (gtIdx === -1) return null;
  const afterGt = trailing.slice(gtIdx + 1);
  if (afterGt.includes('<')) return null;
  return `${lastLine}${closeTag}`;
}

function escapeForRender(text: string): string {
  // 这里不做 HTML 转义（输出仍是 XML 文本，调用方决定如何渲染）；
  // 仅折叠多余空白避免行间垃圾
  return text;
}

function escapeRegExp(s: string): string {
  return s.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
}
