import React from "react";
import { fontFamilyMono } from "../../fonts/load";
import { COLORS } from "../../constants/theme";

type ButtonVariant = "primary" | "red" | "outline" | "outline-red";
type ButtonSize = "default" | "sm";

interface ButtonProps {
  label: string;
  variant: ButtonVariant;
  size?: ButtonSize;
  hovered?: boolean;
  style?: React.CSSProperties;
}

const getBaseStyle = (
  variant: ButtonVariant,
  size: ButtonSize,
): React.CSSProperties => {
  const isSmall = size === "sm";

  const shared: React.CSSProperties = {
    display: "inline-block",
    fontFamily: fontFamilyMono,
    fontSize: isSmall ? 10 : 11,
    fontWeight: 700,
    textTransform: "uppercase",
    letterSpacing: "0.12em",
    cursor: "pointer",
    textDecoration: "none",
    lineHeight: 1.6,
    boxSizing: "border-box",
  };

  switch (variant) {
    case "primary":
      return {
        ...shared,
        padding: isSmall ? "8px 16px" : "14px 28px",
        background: COLORS.black,
        color: COLORS.white,
        border: "none",
      };
    case "red":
      return {
        ...shared,
        padding: isSmall ? "8px 16px" : "14px 28px",
        background: COLORS.red,
        color: COLORS.white,
        border: "none",
      };
    case "outline":
      return {
        ...shared,
        padding: isSmall ? "8px 16px" : "12px 24px",
        background: "transparent",
        color: COLORS.black,
        border: `2px solid ${COLORS.black}`,
      };
    case "outline-red":
      return {
        ...shared,
        padding: isSmall ? "8px 16px" : "12px 24px",
        background: "transparent",
        color: COLORS.red,
        border: `2px solid ${COLORS.red}`,
      };
  }
};

const getHoveredOverrides = (
  variant: ButtonVariant,
): React.CSSProperties => {
  switch (variant) {
    case "primary":
      return { background: COLORS.red, color: COLORS.white };
    case "red":
      return { background: COLORS.black, color: COLORS.white };
    case "outline":
      return { background: COLORS.black, color: COLORS.white };
    case "outline-red":
      return { background: COLORS.red, color: COLORS.white };
  }
};

export const Button: React.FC<ButtonProps> = ({
  label,
  variant,
  size = "default",
  hovered = false,
  style,
}) => {
  const base = getBaseStyle(variant, size);
  const hover = hovered ? getHoveredOverrides(variant) : {};

  return (
    <div style={{ ...base, ...hover, ...style }}>
      {label}
    </div>
  );
};
