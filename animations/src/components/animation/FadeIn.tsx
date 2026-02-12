import React from "react";
import { useCurrentFrame, interpolate } from "remotion";

interface FadeInProps {
  startFrame?: number;
  durationFrames?: number;
  direction?: "up" | "down" | "none";
  children: React.ReactNode;
  style?: React.CSSProperties;
}

/**
 * Generic fade-in wrapper. Fades children from transparent to opaque,
 * optionally sliding them in from a given direction. All animation is
 * driven purely by Remotion's interpolate().
 */
export const FadeIn: React.FC<FadeInProps> = ({
  startFrame = 0,
  durationFrames = 15,
  direction = "up",
  children,
  style,
}) => {
  const frame = useCurrentFrame();

  const opacity = interpolate(
    frame,
    [startFrame, startFrame + durationFrames],
    [0, 1],
    { extrapolateLeft: "clamp", extrapolateRight: "clamp" },
  );

  let translateY = 0;
  if (direction === "up") {
    translateY = interpolate(
      frame,
      [startFrame, startFrame + durationFrames],
      [8, 0],
      { extrapolateLeft: "clamp", extrapolateRight: "clamp" },
    );
  } else if (direction === "down") {
    translateY = interpolate(
      frame,
      [startFrame, startFrame + durationFrames],
      [-8, 0],
      { extrapolateLeft: "clamp", extrapolateRight: "clamp" },
    );
  }

  return (
    <div
      style={{
        opacity,
        transform: direction !== "none" ? `translateY(${translateY}px)` : undefined,
        ...style,
      }}
    >
      {children}
    </div>
  );
};
