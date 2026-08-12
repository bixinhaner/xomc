const FIXED_PROTOCOL = 'http://';
const FIXED_PORT = '8080';

interface EmptyTransferAddress {
  kind: 'empty';
  mode: 'standard';
  raw: '';
  host: '';
}

interface StandardTransferAddress {
  kind: 'standard';
  mode: 'standard';
  raw: string;
  host: string;
}

interface CustomTransferAddress {
  kind: 'custom';
  mode: 'full';
  raw: string;
}

type InvalidTransferAddress =
  | {
    kind: 'invalid';
    mode: 'standard';
    raw: string;
    host: string;
  }
  | {
    kind: 'invalid';
    mode: 'full';
    raw: string;
  };

export type TransferAddressResult =
  | EmptyTransferAddress
  | StandardTransferAddress
  | CustomTransferAddress
  | InvalidTransferAddress;

function isCanonicalIPv4(host: string): boolean {
  const parts = host.split('.');
  if (parts.length !== 4) return false;
  return parts.every((part) => {
    if (!/^\d+$/.test(part)) return false;
    if (part.length > 1 && part.startsWith('0')) return false;
    const octet = Number(part);
    return octet >= 0 && octet <= 255;
  });
}

function isNonCanonicalNumericHost(host: string): boolean {
  return /^[0-9.]+$/.test(host) && !isCanonicalIPv4(host);
}

export function isValidTransferHost(host: string): boolean {
  if (!host || host !== host.trim() || /[\s/?#@\\%]/.test(host)) return false;
  if (host.includes(':')) {
    try {
      const url = new URL(`${FIXED_PROTOCOL}[${host}]:${FIXED_PORT}`);
      return Boolean(url.hostname);
    } catch {
      return false;
    }
  }
  if (/^[0-9.]+$/.test(host)) {
    return isCanonicalIPv4(host);
  }
  return /^[a-zA-Z0-9]([a-zA-Z0-9-]*[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9-]*[a-zA-Z0-9])?)*$/.test(host);
}

export function buildStandardBaseURL(host: string): string {
  const trimmed = host.trim();
  if (!trimmed) return '';
  const urlHost = trimmed.includes(':') ? `[${trimmed}]` : trimmed;
  return `${FIXED_PROTOCOL}${urlHost}:${FIXED_PORT}`;
}

function splitAuthority(authority: string): { host: string; port?: string } | undefined {
  if (!authority || authority.includes('@')) return undefined;

  if (authority.startsWith('[')) {
    const closingBracket = authority.indexOf(']');
    if (closingBracket <= 1) return undefined;
    const host = authority.slice(1, closingBracket);
    const suffix = authority.slice(closingBracket + 1);
    if (!suffix) return { host };
    if (!suffix.startsWith(':')) return undefined;
    return { host, port: suffix.slice(1) };
  }

  const firstColon = authority.indexOf(':');
  if (firstColon < 0) return { host: authority };
  if (firstColon !== authority.lastIndexOf(':')) return undefined;
  return {
    host: authority.slice(0, firstColon),
    port: authority.slice(firstColon + 1),
  };
}

function hasInvalidPort(port: string | undefined): boolean {
  if (port === undefined) return false;
  if (!/^\d+$/.test(port)) return true;
  const numericPort = Number(port);
  return numericPort < 1 || numericPort > 65535;
}

function hasUnsafePath(rawPath: string): boolean {
  try {
    const decodedPath = decodeURIComponent(rawPath);
    if (decodedPath.includes('\\')) return true;
    return decodedPath.split('/').some((segment) => segment === '.' || segment === '..');
  } catch {
    return true;
  }
}

function invalidFull(raw: string): InvalidTransferAddress {
  return { kind: 'invalid', mode: 'full', raw };
}

export function parseTransferAddress(value?: string): TransferAddressResult {
  const raw = value ?? '';
  if (!raw) {
    return { kind: 'empty', mode: 'standard', raw: '', host: '' };
  }

  const standardMatch = raw.match(/^http:\/\/(\[[^\]]+\]|[^/?#:@\\]+):8080$/);
  if (standardMatch) {
    const host = standardMatch[1].replace(/^\[|\]$/g, '');
    if (isValidTransferHost(host)) {
      return { kind: 'standard', mode: 'standard', raw, host };
    }
    return { kind: 'invalid', mode: 'standard', raw, host };
  }

  if (raw !== raw.trim() || raw.includes('\\') || raw.includes('?') || raw.includes('#')) {
    return invalidFull(raw);
  }

  const schemeMatch = raw.match(/^https?:\/\//i);
  if (!schemeMatch) return invalidFull(raw);

  const remainder = raw.slice(schemeMatch[0].length);
  const pathIndex = remainder.indexOf('/');
  const authority = pathIndex >= 0 ? remainder.slice(0, pathIndex) : remainder;
  const rawPath = pathIndex >= 0 ? remainder.slice(pathIndex) : '';
  const authorityParts = splitAuthority(authority);
  if (!authorityParts || hasInvalidPort(authorityParts.port)) return invalidFull(raw);
  if (
    !authorityParts.host
    || /[\s/?#@\\%]/.test(authorityParts.host)
    || isNonCanonicalNumericHost(authorityParts.host)
    || hasUnsafePath(rawPath)
  ) {
    return invalidFull(raw);
  }

  try {
    const url = new URL(raw);
    if (
      (url.protocol !== 'http:' && url.protocol !== 'https:')
      || !url.hostname
      || url.username
      || url.password
      || url.search
      || url.hash
    ) {
      return invalidFull(raw);
    }
  } catch {
    return invalidFull(raw);
  }

  return { kind: 'custom', mode: 'full', raw };
}
