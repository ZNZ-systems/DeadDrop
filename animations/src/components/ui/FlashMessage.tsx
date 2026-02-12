import React from "react";
import { fontFamilyMono } from "../../fonts/load";
import { COLORS } from "../../constants/theme";

interface FlashMessageProps {
  message: string;
  type: "success" | "error";
  style?: React.CSSProperties;
}

export const FlashMessage: React.FC<FlashMessageProps> = ({
  message,
  type,
  style,
}) => {
  const isError = type === "error";

  const baseStyle: React.CSSProperties = {
    padding: "1rem 1.5rem",
    fontFamily: fontFamilyMono,
    fontSize: 12,
    fontWeight: 700,
    letterSpacing: "0.05em",
    textTransform: "uppercase",
    lineHeight: 1.6,
    border: `4px solid ${isError ? COLORS.red : COLORS.black}`,
    background: isError ? COLORS.red : COLORS.black,
    color: COLORS.white,
    boxSizing: "border-box",
  };

  return (
    <div style={{ ...baseStyle, ...style }}>
      {message}
    </div>
  );
};
