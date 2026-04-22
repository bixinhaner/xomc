import React, { useRef, useEffect } from 'react';
import { useAppStore } from '@core/store/appStore';
import { getThemeEffects } from './themeEffectConfig';

/**
 * Mouse-tracking radial light overlay.
 *
 * Performance-optimised:
 *  - NO independent RAF loop — only updates on actual mousemove events
 *  - Throttled to ~20 fps via timestamp check (not 60 fps)
 *  - CSS `transition` handles interpolation between updates
 */
const DynamicLightSource: React.FC = () => {
  const ref = useRef<HTMLDivElement>(null);
  const lastUpdate = useRef(0);
  const theme = useAppStore((s) => s.theme);
  const config = getThemeEffects(theme);

  useEffect(() => {
    if (!config.lightSource.enabled) return;

    const mq = window.matchMedia('(prefers-reduced-motion: reduce)');
    if (mq.matches) return;

    const el = ref.current;
    if (!el) return;

    const THROTTLE_MS = 50; // ~20 fps — more than enough for a soft glow

    const updateLight = (clientX: number, clientY: number) => {
      const now = performance.now();
      if (now - lastUpdate.current < THROTTLE_MS) return;
      lastUpdate.current = now;

      const x = ((clientX / window.innerWidth) * 100).toFixed(1);
      const y = ((clientY / window.innerHeight) * 100).toFixed(1);

      el.style.background =
        `radial-gradient(${config.lightSource.radius}px circle at ${x}% ${y}%, ${config.lightSource.color}, transparent)`;
    };

    const handleMove = (e: MouseEvent) => {
      updateLight(e.clientX, e.clientY);
    };

    const handleTouch = (e: TouchEvent) => {
      const touch = e.touches[0];
      if (!touch) return;
      updateLight(touch.clientX, touch.clientY);
    };

    window.addEventListener('mousemove', handleMove, { passive: true });
    window.addEventListener('touchmove', handleTouch, { passive: true });

    return () => {
      window.removeEventListener('mousemove', handleMove);
      window.removeEventListener('touchmove', handleTouch);
    };
  }, [theme, config.lightSource.enabled, config.lightSource.color, config.lightSource.radius]);

  if (!config.lightSource.enabled) return null;

  return (
    <div
      ref={ref}
      style={{
        position: 'absolute',
        inset: 0,
        pointerEvents: 'none',
        zIndex: 0,
        mixBlendMode: 'overlay',
        transition: 'background 0.15s ease-out',
      }}
    />
  );
};

export default DynamicLightSource;
