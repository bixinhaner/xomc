import React, { useMemo } from 'react';
import { useAppStore } from '@/store/appStore';

export interface FloatingElementProps {
  children: React.ReactNode;
  /** Float amplitude in px (default 6) */
  amplitude?: number;
  /** Animation duration in ms (default 3000) */
  speed?: number;
  /** Animation delay in ms (default 0) */
  delay?: number;
  className?: string;
  style?: React.CSSProperties;
}

const FloatingElement: React.FC<FloatingElementProps> = ({
  children,
  amplitude = 6,
  speed = 3000,
  delay = 0,
  className,
  style,
}) => {
  const effects3D = useAppStore((s) => s.effects3DEnabled);

  const animStyle = useMemo<React.CSSProperties>(
    () => effects3D
      ? {
          animation: `omc-float ${speed}ms ease-in-out ${delay}ms infinite`,
          '--float-amplitude': `${amplitude}px`,
          ...style,
        } as React.CSSProperties
      : { ...style } as React.CSSProperties,
    [amplitude, speed, delay, style, effects3D],
  );

  return (
    <div className={className} style={animStyle}>
      {children}
    </div>
  );
};

export default FloatingElement;
