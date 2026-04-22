import React, { useEffect, useRef, useCallback } from 'react';
import { useAppStore } from '@core/store/appStore';
import { getThemeEffects } from './themeEffectConfig';
import { getParticlePreset, type Particle } from './particlePresets';

/** Target 24 fps — particles don't need 60 fps smoothness */
const TARGET_FPS = 24;
const FRAME_INTERVAL = 1000 / TARGET_FPS;
/** Hard cap: never exceed this many particles regardless of density calc */
const MAX_PARTICLES = 80;

const ParticleCanvas: React.FC = () => {
  const canvasRef = useRef<HTMLCanvasElement>(null);
  const particlesRef = useRef<Particle[]>([]);
  const rafRef = useRef(0);
  const lastFrameRef = useRef(0);
  const mouseRef = useRef({ x: 0, y: 0 });
  const theme = useAppStore((s) => s.theme);

  const handleMouseMove = useCallback((e: MouseEvent) => {
    const canvas = canvasRef.current;
    if (!canvas) return;
    const rect = canvas.getBoundingClientRect();
    mouseRef.current.x = e.clientX - rect.left;
    mouseRef.current.y = e.clientY - rect.top;
  }, []);

  const handleTouchMove = useCallback((e: TouchEvent) => {
    const canvas = canvasRef.current;
    if (!canvas) return;
    const touch = e.touches[0];
    if (!touch) return;
    const rect = canvas.getBoundingClientRect();
    mouseRef.current.x = touch.clientX - rect.left;
    mouseRef.current.y = touch.clientY - rect.top;
  }, []);

  useEffect(() => {
    const canvas = canvasRef.current;
    if (!canvas) return;
    const ctx = canvas.getContext('2d');
    if (!ctx) return;

    const mq = window.matchMedia('(prefers-reduced-motion: reduce)');
    if (mq.matches) return;

    const config = getThemeEffects(theme);
    const preset = getParticlePreset(theme);
    const { density, interactive, colors } = config.particles;

    // Determine if this preset needs a fixed font
    const isMatrixRain = config.particles.preset === 'matrix-rain';

    const resize = () => {
      const parent = canvas.parentElement;
      if (!parent) return;
      const w = parent.clientWidth;
      const h = parent.clientHeight;
      const dpr = Math.min(window.devicePixelRatio, 2);
      canvas.width = w * dpr;
      canvas.height = h * dpr;
      canvas.style.width = `${w}px`;
      canvas.style.height = `${h}px`;
      ctx.setTransform(dpr, 0, 0, dpr, 0, 0);

      // Init particles with hard cap
      let particles = preset.init(w, h, density, colors);
      if (particles.length > MAX_PARTICLES) {
        particles = particles.slice(0, MAX_PARTICLES);
      }
      particlesRef.current = particles;
    };

    resize();
    const ro = new ResizeObserver(resize);
    ro.observe(canvas.parentElement!);

    window.addEventListener('mousemove', handleMouseMove, { passive: true });
    window.addEventListener('touchmove', handleTouchMove, { passive: true });

    let visible = true;
    const handleVisibility = () => { visible = !document.hidden; };
    document.addEventListener('visibilitychange', handleVisibility);

    const animate = (now: number) => {
      rafRef.current = requestAnimationFrame(animate);
      if (!visible) return;

      const elapsed = now - lastFrameRef.current;
      if (elapsed < FRAME_INTERVAL) return;
      lastFrameRef.current = now - (elapsed % FRAME_INTERVAL);

      const dt = Math.min(elapsed / 1000, 0.05);
      const dpr = Math.min(window.devicePixelRatio, 2);
      const w = canvas.width / dpr;
      const h = canvas.height / dpr;

      ctx.clearRect(0, 0, w, h);

      // Set font once per frame for matrix rain (instead of per-particle)
      if (isMatrixRain) {
        ctx.font = '14px monospace';
      }

      const particles = particlesRef.current;
      const { x: mx, y: my } = mouseRef.current;

      for (const p of particles) {
        preset.update(p, dt, w, h, mx, my, interactive);
        preset.draw(ctx, p);
      }

      if (preset.drawConnections) {
        preset.drawConnections(ctx, particles);
      }
    };

    rafRef.current = requestAnimationFrame(animate);

    return () => {
      cancelAnimationFrame(rafRef.current);
      ro.disconnect();
      window.removeEventListener('mousemove', handleMouseMove);
      window.removeEventListener('touchmove', handleTouchMove);
      document.removeEventListener('visibilitychange', handleVisibility);
    };
  }, [theme, handleMouseMove, handleTouchMove]);

  return (
    <canvas
      ref={canvasRef}
      style={{
        position: 'absolute',
        inset: 0,
        width: '100%',
        height: '100%',
        pointerEvents: 'none',
        zIndex: 0,
      }}
    />
  );
};

export default ParticleCanvas;
