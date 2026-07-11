import { useCallback, useEffect, useMemo, useRef, useState, type CSSProperties, type HTMLAttributes, type KeyboardEvent, type PointerEvent } from 'react';

const STORAGE_KEY = 'omc-agent-panel-layout';
const DEFAULT_WIDTH = 400;
const MIN_WIDTH = 360;
const MAX_WIDTH = 760;
const EXPANDED_RATIO = 0.68;
const MOBILE_BREAKPOINT = 768;
const KEYBOARD_STEP = 24;

interface StoredAgentPanelLayout {
  width?: number;
  expanded?: boolean;
}

export interface AgentPanelLayoutOptions {
  storageKey?: string;
  defaultWidth?: number;
  minWidth?: number;
  maxWidth?: number;
  expandedRatio?: number;
  mobileBreakpoint?: number;
  horizontalMargin?: number;
}

export interface AgentPanelWidthBounds {
  min: number;
  max: number;
  expanded: number;
  compact: boolean;
}

export interface UseAgentPanelLayoutResult {
  width: number;
  isExpanded: boolean;
  isResizing: boolean;
  canResize: boolean;
  panelStyle: CSSProperties;
  resizeHandleProps: HTMLAttributes<HTMLDivElement>;
  toggleExpanded: () => void;
  resetWidth: () => void;
}

function windowWidth(): number {
  return typeof window === 'undefined' ? DEFAULT_WIDTH : window.innerWidth;
}

function safeStorage(): Storage | undefined {
  if (typeof window === 'undefined') return undefined;
  try {
    return window.localStorage;
  } catch {
    return undefined;
  }
}

function readLayout(storageKey: string): StoredAgentPanelLayout | undefined {
  const storage = safeStorage();
  if (!storage) return undefined;
  try {
    const raw = storage.getItem(storageKey);
    if (!raw) return undefined;
    const parsed = JSON.parse(raw) as StoredAgentPanelLayout;
    return {
      width: typeof parsed.width === 'number' ? parsed.width : undefined,
      expanded: parsed.expanded === true,
    };
  } catch {
    return undefined;
  }
}

function writeLayout(storageKey: string, layout: StoredAgentPanelLayout) {
  const storage = safeStorage();
  if (!storage) return;
  try {
    storage.setItem(storageKey, JSON.stringify(layout));
  } catch {
    // Layout memory is a convenience only. Ignore private-mode/quota failures.
  }
}

function normalizeOptions(options: AgentPanelLayoutOptions) {
  return {
    storageKey: options.storageKey ?? STORAGE_KEY,
    defaultWidth: options.defaultWidth ?? DEFAULT_WIDTH,
    minWidth: options.minWidth ?? MIN_WIDTH,
    maxWidth: options.maxWidth ?? MAX_WIDTH,
    expandedRatio: options.expandedRatio ?? EXPANDED_RATIO,
    mobileBreakpoint: options.mobileBreakpoint ?? MOBILE_BREAKPOINT,
    horizontalMargin: Math.max(0, options.horizontalMargin ?? 0),
  };
}

export function getAgentPanelWidthBounds(
  viewportWidth: number,
  options: AgentPanelLayoutOptions = {}
): AgentPanelWidthBounds {
  const normalized = normalizeOptions(options);
  const available = Math.max(0, viewportWidth - normalized.horizontalMargin);
  const max = Math.max(0, Math.min(normalized.maxWidth, available || normalized.maxWidth));
  const min = Math.min(normalized.minWidth, max || normalized.minWidth);
  const expanded = clampAgentPanelWidth(Math.floor(viewportWidth * normalized.expandedRatio), { min, max });
  return {
    min,
    max,
    expanded,
    compact: viewportWidth > 0 && viewportWidth <= normalized.mobileBreakpoint,
  };
}

export function clampAgentPanelWidth(value: number, bounds: Pick<AgentPanelWidthBounds, 'min' | 'max'>): number {
  if (!Number.isFinite(value)) return bounds.min;
  return Math.min(bounds.max, Math.max(bounds.min, Math.round(value)));
}

