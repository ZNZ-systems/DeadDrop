import React from "react";
import { useCurrentFrame, useVideoConfig, spring, interpolate } from "remotion";
import { SPRING_PRESETS } from "../../constants/theme";

interface SlideInProps {
  startFrame?: number;
  direction?: "left" | "right" | "up" | "down";
  children: React.ReactNode;
  style?: React.CSSProperties;
}

/**
 * Slide-in wrapper using a spring animation. Slides children in from
 * 40px off-axis and simultaneously fades them in. Uses the "smooth"
 * spring preset (damping: 200) for a clean deceleration.
 */
export const SlideIn: React.FC<SlideInProps> = ({
  startFrame = 0,
  direction = "left",
  children,
  style,
}) => {
  const frame = useCurrentFrame();
  const { fps } = useVideoConfig();

  const progress = spring({
    frame,
    fps,
    config: SPRING_PRESETS.smooth,
    delay: startFrame,
  });

  const offsetMap: Record<string, { x: number; y: number }> = {
    left: { x: -40, y: 0 },
    right: { x: 40, y: 0 },
    up: { x: 0, y: -40 },
    down: { x: 0, y: 40 },
  };

  const offset = offsetMap[direction];
  const translateX = interpolate(progress, [0, 1], [offset.x, 0]);
  const translateY = interpolate(progress, [0, 1], [offset.y, 0]);
  const opacity = progress;

  return (
    <div
      style={{
        opacity,
        transform: `translate(${translateX}px, ${translateY}px)`,
        ...style,
      }}
    >
      {children}
    </div>
  );
};
