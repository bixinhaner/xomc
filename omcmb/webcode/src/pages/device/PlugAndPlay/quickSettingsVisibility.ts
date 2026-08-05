export function isIpsecParametersVisible(value: unknown): boolean {
  return value === true || String(value ?? '') === '1';
}

export function isPtpDetailsVisible(mode: unknown): boolean {
  return String(mode ?? '') === '1588_PPS';
}
