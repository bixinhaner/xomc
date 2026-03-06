import type { Theme } from '@/types/common';

export interface Particle {
  x: number;
  y: number;
  vx: number;
  vy: number;
  size: number;
  opacity: number;
  color: string;
  /** For text-based particles (matrix rain) */
  char?: string;
  /** Rotation angle in radians */
  rotation?: number;
  /** Rotation speed */
  rotationSpeed?: number;
  /** Life remaining (0-1) for fading particles */
  life?: number;
}

export interface ParticlePreset {
  /** Create initial particles */
  init(w: number, h: number, density: number, colors: [string, string, string]): Particle[];
  /** Update particle per frame */
  update(p: Particle, dt: number, w: number, h: number, mouseX: number, mouseY: number, interactive: boolean): void;
  /** Draw single particle */
  draw(ctx: CanvasRenderingContext2D, p: Particle): void;
  /** Optional: draw connections between particles */
  drawConnections?(ctx: CanvasRenderingContext2D, particles: Particle[]): void;
}

// ======================== CYBERPUNK: Matrix Rain ========================
const MATRIX_CHARS = 'アイウエオカキクケコサシスセソタチツテトナニヌネノハヒフヘホマミムメモヤユヨラリルレロワヲン0123456789ABCDEF';

const matrixRain: ParticlePreset = {
  init(w, h, density, colors) {
    const count = Math.floor((w * h) / 10000 * density);
    return Array.from({ length: count }, () => ({
      x: Math.random() * w,
      y: Math.random() * h,
      vx: 0,
      vy: 30 + Math.random() * 80,
      size: 12 + Math.random() * 6,
      opacity: 0.15 + Math.random() * 0.5,
      color: colors[Math.floor(Math.random() * colors.length)],
      char: MATRIX_CHARS[Math.floor(Math.random() * MATRIX_CHARS.length)],
      life: Math.random(),
    }));
  },
  update(p, dt, _w, h) {
    p.y += p.vy * dt;
    if (Math.random() < 0.02) {
      p.char = MATRIX_CHARS[Math.floor(Math.random() * MATRIX_CHARS.length)];
    }
    if (p.y > h + p.size) {
      p.y = -p.size;
      p.opacity = 0.15 + Math.random() * 0.5;
    }
  },
  draw(ctx, p) {
    // Batch-friendly: font is set once in the canvas loop via ParticleCanvas
    ctx.globalAlpha = p.opacity;
    ctx.fillStyle = p.color;
    ctx.fillText(p.char ?? '0', p.x, p.y);
    ctx.globalAlpha = 1;
  },
};

// ======================== TECH: Node Network ========================
const MAX_CONNECTIONS_PER_NODE = 3;
const CONNECTION_DIST = 120;
const CONNECTION_DIST_SQ = CONNECTION_DIST * CONNECTION_DIST;

const nodeNetwork: ParticlePreset = {
  init(w, h, density, colors) {
    const count = Math.floor((w * h) / 10000 * density);
    return Array.from({ length: count }, () => ({
      x: Math.random() * w,
      y: Math.random() * h,
      vx: (Math.random() - 0.5) * 20,
      vy: (Math.random() - 0.5) * 20,
      size: 1.5 + Math.random() * 2,
      opacity: 0.3 + Math.random() * 0.5,
      color: colors[Math.floor(Math.random() * colors.length)],
    }));
  },
  update(p, dt, w, h, mouseX, mouseY, interactive) {
    p.x += p.vx * dt;
    p.y += p.vy * dt;

    if (interactive && mouseX > 0 && mouseY > 0) {
      const dx = p.x - mouseX;
      const dy = p.y - mouseY;
      const distSq = dx * dx + dy * dy;
      if (distSq < 14400) { // 120²
        const dist = Math.sqrt(distSq);
        const force = (120 - dist) / 120 * 1.5;
        p.vx += (dx / dist) * force;
        p.vy += (dy / dist) * force;
      }
    }

    p.vx *= 0.99;
    p.vy *= 0.99;

    if (p.x < 0) p.x = w;
    if (p.x > w) p.x = 0;
    if (p.y < 0) p.y = h;
    if (p.y > h) p.y = 0;
  },
  draw(ctx, p) {
    ctx.beginPath();
    ctx.arc(p.x, p.y, p.size, 0, Math.PI * 2);
    ctx.fillStyle = p.color;
    ctx.globalAlpha = p.opacity;
    ctx.fill();
    ctx.globalAlpha = 1;
  },
  /**
   * O(n) connection drawing — each particle only connects to its nearest
   * few neighbours (capped at MAX_CONNECTIONS_PER_NODE).
   * We limit the inner loop range to ±10 indices as a cheap spatial heuristic.
   */
  drawConnections(ctx, particles) {
    const len = particles.length;
    ctx.lineWidth = 0.5;

    for (let i = 0; i < len; i++) {
      const pi = particles[i];
      let drawn = 0;
      // Only check a small window of nearby array indices
      const jEnd = Math.min(i + 12, len);
      for (let j = i + 1; j < jEnd && drawn < MAX_CONNECTIONS_PER_NODE; j++) {
        const pj = particles[j];
        const dx = pi.x - pj.x;
        const dy = pi.y - pj.y;
        const distSq = dx * dx + dy * dy;
        if (distSq < CONNECTION_DIST_SQ) {
          const opacity = (1 - Math.sqrt(distSq) / CONNECTION_DIST) * 0.25;
          ctx.beginPath();
          ctx.moveTo(pi.x, pi.y);
          ctx.lineTo(pj.x, pj.y);
          ctx.strokeStyle = pi.color;
          ctx.globalAlpha = opacity;
          ctx.stroke();
          drawn++;
        }
      }
    }
    ctx.globalAlpha = 1;
  },
};

