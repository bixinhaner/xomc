const ACCESS_TABS = ['states', 'policies', 'lists', 'candidates', 'actions'] as const;
const OPERATORS = ['cmcc', 'ctcc', 'cucc'] as const;
const REVIEW_STATUSES = ['pending', 'approved', 'rejected', 'expired'] as const;

export type AccessTab = (typeof ACCESS_TABS)[number];
export type AccessOperator = (typeof OPERATORS)[number];
export type CandidateReviewStatus = (typeof REVIEW_STATUSES)[number];

export interface AccessControlDeepLink {
  tab: AccessTab;
  operator: AccessOperator;
  reviewStatus: CandidateReviewStatus;
  candidateId?: string;
}

const UUID_PATTERN = /^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i;

function allowed<T extends string>(value: string | null, values: readonly T[], fallback: T): T {
  return value && values.includes(value as T) ? (value as T) : fallback;
}

export function parseAccessControlDeepLink(search: URLSearchParams): AccessControlDeepLink {
  const candidateId = search.get('candidateId')?.trim();
  return {
    tab: allowed(search.get('tab'), ACCESS_TABS, 'states'),
    operator: allowed(search.get('operator'), OPERATORS, 'cmcc'),
    reviewStatus: allowed(search.get('reviewStatus'), REVIEW_STATUSES, 'pending'),
    ...(candidateId && UUID_PATTERN.test(candidateId) ? { candidateId } : {}),
  };
}

export function updateAccessControlSearch(
  current: URLSearchParams,
  patch: Partial<Record<'tab' | 'operator' | 'reviewStatus' | 'candidateId', string | undefined>>,
): URLSearchParams {
  const next = new URLSearchParams(current);
  Object.entries(patch).forEach(([key, value]) => {
    if (value) next.set(key, value);
    else next.delete(key);
  });
  return next;
}
