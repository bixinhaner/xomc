import { useEffect, useRef, type RefObject } from 'react';

export interface TiltOptions {
  /** Max tilt angle in degrees (default 12) */
  maxTilt?: number;
  /** CSS perspective value in px (default 1000) */
  perspective?: number;
  /** Scale factor on hover (default 1.02) */
  scale?: number;
  /** Transition speed in ms for return animation (default 400) */
  speed?: number;
  /** Enable dynamic shadow — now pure CSS, no JS per-frame */
  shadow?: boolean;
  /** Disable the entire effect */
  disabled?: boolean;
}

/**
 * Applies a 3D perspective tilt effect to an element based on mouse position.
 *
 * Performance-optimised:
 *  - `getBoundingClientRect()` cached on mouseenter (not every frame)
 *  - Only `transform` is changed per frame (compositor-only, zero paint)
 *  - No JS box-shadow manipulation — uses CSS class for static shadow
 *  - Return animation handled by CSS transition (no JS animation loop)
 */
export function use3DTilt(
  ref: RefObject<HTMLElement | null>,
  options: TiltOptions = {},
) {
  const {
    maxTilt = 12,
    perspective = 1000,
    scale = 1.02,
    speed = 400,
    disabled = false,
  } = options;

  const rafRef = useRef(0);
  const rectRef = useRef<DOMRect | null>(null);

  useEffect(() => {
    if (disabled) return;
    const el = ref.current;
    if (!el) return;

    // Respect reduced motion preference
    const mq = window.matchMedia('(prefers-reduced-motion: reduce)');
    if (mq.matches) return;

    // Disable on touch-primary devices (finger covers the card, tilt is meaningless)
    const touchMq = window.matchMedia('(pointer: coarse)');
    if (touchMq.matches) return;

    // Promote to own compositor layer — set once
    el.style.transformStyle = 'preserve-3d';
    el.style.willChange = 'transform';

    const handleEnter = () => {
      // Cache rect once per hover — eliminates per-frame layout thrashing
      rectRef.current = el.getBoundingClientRect();
      // Disable transition for instant response during mouse tracking
      el.style.transition = 'none';
    };

    const handleMove = (e: MouseEvent) => {
      const rect = rectRef.current;
      if (!rect) return;
      if (rafRef.current) return; // already scheduled

      rafRef.current = requestAnimationFrame(() => {
        rafRef.current = 0;
        if (!rectRef.current) return;

        const centerX = rect.left + rect.width / 2;
        const centerY = rect.top + rect.height / 2;

        // Normalise to -1…1
        const px = Math.max(-1, Math.min(1, (e.clientX - centerX) / (rect.width / 2)));
        const py = Math.max(-1, Math.min(1, (e.clientY - centerY) / (rect.height / 2)));

        const rotateX = py * -maxTilt;
        const rotateY = px * maxTilt;

        // Pure transform — compositor-only, zero paint/layout cost
        el.style.transform =
          `perspective(${perspective}px) rotateX(${rotateX.toFixed(2)}deg) rotateY(${rotateY.toFixed(2)}deg) scale3d(${scale}, ${scale}, ${scale})`;

        // Expose tilt values for CSS-driven glare / holo overlays
        el.style.setProperty('--tilt-x', `${px.toFixed(3)}`);
        el.style.setProperty('--tilt-y', `${py.toFixed(3)}`);
      });
    };

    const handleLeave = () => {
      rectRef.current = null;
      if (rafRef.current) {
        cancelAnimationFrame(rafRef.current);
        rafRef.current = 0;
      }
      // Enable CSS transition for smooth spring-back
      el.style.transition = `transform ${speed}ms cubic-bezier(0.22, 1, 0.36, 1)`;
      el.style.transform = `perspective(${perspective}px) rotateX(0deg) rotateY(0deg) scale3d(1, 1, 1)`;
      el.style.removeProperty('--tilt-x');
      el.style.removeProperty('--tilt-y');
    };

    el.addEventListener('mouseenter', handleEnter, { passive: true });
    el.addEventListener('mousemove', handleMove, { passive: true });
    el.addEventListener('mouseleave', handleLeave, { passive: true });

    return () => {
      el.removeEventListener('mouseenter', handleEnter);
      el.removeEventListener('mousemove', handleMove);
      el.removeEventListener('mouseleave', handleLeave);
      if (rafRef.current) cancelAnimationFrame(rafRef.current);
      el.style.transform = '';
      el.style.willChange = '';
      el.style.transformStyle = '';
      el.style.transition = '';
    };
  }, [maxTilt, perspective, scale, speed, ref, disabled]);
}
