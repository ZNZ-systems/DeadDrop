import { spring, SpringConfig } from "remotion";
import { SPRING_PRESETS } from "../constants/theme";

export const smoothSpring = (
  frame: number,
  fps: number,
  delay = 0,
): number => {
  return spring({ frame, fps, config: SPRING_PRESETS.smooth, delay });
};

export const snappySpring = (
  frame: number,
  fps: number,
  delay = 0,
): number => {
  return spring({ frame, fps, config: SPRING_PRESETS.snappy, delay });
};

export const bouncySpring = (
  frame: number,
  fps: number,
  delay = 0,
): number => {
  return spring({ frame, fps, config: SPRING_PRESETS.bouncy, delay });
};

export const delayedSpring = (
  frame: number,
  fps: number,
  delay: number,
  config: SpringConfig = SPRING_PRESETS.smooth,
): number => {
  return spring({ frame, fps, config, delay });
};