// ======================== FRESH: Aurora Orbs ========================
const auroraOrbs: ParticlePreset = {
  init(w, h, density, colors) {
    const count = Math.floor((w * h) / 10000 * density);
    return Array.from({ length: count }, () => ({
      x: Math.random() * w,
      y: Math.random() * h,
      vx: (Math.random() - 0.5) * 8,
      vy: (Math.random() - 0.5) * 8,
      size: 15 + Math.random() * 40,
      opacity: 0.03 + Math.random() * 0.06,
      color: colors[Math.floor(Math.random() * colors.length)],
      rotation: Math.random() * Math.PI * 2,
      rotationSpeed: (Math.random() - 0.5) * 0.3,
    }));
  },
  update(p, dt, w, h) {
    p.x += p.vx * dt;
    p.y += p.vy * dt;
    p.rotation = (p.rotation ?? 0) + (p.rotationSpeed ?? 0) * dt;

    if (p.x < -p.size) p.x = w + p.size;
    if (p.x > w + p.size) p.x = -p.size;
    if (p.y < -p.size) p.y = h + p.size;
    if (p.y > h + p.size) p.y = -p.size;
  },
  draw(ctx, p) {
    // Use simple filled circle with alpha instead of createRadialGradient per frame
    ctx.globalAlpha = p.opacity;
    ctx.fillStyle = p.color;
    ctx.beginPath();
    ctx.arc(p.x, p.y, p.size, 0, Math.PI * 2);
    ctx.fill();
    // Softer inner core
    ctx.globalAlpha = p.opacity * 1.5;
    ctx.beginPath();
    ctx.arc(p.x, p.y, p.size * 0.4, 0, Math.PI * 2);
    ctx.fill();
    ctx.globalAlpha = 1;
  },
};

// ======================== MINIONS: Bubbles ========================
const bubbles: ParticlePreset = {
  init(w, h, density, colors) {
    const count = Math.floor((w * h) / 10000 * density);
    return Array.from({ length: count }, () => ({
      x: Math.random() * w,
      y: h + Math.random() * h,
      vx: (Math.random() - 0.5) * 10,
      vy: -(10 + Math.random() * 25),
      size: 4 + Math.random() * 12,
      opacity: 0.1 + Math.random() * 0.25,
      color: colors[Math.floor(Math.random() * colors.length)],
      rotation: Math.random() * Math.PI * 2,
      rotationSpeed: (Math.random() - 0.5) * 2,
    }));
  },
  update(p, dt, w, h, mouseX, mouseY, interactive) {
    p.x += p.vx * dt + Math.sin((p.rotation ?? 0)) * 0.3;
    p.y += p.vy * dt;
    p.rotation = (p.rotation ?? 0) + (p.rotationSpeed ?? 0) * dt;

    if (interactive && mouseX > 0) {
      const dx = mouseX - p.x;
      const dy = mouseY - p.y;
      const distSq = dx * dx + dy * dy;
      if (distSq < 22500) { // 150²
        const dist = Math.sqrt(distSq);
        p.vx += (dx / dist) * 0.3;
        p.vy += (dy / dist) * 0.3;
      }
    }

    if (p.y < -p.size * 2) {
      p.y = h + p.size;
      p.x = Math.random() * w;
      p.opacity = 0.1 + Math.random() * 0.25;
    }
  },
  draw(ctx, p) {
    ctx.beginPath();
    ctx.arc(p.x, p.y, p.size, 0, Math.PI * 2);
    ctx.fillStyle = p.color;
    ctx.globalAlpha = p.opacity;
    ctx.fill();
    // Highlight — no save/restore needed
    ctx.beginPath();
    ctx.arc(p.x - p.size * 0.3, p.y - p.size * 0.3, p.size * 0.25, 0, Math.PI * 2);
    ctx.fillStyle = 'rgba(255,255,255,0.6)';
    ctx.fill();
    ctx.globalAlpha = 1;
  },
};

