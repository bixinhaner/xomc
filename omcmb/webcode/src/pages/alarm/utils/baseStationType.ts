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

	return BASE_STATION_TYPE_LABELS[value] || value;
}