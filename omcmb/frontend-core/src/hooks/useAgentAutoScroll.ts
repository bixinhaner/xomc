import { useCallback, useEffect, useRef, useState } from 'react';

const DEFAULT_BOTTOM_THRESHOLD = 72;

export interface AgentScrollMetrics {
  scrollTop: number;
  scrollHeight: number;
  clientHeight: number;
}

export interface AgentScrollAnchor {
  element: HTMLElement;
  viewportOffset: number;
}

export interface UseAgentAutoScrollOptions {
  active: boolean;
}

export interface UseAgentAutoScrollResult {
  scrollContainerRef: React.RefObject<HTMLDivElement | null>;
  scrollContentRef: React.RefObject<HTMLDivElement | null>;
  showJumpToLatest: boolean;
  handleScroll: () => void;
  scrollToLatest: (behavior?: ScrollBehavior) => void;
}

export function isAgentViewportAtBottom(
  metrics: AgentScrollMetrics,
  threshold = DEFAULT_BOTTOM_THRESHOLD
): boolean {
  return metrics.scrollHeight - metrics.clientHeight - metrics.scrollTop <= threshold;
}

export function captureAgentScrollAnchor(
  container: HTMLElement,
  content: HTMLElement
): AgentScrollAnchor | null {
  const containerTop = container.getBoundingClientRect().top;
  const firstVisible = Array.from(content.children).find((child) => {
    const rect = child.getBoundingClientRect();
    return rect.bottom > containerTop;
  });
  if (!(firstVisible instanceof HTMLElement)) return null;

  return {
    element: firstVisible,
    viewportOffset: firstVisible.getBoundingClientRect().top - containerTop,
  };
}

export function getAgentScrollAnchorAdjustment(
  container: HTMLElement,
  anchor: AgentScrollAnchor
): number {
  return (
    anchor.element.getBoundingClientRect().top -
    container.getBoundingClientRect().top -
    anchor.viewportOffset
  );
}

export function useAgentAutoScroll({ active }: UseAgentAutoScrollOptions): UseAgentAutoScrollResult {
  const scrollContainerRef = useRef<HTMLDivElement>(null);
  const scrollContentRef = useRef<HTMLDivElement>(null);
  const pinnedRef = useRef(true);
  const animationFrameRef = useRef<number | null>(null);
  const smoothScrollTimerRef = useRef<number | null>(null);
  const programmaticScrollRef = useRef(false);
  const scrollAnchorRef = useRef<AgentScrollAnchor | null>(null);
  const [showJumpToLatest, setShowJumpToLatest] = useState(false);

  const setPinned = useCallback((pinned: boolean) => {
    pinnedRef.current = pinned;
    if (pinned) scrollAnchorRef.current = null;
    setShowJumpToLatest((current) => {
      const next = !pinned;
      return current === next ? current : next;
    });
  }, []);

  const scrollToLatest = useCallback(
    (behavior: ScrollBehavior = 'smooth') => {
      setPinned(true);
      programmaticScrollRef.current = behavior === 'smooth';

      if (animationFrameRef.current !== null) {
        window.cancelAnimationFrame(animationFrameRef.current);
      }
      if (smoothScrollTimerRef.current !== null) {
        window.clearTimeout(smoothScrollTimerRef.current);
        smoothScrollTimerRef.current = null;
      }

      animationFrameRef.current = window.requestAnimationFrame(() => {
        animationFrameRef.current = null;
        const container = scrollContainerRef.current;
        if (!container) {
          programmaticScrollRef.current = false;
          return;
        }
        container.scrollTo({ top: container.scrollHeight, behavior });

        if (behavior === 'smooth') {
          smoothScrollTimerRef.current = window.setTimeout(() => {
            smoothScrollTimerRef.current = null;
            programmaticScrollRef.current = false;
            const current = scrollContainerRef.current;
            if (current) setPinned(isAgentViewportAtBottom(current));
          }, 400);
        } else {
          programmaticScrollRef.current = false;
        }
      });
    },
    [setPinned]
  );

  const handleScroll = useCallback(() => {
    if (programmaticScrollRef.current) return;
    const container = scrollContainerRef.current;
    const content = scrollContentRef.current;
    if (!container) return;
    const pinned = isAgentViewportAtBottom(container);
    setPinned(pinned);
    if (!pinned && content) {
      scrollAnchorRef.current = captureAgentScrollAnchor(container, content);
    }
  }, [setPinned]);

  useEffect(() => {
    if (!active) return;
    const frame = window.requestAnimationFrame(() => scrollToLatest('auto'));
    return () => window.cancelAnimationFrame(frame);
  }, [active, scrollToLatest]);

  useEffect(() => {
    if (!active || typeof ResizeObserver === 'undefined') return undefined;
    const container = scrollContainerRef.current;
    const content = scrollContentRef.current;
    if (!container || !content) return undefined;

    const observer = new ResizeObserver(() => {
      if (pinnedRef.current) {
        scrollToLatest('auto');
        return;
      }

      const anchor = scrollAnchorRef.current;
      if (anchor && content.contains(anchor.element)) {
        const adjustment = getAgentScrollAnchorAdjustment(container, anchor);
        if (Math.abs(adjustment) >= 0.5) {
          container.scrollTop += adjustment;
        }
      } else {
        scrollAnchorRef.current = captureAgentScrollAnchor(container, content);
      }
    });
    observer.observe(container);
    observer.observe(content);
    return () => observer.disconnect();
  }, [active, scrollToLatest, setPinned]);

  useEffect(() => {
    if (!active) return undefined;
    const container = scrollContainerRef.current;
    if (!container) return undefined;

    const interruptProgrammaticScroll = () => {
      if (!programmaticScrollRef.current) return;
      if (smoothScrollTimerRef.current !== null) {
        window.clearTimeout(smoothScrollTimerRef.current);
        smoothScrollTimerRef.current = null;
      }
      programmaticScrollRef.current = false;
      container.scrollTo({ top: container.scrollTop, behavior: 'auto' });
      setPinned(false);
      const content = scrollContentRef.current;
      if (content) {
        scrollAnchorRef.current = captureAgentScrollAnchor(container, content);
      }
    };

    container.addEventListener('wheel', interruptProgrammaticScroll, { passive: true });
    container.addEventListener('touchstart', interruptProgrammaticScroll, { passive: true });
    return () => {
      container.removeEventListener('wheel', interruptProgrammaticScroll);
      container.removeEventListener('touchstart', interruptProgrammaticScroll);
    };
  }, [active, setPinned]);

  useEffect(
    () => () => {
      if (animationFrameRef.current !== null) {
        window.cancelAnimationFrame(animationFrameRef.current);
      }
      if (smoothScrollTimerRef.current !== null) {
        window.clearTimeout(smoothScrollTimerRef.current);
      }
    },
    []
  );

  return {
    scrollContainerRef,
    scrollContentRef,
    showJumpToLatest,
    handleScroll,
    scrollToLatest,
  };
}
