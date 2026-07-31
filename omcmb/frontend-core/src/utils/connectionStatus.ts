export function normalizeConnectionStatus(
  status?: string | null,
  partialAsConnected = false,
): string {
  switch (status?.trim().toLowerCase()) {
    case 'connected':
      return 'connected';
    case 'partial':
      return partialAsConnected ? 'connected' : '';
    case 'disconnected':
      return 'disconnected';
    default:
      return '';
  }
}

export function connectionStatusMessageId(
  status?: string | null,
  partialAsConnected = false,
): string {
  switch (normalizeConnectionStatus(status, partialAsConnected)) {
    case 'connected':
      return 'status.connected';
    case 'disconnected':
      return 'status.disconnected';
    default:
      return '';
  }
}
