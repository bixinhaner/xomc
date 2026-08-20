export function normalizeOriginalVersionsForSubmit(
  specifyVersionType: unknown,
  originalVersion: unknown,
): string[] {
  if (specifyVersionType === 'all') return ['all'];
  const rawVersions = Array.isArray(originalVersion)
    ? originalVersion
    : String(originalVersion ?? '').split(/[,，;；\n\r]+/);
  return Array.from(
    new Set(rawVersions.map((version) => String(version).trim()).filter(Boolean)),
  );
}
