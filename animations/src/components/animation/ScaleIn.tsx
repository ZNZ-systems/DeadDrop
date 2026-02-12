import React from "react";
import {
  useCurrentFrame,
  useVideoConfig,
  spring,
  interpolate,
  SpringConfig,
} from "remotion";
import { SPRING_PRESETS } from "../../constants/theme";

interface ScaleInProps {
  startFrame?: number;
  config?: SpringConfig;
  children: React.ReactNode;
  style?: React.CSSProperties;
}

/**
 * Scale-in wrapper using a spring animation. Pops children from 50% to
 * 100% scale while fading in. Uses the "snappy" spring preset by
 * default (damping: 20, stiffness: 200) for a quick, punchy entrance.
 */
export const ScaleIn: React.FC<ScaleInProps> = ({
  startFrame = 0,
  config = SPRING_PRESETS.snappy,
  children,
  style,
}) => {
  const frame = useCurrentFrame();
  const { fps } = useVideoConfig();

  const progress = spring({
    frame,
    fps,
    config,
    delay: startFrame,
  });

  const scale = interpolate(progress, [0, 1], [0.5, 1]);
  const opacity = progress;

  return (
    <div
      style={{
        opacity,
        transform: `scale(${scale})`,
        ...style,
      }}
    >
      {children}
    </div>
  );
};
