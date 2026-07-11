import { describe, expect, it } from 'vitest';

import { clampAgentPanelWidth, getAgentPanelWidthBounds } from './useAgentPanelLayout';

describe('agent panel layout sizing', () => {
  it('clamps panel width inside configured bounds', () => {
    const bounds = { min: 360, max: 760 };

    expect(clampAgentPanelWidth(320, bounds)).toBe(360);
    expect(clampAgentPanelWidth(480, bounds)).toBe(480);
    expect(clampAgentPanelWidth(900, bounds)).toBe(760);
  });

  it('uses viewport and margin to compute max width', () => {
    const bounds = getAgentPanelWidthBounds(1200, {
      minWidth: 360,
      maxWidth: 760,
      expandedRatio: 0.68,
      horizontalMargin: 32,
    });

    expect(bounds.min).toBe(360);
    expect(bounds.max).toBe(760);
    expect(bounds.expanded).toBe(760);
    expect(bounds.compact).toBe(false);
  });

  it('marks narrow viewports compact and allows full available width', () => {
    const bounds = getAgentPanelWidthBounds(390, {
      minWidth: 360,
      maxWidth: 760,
      horizontalMargin: 32,
      mobileBreakpoint: 768,
    });

    expect(bounds.min).toBe(358);
    expect(bounds.max).toBe(358);
    expect(bounds.compact).toBe(true);
  });
});
