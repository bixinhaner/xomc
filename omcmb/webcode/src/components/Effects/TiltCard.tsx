import React, { useRef } from 'react';
import { useAppStore } from '@/store/appStore';
import { use3DTilt } from '@/hooks/use3DTilt';
import { getThemeEffects } from './themeEffectConfig';

export interface TiltCardProps {
  children: React.ReactNode;
  /** Override max tilt angle in degrees */
  maxTilt?: number;
  /** Override perspective in px */
  perspective?: number;
  /** Override scale factor */
  scale?: number;
  /** Enable glare overlay */
  glare?: boolean;
  /** Enable dynamic shadow (default true) */
  shadow?: boolean;
  className?: string;
  style?: React.CSSProperties;
}

/**
 * 3D tilt card wrapper.
 *
 * Glare is driven entirely by CSS `--tilt-x` / `--tilt-y` custom properties
 * set by the use3DTilt hook — no JS gradient recalculation in React render.
 */
const TiltCard: React.FC<TiltCardProps> = ({
  children,
  maxTilt,
  perspective,
  scale,
  glare,
  shadow = true,
  className,
  style,
}) => {
  const ref = useRef<HTMLDivElement>(null);
  const theme = useAppStore((s) => s.theme);
  const effects3D = useAppStore((s) => s.effects3DEnabled);
  const config = getThemeEffects(theme);

  use3DTilt(ref, {
    maxTilt: maxTilt ?? config.tilt.maxTilt,
    perspective: perspective ?? config.tilt.perspective,
    scale: scale ?? config.tilt.scale,
    shadow,
    disabled: !effects3D,
  });

  if (!effects3D) {
    return <div className={className} style={style}>{children}</div>;
  }

  const showGlare = glare ?? config.tilt.glare;

  return (
    <div
      ref={ref}
      className={className}
      style={{
        position: 'relative',
        borderRadius: 'inherit',
        willChange: 'transform',
        ...style,
      }}
    >
      {children}
      {showGlare && (
        <div
          style={{
            position: 'absolute',
            inset: 0,
            borderRadius: 'inherit',
            pointerEvents: 'none',
            background:
              `radial-gradient(ellipse at calc(50% + calc(var(--tilt-x, 0) * 40%)) calc(50% + calc(var(--tilt-y, 0) * 40%)), ${config.tilt.glareColor}, transparent 70%)`,
            opacity: 0,
            transition: 'opacity 0.3s ease',
            zIndex: 1,
          }}
          className="tilt-glare"
        />
      )}
    </div>
  );
};

export default TiltCard;
