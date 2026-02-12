import React from "react";
import { useCurrentFrame, interpolate } from "remotion";

interface TypeWriterProps {
  text: string;
  startFrame: number;
  framesPerChar?: number;
  showCursor?: boolean;
  style?: React.CSSProperties;
}

/**
 * Reveals text character-by-character over time using Remotion's frame
 * clock. Optionally shows a blinking caret that disappears once all
 * characters have been typed.
 */
export const TypeWriter: React.FC<TypeWriterProps> = ({
  text,
  startFrame,
  framesPerChar = 2,
  showCursor = true,
  style,
}) => {
  const frame = useCurrentFrame();

  const elapsed = Math.max(0, frame - startFrame);
  const charsToShow = Math.min(
    text.length,
    Math.floor(elapsed / framesPerChar),
  );
  const revealedText = frame >= startFrame ? text.slice(0, charsToShow) : "";

  const allTyped = charsToShow >= text.length;
  const finishFrame = startFrame + text.length * framesPerChar;
  const holdElapsed = frame - finishFrame;

  // Cursor blinks via interpolate on a 16-frame cycle
  const cursorOpacity = interpolate(frame % 16, [0, 4, 8, 12, 16], [1, 1, 0, 0, 1], {
    extrapolateLeft: "clamp",
    extrapolateRight: "clamp",
  });

  // Hide cursor after text is fully typed + a short hold
  const cursorVisible = showCursor && !(allTyped && holdElapsed > 10);

  return (
    <span style={style}>
      {revealedText}
      {cursorVisible && (
        <span
          style={{
            opacity: cursorOpacity,
            userSelect: "none",
          }}
        >
          {"\u258C"}
        </span>
      )}
    </span>
  );
};
