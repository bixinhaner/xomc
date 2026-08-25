import { describe, expect, it } from 'vitest';
import { parseAccessControlDeepLink, updateAccessControlSearch } from './deepLink';

describe('AccessControl deep links', () => {
  it('parses a candidate review target', () => {
    expect(parseAccessControlDeepLink(new URLSearchParams(
      'tab=candidates&operator=ctcc&reviewStatus=pending&candidateId=892d12c0-ec1a-4fd1-8070-902d3aaf84e9',
    ))).toEqual({
      tab: 'candidates',
      operator: 'ctcc',
      reviewStatus: 'pending',
      candidateId: '892d12c0-ec1a-4fd1-8070-902d3aaf84e9',
    });
  });

  it('fails closed to safe defaults for unsupported values', () => {
    expect(parseAccessControlDeepLink(new URLSearchParams(
      'tab=admin&operator=unknown&reviewStatus=deleted&candidateId=not-a-uuid',
    ))).toEqual({ tab: 'states', operator: 'cmcc', reviewStatus: 'pending' });
  });

  it('removes only the handled candidate while preserving page context', () => {
    const current = new URLSearchParams('tab=candidates&operator=cucc&reviewStatus=pending&candidateId=892d12c0-ec1a-4fd1-8070-902d3aaf84e9&from=dashboard');
    expect(updateAccessControlSearch(current, { candidateId: undefined }).toString()).toBe(
      'tab=candidates&operator=cucc&reviewStatus=pending&from=dashboard',
    );
  });
});
