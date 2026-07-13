const contactNumberPattern = /^[0-9+\-()xX. ]+$/;

export function normalizeContactNumber(value: string): string {
  return value.trim();
}

export function isValidContactNumber(value: string): boolean {
  const normalized = normalizeContactNumber(value);
  if (normalized.length < 3 || normalized.length > 32 || !contactNumberPattern.test(normalized)) {
    return false;
  }

  return (normalized.match(/\d/g) ?? []).length >= 3;
}
