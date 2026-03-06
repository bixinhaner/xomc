import type { Theme } from '@/types/common';

export interface TiltConfig {
  maxTilt: number;
  perspective: number;
  scale: number;
  glare: boolean;
  glareColor: string;
}

export interface FloatConfig {
  amplitude: number;
  speed: number;
}

export interface ParticleConfig {
  preset: string;
  density: number;
  interactive: boolean;
  colors: [string, string, string];
}

export interface LightSourceConfig {
  enabled: boolean;
  color: string;
  radius: number;
}

export interface ThemeEffectConfig {
  tilt: TiltConfig;
  float: FloatConfig;
  particles: ParticleConfig;
  lightSource: LightSourceConfig;
  glowColor: string;
}

/**
 * Particle density values are intentionally LOW.
 * The formula `(w*h / 10000) * density` means density 10 on a
 * 1920×1080 screen ≈ 207 * 10 ≈ 20 particles. That's plenty.
 */
export const THEME_EFFECTS: Record<Theme, ThemeEffectConfig> = {
  classic: {
    tilt: { maxTilt: 5, perspective: 1200, scale: 1.01, glare: false, glareColor: 'rgba(255,255,255,0.08)' },
    float: { amplitude: 3, speed: 4000 },
    particles: { preset: 'subtle-dots', density: 4, interactive: false, colors: ['#1677FF', '#4096FF', '#69B1FF'] },
    lightSource: { enabled: true, color: 'rgba(22,119,255,0.04)', radius: 450 },
    glowColor: '#1677FF',
  },
  tech: {
    tilt: { maxTilt: 10, perspective: 1000, scale: 1.02, glare: true, glareColor: 'rgba(0,212,255,0.1)' },
    float: { amplitude: 5, speed: 3500 },
    particles: { preset: 'node-network', density: 8, interactive: true, colors: ['#00D4FF', '#00FF88', '#6C5CE7'] },
    lightSource: { enabled: true, color: 'rgba(0,212,255,0.05)', radius: 350 },
    glowColor: '#00D4FF',
  },
  fresh: {
    tilt: { maxTilt: 8, perspective: 1100, scale: 1.02, glare: false, glareColor: 'rgba(99,102,241,0.08)' },
    float: { amplitude: 4, speed: 3800 },
    particles: { preset: 'aurora-orbs', density: 6, interactive: true, colors: ['#6366F1', '#22C55E', '#F59E0B'] },
    lightSource: { enabled: true, color: 'rgba(99,102,241,0.04)', radius: 400 },
    glowColor: '#6366F1',
  },
  cyberpunk: {
    tilt: { maxTilt: 15, perspective: 800, scale: 1.03, glare: true, glareColor: 'rgba(0,245,255,0.15)' },
    float: { amplitude: 8, speed: 2500 },
    particles: { preset: 'matrix-rain', density: 10, interactive: true, colors: ['#00F5FF', '#BF00FF', '#00FF88'] },
    lightSource: { enabled: true, color: 'rgba(0,245,255,0.06)', radius: 300 },
    glowColor: '#00F5FF',
  },
  minions: {
    tilt: { maxTilt: 12, perspective: 900, scale: 1.04, glare: false, glareColor: 'rgba(255,217,61,0.12)' },
    float: { amplitude: 7, speed: 2800 },
    particles: { preset: 'bubbles', density: 8, interactive: true, colors: ['#FFD93D', '#4169E1', '#FF6B35'] },
    lightSource: { enabled: true, color: 'rgba(255,217,61,0.05)', radius: 380 },
    glowColor: '#FFD93D',
  },
  tiffany: {
    tilt: { maxTilt: 6, perspective: 1100, scale: 1.015, glare: true, glareColor: 'rgba(129,216,208,0.1)' },
    float: { amplitude: 3, speed: 4200 },
    particles: { preset: 'sparkle', density: 6, interactive: false, colors: ['#81D8D0', '#B76E79', '#E8E8E8'] },
    lightSource: { enabled: true, color: 'rgba(129,216,208,0.04)', radius: 420 },
    glowColor: '#81D8D0',
  },
  rmb: {
    tilt: { maxTilt: 7, perspective: 1100, scale: 1.02, glare: true, glareColor: 'rgba(212,175,55,0.1)' },
    float: { amplitude: 3, speed: 4000 },
    particles: { preset: 'golden-fall', density: 6, interactive: false, colors: ['#D4AF37', '#E60012', '#FFD700'] },
    lightSource: { enabled: true, color: 'rgba(212,175,55,0.04)', radius: 400 },
    glowColor: '#D4AF37',
  },
};

export function getThemeEffects(theme: Theme): ThemeEffectConfig {
  return THEME_EFFECTS[theme];
}
