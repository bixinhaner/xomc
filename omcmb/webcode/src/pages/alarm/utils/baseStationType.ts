export const BASE_STATION_TYPE_OPTIONS = [
	{ label: 'eNB (LTE)', value: 'eNB' },
	{ label: 'gNB (NR)', value: 'gNB' },
	{ label: 'GSM', value: 'GSM' },
] as const;

const BASE_STATION_TYPE_LABELS: Record<string, string> = {
	eNB: 'eNB (LTE)',
	lte: 'eNB (LTE)',
	LTE: 'eNB (LTE)',
	gNB: 'gNB (NR)',
	nr: 'gNB (NR)',
	NR: 'gNB (NR)',
	'5G NR': 'gNB (NR)',
	GSM: 'GSM',
	gsm: 'GSM',
};

export function formatBaseStationTypeLabel(value?: string | null): string {
	if (!value) {
		return '-';
	}

	// Try exact match first
	if (BASE_STATION_TYPE_LABELS[value]) {
		return BASE_STATION_TYPE_LABELS[value];
	}

	// Try case-insensitive match as fallback
	const lowerKey = value.toLowerCase();
	const matchedKey = Object.keys(BASE_STATION_TYPE_LABELS).find(
		key => key.toLowerCase() === lowerKey
	);
	if (matchedKey) {
		return BASE_STATION_TYPE_LABELS[matchedKey];
	}

	// Return original value if no match found
	return value;
}