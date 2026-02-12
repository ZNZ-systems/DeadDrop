export const COLORS = {
  bg: "#f5f0e8",
  black: "#0a0a0a",
  red: "#ff2200",
  green: "#22c55e",
  yellow: "#eab308",
  gray: "#888888",
  white: "#f5f0e8",
  codeBg: "#0a0a0a",
  codeText: "#aaaaaa",
  codeHl: "#ff2200",
  codeVal: "#ffd700",
} as const;

export const FONTS = {
  heading: "'Space Grotesk', system-ui, sans-serif",
  mono: "'Space Mono', monospace",
} as const;

export const BORDERS = {
  thick: `4px solid ${COLORS.black}`,
  thin: `1px solid ${COLORS.black}`,
  thickRed: `4px solid ${COLORS.red}`,
  thickGreen: `4px solid ${COLORS.green}`,
  thickYellow: `4px solid ${COLORS.yellow}`,
} as const;

export const SPRING_PRESETS = {
  smooth: { mass: 1, damping: 200, stiffness: 100, overshootClamping: false },
  snappy: { mass: 1, damping: 20, stiffness: 200, overshootClamping: false },
  bouncy: { mass: 1, damping: 8, stiffness: 100, overshootClamping: false },
  heavy: { mass: 2, damping: 15, stiffness: 80, overshootClamping: false },
} as const;

export const VIDEO = {
  width: 1920,
  height: 1080,
  fps: 30,
} as const;
