export interface TargetFileDisplaySource {
  targetFile?: string;
  downloadUrl?: string;
  fileDeleted?: boolean;
}

export interface TargetFileDisplayState {
  file: string;
  downloadUrl: string;
  tooltip: string;
  deleted: boolean;
}

export function resolveTargetFileDisplay(
  source: TargetFileDisplaySource,
  quotaCleanedTooltip: string,
): TargetFileDisplayState {
  const file = source.targetFile?.trim() ?? '';
  if (!file) {
    return { file: '', downloadUrl: '', tooltip: '', deleted: false };
  }
  if (source.fileDeleted) {
    return { file, downloadUrl: '', tooltip: quotaCleanedTooltip, deleted: true };
  }
  return {
    file,
    downloadUrl: source.downloadUrl?.trim() ?? '',
    tooltip: file,
    deleted: false,
  };
}
