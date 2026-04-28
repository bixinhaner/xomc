import { describe, it, expect } from 'vitest';
import { antdClassicTheme } from '../classicTheme';
import { antdTechTheme } from '../techTheme';
import { antdFreshTheme } from '../freshTheme';
import { antdCyberpunkTheme } from '../cyberpunkTheme';
import { antdMinionsTheme } from '../minionsTheme';
import { antdTiffanyTheme } from '../tiffanyTheme';
import { antdRmbTheme } from '../rmbTheme';
import * as tokens from '../tokens';

describe('theme configs', () => {
  // Each themeConfig is the source of truth for a skin variant; we only assert
  // the public shape (token + components), not exhaustive values, so future
  // visual tweaks won't churn this test.
  const themes = {
    classic: antdClassicTheme,
    tech: antdTechTheme,
    fresh: antdFreshTheme,
    cyberpunk: antdCyberpunkTheme,
    minions: antdMinionsTheme,
    tiffany: antdTiffanyTheme,
    rmb: antdRmbTheme,
  };

  it.each(Object.entries(themes))(
    '%s theme exposes a token block',
    (_name, config) => {
      expect(config).toBeDefined();
      expect(config.token).toBeTypeOf('object');
      expect(config.token).not.toBeNull();
    },
  );

  it.each(Object.entries(themes))(
    '%s theme defines a colorPrimary',
    (_name, config) => {
      expect(config.token?.colorPrimary).toMatch(/^#?[0-9a-fA-F]{3,8}$/);
    },
  );

  it.each(Object.entries(themes))(
    '%s theme provides component overrides',
    (_name, config) => {
      expect(config.components).toBeDefined();
      // A skin must at least style Layout + Menu since those drive shell chrome.
      expect(config.components?.Layout).toBeDefined();
      expect(config.components?.Menu).toBeDefined();
    },
  );

  it('classic theme uses primary600 token from design system', () => {
    expect(antdClassicTheme.token?.colorPrimary).toBe(tokens.COLOR_PRIMARY_600);
  });
});

describe('design tokens', () => {
  it('exports primary scale 50-900', () => {
    expect(tokens.COLOR_PRIMARY_50).toBe('#EBF5FF');
    expect(tokens.COLOR_PRIMARY_600).toBe('#1677FF');
    expect(tokens.COLOR_PRIMARY_900).toBe('#002C8C');
  });

  it('exports neutral scale', () => {
    expect(tokens.COLOR_NEUTRAL_50).toBeDefined();
    expect(tokens.COLOR_NEUTRAL_950).toBeDefined();
  });

  it('exports YD/T alarm severities (critical/major/minor/warning)', () => {
    expect(tokens.SEVERITY_CRITICAL).toBe('#F5222D');
    expect(tokens.SEVERITY_MAJOR).toBe('#FA8C16');
    expect(tokens.SEVERITY_MINOR).toBe('#FAAD14');
    expect(tokens.SEVERITY_WARNING).toBe('#1890FF');
  });

  it('exports breakpoint constants', () => {
    expect(tokens.SCREEN_SM).toBe(576);
    expect(tokens.SCREEN_MD).toBe(768);
    expect(tokens.SCREEN_LG).toBe(992);
    expect(tokens.SCREEN_XL).toBe(1200);
    expect(tokens.SCREEN_2XL).toBe(1600);
  });

  it('exports z-index hierarchy in ascending order', () => {
    expect(tokens.Z_CONTENT).toBeLessThan(tokens.Z_FLOATING);
    expect(tokens.Z_FLOATING).toBeLessThan(tokens.Z_SIDEBAR);
    expect(tokens.Z_SIDEBAR).toBeLessThanOrEqual(tokens.Z_HEADER);
    expect(tokens.Z_HEADER).toBeLessThan(tokens.Z_TASK_PANEL);
    expect(tokens.Z_TASK_PANEL).toBeLessThan(tokens.Z_DROPDOWN);
    expect(tokens.Z_DROPDOWN).toBeLessThan(tokens.Z_TOOLTIP);
    expect(tokens.Z_TOOLTIP).toBeLessThan(tokens.Z_TOAST);
  });

  it('exports motion durations from fast to slower', () => {
    expect(tokens.DURATION_FAST).toBeLessThan(tokens.DURATION_NORMAL);
    expect(tokens.DURATION_NORMAL).toBeLessThan(tokens.DURATION_SLOW);
    expect(tokens.DURATION_SLOW).toBeLessThan(tokens.DURATION_SLOWER);
  });

  it('exports a chart palette of at least 4 colors', () => {
    expect(Array.isArray(tokens.CHART_COLORS)).toBe(true);
    expect(tokens.CHART_COLORS.length).toBeGreaterThanOrEqual(4);
  });

  it('exports radius scale from xs to full', () => {
    expect(tokens.RADIUS_XS).toBeLessThan(tokens.RADIUS_SM);
    expect(tokens.RADIUS_SM).toBeLessThan(tokens.RADIUS_MD);
    expect(tokens.RADIUS_MD).toBeLessThan(tokens.RADIUS_LG);
    expect(tokens.RADIUS_LG).toBeLessThan(tokens.RADIUS_XL);
    expect(tokens.RADIUS_XL).toBeLessThan(tokens.RADIUS_FULL);
  });
});
