/** Return parent row ids that contain at least one nested child-object instance. */
export function rowsWithNestedInstances(
  paths: Iterable<string>,
  parentPrefix: string,
  rowIds: number[],
  childObjects: string[],
): number[] {
  if (childObjects.length === 0) return [];
  const pathList = Array.from(paths);
  return rowIds.filter((rowId) => childObjects.some((childObject) => {
    const prefix = `${parentPrefix}${rowId}.${childObject}.`;
    return pathList.some((path) => {
      if (!path.startsWith(prefix)) return false;
      const instanceAndLeaf = path.slice(prefix.length);
      const separator = instanceAndLeaf.indexOf('.');
      return separator > 0 && Number.isInteger(Number(instanceAndLeaf.slice(0, separator)));
    });
  }));
}
