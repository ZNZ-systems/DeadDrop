import React from "react";
import { fontFamilyMono } from "../../fonts/load";
import { COLORS } from "../../constants/theme";

type BadgeVariant = "verified" | "unverified" | "red" | "count";

interface BadgeProps {
  text: string;
  variant: BadgeVariant;
  style?: React.CSSProperties;
}

const getVariantStyle = (variant: BadgeVariant): React.CSSProperties => {
  switch (variant) {
    case "verified":
      return {
        border: `2px solid ${COLORS.green}`,
        color: COLORS.green,
      };
    case "unverified":
      return {
        border: `2px solid ${COLORS.yellow}`,
        color: COLORS.yellow,
      };
    case "red":
      return {
        border: `2px solid ${COLORS.red}`,
        color: COLORS.red,
      };
    case "count":
      return {
        border: `2px solid ${COLORS.red}`,
        color: COLORS.red,
      };
  }
};

export const Badge: React.FC<BadgeProps> = ({ text, variant, style }) => {
  const baseStyle: React.CSSProperties = {
    display: "inline-block",
    fontFamily: fontFamilyMono,
    fontSize: 10,
    fontWeight: 700,
    letterSpacing: "0.15em",
    textTransform: "uppercase",
    padding: "4px 12px",
    lineHeight: 1.6,
    boxSizing: "border-box",
  };

  const variantStyle = getVariantStyle(variant);

  return (
    <span style={{ ...baseStyle, ...variantStyle, ...style }}>
      {text}
    </span>
  );
};
