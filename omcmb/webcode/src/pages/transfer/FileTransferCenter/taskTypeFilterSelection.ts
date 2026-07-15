interface TaskTypeFilterLike {
  typeCode: string;
}

export function resolveTaskTypeFilterValue(
  taskTypes: readonly TaskTypeFilterLike[],
  selectedTypeCode: string | undefined,
): string | undefined {
  if (!selectedTypeCode) {
    return undefined;
  }
  return taskTypes.some((item) => item.typeCode === selectedTypeCode)
    ? selectedTypeCode
    : undefined;
}

export function shouldShowTaskTypeFilter(taskTypes: readonly TaskTypeFilterLike[]): boolean {
  return taskTypes.length > 1;
}
