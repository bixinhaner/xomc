import { useEffect, useRef } from 'react';
import { useAppStore } from '@core/store/appStore';

/**
 * Adds the 'omc-visible' class to elements with 'omc-scroll-reveal' class
 * when they enter the viewport.  Uses IntersectionObserver for performance.
 */
export function useScrollReveal(containerRef: React.RefObject<HTMLElement | null>) {
  const observerRef = useRef<IntersectionObserver | null>(null);
  const effects3D = useAppStore((s) => s.effects3DEnabled);
  const theme = useAppStore((s) => s.theme);

  useEffect(() => {
    const container = containerRef.current;
    if (!container) return;

    // When 3D disabled or reduced motion — show everything immediately
    const mq = window.matchMedia('(prefers-reduced-motion: reduce)');
    if (!effects3D || mq.matches) {
      container.querySelectorAll('.omc-scroll-reveal').forEach((el) => {
        el.classList.add('omc-visible');
      });
      return;
    }

    observerRef.current = new IntersectionObserver(
      (entries) => {
        entries.forEach((entry) => {
          if (entry.isIntersecting) {
            entry.target.classList.add('omc-visible');
            observerRef.current?.unobserve(entry.target);
          }
        });
      },
      { threshold: 0.1, rootMargin: '0px 0px -40px 0px' },
    );

    const elements = container.querySelectorAll('.omc-scroll-reveal');
    elements.forEach((el) => observerRef.current?.observe(el));

    return () => {
      observerRef.current?.disconnect();
    };
  }, [containerRef, effects3D, theme]);
}
