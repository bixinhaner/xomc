export function scopeSelectedDeviceKeys(selected: string[], eligible: Iterable<string>): string[] {
  const eligibleSet = new Set(eligible);
  return selected.filter((sn) => eligibleSet.has(sn));
}