export function useAgentPanelLayout(options: AgentPanelLayoutOptions = {}): UseAgentPanelLayoutResult {
  const normalized = useMemo(
    () => normalizeOptions(options),
    [
      options.defaultWidth,
      options.expandedRatio,
      options.horizontalMargin,
      options.maxWidth,
      options.minWidth,
      options.mobileBreakpoint,
      options.storageKey,
    ]
  );
  const [viewportWidth, setViewportWidth] = useState(windowWidth);
  const [width, setWidth] = useState(() => normalized.defaultWidth);
  const [savedWidth, setSavedWidth] = useState(() => normalized.defaultWidth);
  const [isExpanded, setIsExpanded] = useState(false);
  const [isResizing, setIsResizing] = useState(false);

  const bounds = useMemo(
    () => getAgentPanelWidthBounds(viewportWidth, normalized),
    [normalized, viewportWidth]
  );
  const canResize = !bounds.compact && bounds.max > bounds.min;
  const effectiveWidth = bounds.compact
    ? bounds.max
    : isExpanded
      ? bounds.expanded
      : clampAgentPanelWidth(width, bounds);

  const boundsRef = useRef(bounds);
  const widthRef = useRef(width);
  const savedWidthRef = useRef(savedWidth);
  const expandedRef = useRef(isExpanded);

  useEffect(() => {
    boundsRef.current = bounds;
    widthRef.current = effectiveWidth;
    savedWidthRef.current = savedWidth;
    expandedRef.current = isExpanded;
  }, [bounds, effectiveWidth, isExpanded, savedWidth]);

  useEffect(() => {
    const onResize = () => setViewportWidth(windowWidth());
    window.addEventListener('resize', onResize);
    return () => window.removeEventListener('resize', onResize);
  }, []);

  useEffect(() => {
    const stored = readLayout(normalized.storageKey);
    const nextWidth = clampAgentPanelWidth(stored?.width ?? normalized.defaultWidth, boundsRef.current);
    setWidth(nextWidth);
    setSavedWidth(nextWidth);
    setIsExpanded(stored?.expanded === true && !boundsRef.current.compact);
  }, [normalized.defaultWidth, normalized.storageKey]);

  useEffect(() => {
    setWidth((current) => clampAgentPanelWidth(current, bounds));
    setSavedWidth((current) => clampAgentPanelWidth(current, bounds));
  }, [bounds]);

  const persist = useCallback(
    (nextWidth: number, nextExpanded: boolean) => {
      writeLayout(normalized.storageKey, {
        width: clampAgentPanelWidth(nextWidth, boundsRef.current),
        expanded: nextExpanded,
      });
    },
    [normalized.storageKey]
  );

  const commitWidth = useCallback(
    (nextWidth: number, nextExpanded = false) => {
      const clamped = clampAgentPanelWidth(nextWidth, boundsRef.current);
      setWidth(clamped);
      if (!nextExpanded) {
        setSavedWidth(clamped);
      }
      setIsExpanded(nextExpanded);
      persist(clamped, nextExpanded);
    },
    [persist]
  );

  const toggleExpanded = useCallback(() => {
    if (boundsRef.current.compact) return;
    if (expandedRef.current) {
      commitWidth(savedWidthRef.current || normalized.defaultWidth, false);
      return;
    }
    const current = clampAgentPanelWidth(widthRef.current, boundsRef.current);
    setSavedWidth(current);
    savedWidthRef.current = current;
    commitWidth(boundsRef.current.expanded, true);
  }, [commitWidth, normalized.defaultWidth]);

  const resetWidth = useCallback(() => {
    commitWidth(normalized.defaultWidth, false);
  }, [commitWidth, normalized.defaultWidth]);

  const resizeBy = useCallback(
    (delta: number) => {
      if (!canResize) return;
      commitWidth(widthRef.current + delta, false);
    },
    [canResize, commitWidth]
  );

  const onPointerDown = useCallback(
    (event: PointerEvent<HTMLDivElement>) => {
      if (!canResize) return;
      event.preventDefault();
      const startX = event.clientX;
      const startWidth = clampAgentPanelWidth(widthRef.current, boundsRef.current);
      let nextWidth = startWidth;
      setIsResizing(true);
      setIsExpanded(false);

      const originalCursor = document.body.style.cursor;
      const originalUserSelect = document.body.style.userSelect;
      document.body.style.cursor = 'ew-resize';
      document.body.style.userSelect = 'none';

      const onPointerMove = (moveEvent: globalThis.PointerEvent) => {
        nextWidth = clampAgentPanelWidth(startWidth + startX - moveEvent.clientX, boundsRef.current);
        setWidth(nextWidth);
        setSavedWidth(nextWidth);
      };
      const onPointerUp = () => {
        window.removeEventListener('pointermove', onPointerMove);
        window.removeEventListener('pointerup', onPointerUp);
        document.body.style.cursor = originalCursor;
        document.body.style.userSelect = originalUserSelect;
        setIsResizing(false);
        persist(nextWidth, false);
      };

      window.addEventListener('pointermove', onPointerMove);
      window.addEventListener('pointerup', onPointerUp, { once: true });
    },
    [canResize, persist]
  );

  const onKeyDown = useCallback(
    (event: KeyboardEvent<HTMLDivElement>) => {
      if (!canResize) return;
      if (event.key === 'ArrowLeft') {
        event.preventDefault();
        resizeBy(KEYBOARD_STEP);
      } else if (event.key === 'ArrowRight') {
        event.preventDefault();
        resizeBy(-KEYBOARD_STEP);
      } else if (event.key === 'Home') {
        event.preventDefault();
        commitWidth(boundsRef.current.min, false);
      } else if (event.key === 'End') {
        event.preventDefault();
        commitWidth(boundsRef.current.expanded, true);
      } else if (event.key === 'Enter' || event.key === ' ') {
        event.preventDefault();
        toggleExpanded();
      }
    },
    [canResize, commitWidth, resizeBy, toggleExpanded]
  );

  return {
    width: effectiveWidth,
    isExpanded: !bounds.compact && isExpanded,
    isResizing,
    canResize,
    panelStyle: {
      width: effectiveWidth,
      maxWidth: normalized.horizontalMargin > 0 ? `calc(100vw - ${normalized.horizontalMargin}px)` : '100vw',
    },
    resizeHandleProps: {
      role: 'separator',
      'aria-orientation': 'vertical',
      'aria-valuemin': bounds.min,
      'aria-valuemax': bounds.max,
      'aria-valuenow': effectiveWidth,
      tabIndex: canResize ? 0 : -1,
      onPointerDown,
      onDoubleClick: resetWidth,
      onKeyDown,
    },
    toggleExpanded,
    resetWidth,
  };
}