// ======================== TIFFANY: Sparkle ========================
const sparkle: ParticlePreset = {
  init(w, h, density, colors) {
    const count = Math.floor((w * h) / 10000 * density);
    return Array.from({ length: count }, () => ({
      x: Math.random() * w,
      y: Math.random() * h,
      vx: (Math.random() - 0.5) * 4,
      vy: (Math.random() - 0.5) * 4,
      size: 1 + Math.random() * 3,
      opacity: 0,
      color: colors[Math.floor(Math.random() * colors.length)],
      life: Math.random(),
      rotation: Math.random() * Math.PI * 2,
      rotationSpeed: (Math.random() - 0.5) * 4,
    }));
  },
  update(p, dt, w, h) {
    p.x += p.vx * dt;
    p.y += p.vy * dt;
    p.life = ((p.life ?? 0) + dt * 0.3) % 1;
    p.opacity = Math.sin(p.life * Math.PI) * 0.5;

    if (p.x < 0) p.x = w;
    if (p.x > w) p.x = 0;
    if (p.y < 0) p.y = h;
    if (p.y > h) p.y = 0;
  },
  draw(ctx, p) {
    // Simple cross + diamond — no save/restore, use direct offset drawing
    ctx.globalAlpha = p.opacity;
    ctx.fillStyle = p.color;
    const s = p.size;
    // Diamond
    ctx.beginPath();
    ctx.moveTo(p.x, p.y - s * 2);
    ctx.lineTo(p.x + s, p.y);
    ctx.lineTo(p.x, p.y + s * 2);
    ctx.lineTo(p.x - s, p.y);
    ctx.closePath();
    ctx.fill();
    // Cross lines
    ctx.strokeStyle = p.color;
    ctx.lineWidth = 0.5;
    ctx.beginPath();
    ctx.moveTo(p.x - s * 1.5, p.y);
    ctx.lineTo(p.x + s * 1.5, p.y);
    ctx.moveTo(p.x, p.y - s * 1.5);
    ctx.lineTo(p.x, p.y + s * 1.5);
    ctx.stroke();
    ctx.globalAlpha = 1;
  },
};

// ======================== RMB: Golden Fall ========================
const goldenFall: ParticlePreset = {
  init(w, h, density, colors) {
    const count = Math.floor((w * h) / 10000 * density);
    return Array.from({ length: count }, () => ({
      x: Math.random() * w,
      y: Math.random() * h,
      vx: (Math.random() - 0.5) * 6,
      vy: 8 + Math.random() * 15,
      size: 2 + Math.random() * 4,
      opacity: 0.1 + Math.random() * 0.3,
      color: colors[Math.floor(Math.random() * colors.length)],
      rotation: Math.random() * Math.PI * 2,
      rotationSpeed: (Math.random() - 0.5) * 3,
    }));
  },
  update(p, dt, w, h) {
    p.x += p.vx * dt + Math.sin((p.rotation ?? 0) * 0.5) * 0.5;
    p.y += p.vy * dt;
    p.rotation = (p.rotation ?? 0) + (p.rotationSpeed ?? 0) * dt;

    if (p.y > h + p.size) {
      p.y = -p.size * 2;
      p.x = Math.random() * w;
    }
    if (p.x < 0) p.x = w;
    if (p.x > w) p.x = 0;
  },
  draw(ctx, p) {
    // Simple ellipse — no save/restore, draw at position directly
    ctx.globalAlpha = p.opacity;
    ctx.beginPath();
    ctx.ellipse(p.x, p.y, p.size, p.size * 0.6, p.rotation ?? 0, 0, Math.PI * 2);
    ctx.fillStyle = p.color;
    ctx.fill();
    ctx.globalAlpha = 1;
  },
};

// ======================== CLASSIC: Subtle Dots ========================
const subtleDots: ParticlePreset = {
  init(w, h, density, colors) {
    const count = Math.floor((w * h) / 10000 * density);
    return Array.from({ length: count }, () => ({
      x: Math.random() * w,
      y: Math.random() * h,
      vx: (Math.random() - 0.5) * 3,
      vy: (Math.random() - 0.5) * 3,
      size: 1 + Math.random() * 1.5,
      opacity: 0.05 + Math.random() * 0.1,
      color: colors[0],
    }));
  },
  update(p, dt, w, h) {
    p.x += p.vx * dt;
    p.y += p.vy * dt;
    if (p.x < 0) p.x = w;
    if (p.x > w) p.x = 0;
    if (p.y < 0) p.y = h;
    if (p.y > h) p.y = 0;
  },
  draw(ctx, p) {
    ctx.beginPath();
    ctx.arc(p.x, p.y, p.size, 0, Math.PI * 2);
    ctx.fillStyle = p.color;
    ctx.globalAlpha = p.opacity;
    ctx.fill();
    ctx.globalAlpha = 1;
  },
};

// ======================== Registry ========================
export const PARTICLE_PRESETS: Record<string, ParticlePreset> = {
  'matrix-rain': matrixRain,
  'node-network': nodeNetwork,
  'aurora-orbs': auroraOrbs,
  'bubbles': bubbles,
  'sparkle': sparkle,
  'golden-fall': goldenFall,
  'subtle-dots': subtleDots,
};

export function getParticlePreset(theme: Theme): ParticlePreset {
  const presetMap: Record<Theme, string> = {
    cyberpunk: 'matrix-rain',
    tech: 'node-network',
    fresh: 'aurora-orbs',
    minions: 'bubbles',
    tiffany: 'sparkle',
    rmb: 'golden-fall',
    classic: 'subtle-dots',
  };
  return PARTICLE_PRESETS[presetMap[theme]];
}
