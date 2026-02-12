import React from "react";
import { useCurrentFrame, interpolate } from "remotion";

interface CursorProps {
  x: number;
  y: number;
  clicking?: boolean;
  visible?: boolean;
  style?: React.CSSProperties;
}

/**
 * Animated mouse cursor that renders at (x, y) with an optional click
 * ripple effect. All animation is driven by Remotion's useCurrentFrame().
 */
export const Cursor: React.FC<CursorProps> = ({
  x,
  y,
  clicking = false,
  visible = true,
  style,
}) => {
  const frame = useCurrentFrame();

  const rippleRadius = clicking
    ? interpolate(frame % 16, [0, 8], [0, 20], {
        extrapolateLeft: "clamp",
        extrapolateRight: "clamp",
      })
    : 0;

  const rippleOpacity = clicking
    ? interpolate(frame % 16, [0, 8], [0.5, 0], {
        extrapolateLeft: "clamp",
        extrapolateRight: "clamp",
      })
    : 0;

  return (
    <div
      style={{
        position: "absolute",
        left: x,
        top: y,
        opacity: visible ? 1 : 0,
        pointerEvents: "none",
        ...style,
      }}
    >
      {/* Click ripple */}
      {clicking && (
        <div
          style={{
            position: "absolute",
            left: 0,
            top: 0,
            width: rippleRadius * 2,
            height: rippleRadius * 2,
            borderRadius: "50%",
            backgroundColor: "rgba(255, 34, 0, 0.4)",
            opacity: rippleOpacity,
            transform: "translate(-50%, -50%)",
          }}
        />
      )}

      {/* Cursor SVG */}
      <svg
        width={20}
        height={20}
        viewBox="0 0 24 24"
        style={{
          filter: "drop-shadow(1px 2px 3px rgba(0,0,0,0.35))",
          display: "block",
        }}
      >
        <path
          d="M5 3l14 10-6.5 1.5L9 21z"
          fill="#0a0a0a"
          stroke="#ffffff"
          strokeWidth={1.5}
          strokeLinejoin="round"
        />
      </svg>
    </div>
  );
};
