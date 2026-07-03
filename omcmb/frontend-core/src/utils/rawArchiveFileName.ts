export function resolveRawArchiveDisplayFileName(fileName: string, minioPath?: string): string {
  const storedPath = minioPath || '';
  if (storedPath.toLowerCase().endsWith('.gz') && !fileName.toLowerCase().endsWith('.gz')) {
    return `${fileName}.gz`;
  }
  return fileName;
}
