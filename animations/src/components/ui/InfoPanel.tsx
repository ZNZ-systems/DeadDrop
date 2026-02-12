import React from "react";
import { fontFamilyHeading, fontFamilyMono } from "../../fonts/load";
import { COLORS } from "../../constants/theme";

type InfoPanelVariant = "default" | "warn" | "success";

interface InfoPanelProps {
  title: string;
  children: React.ReactNode;
  variant: InfoPanelVariant;
  style?: React.CSSProperties;
}

const getBorderColor = (variant: InfoPanelVariant): string => {
  switch (variant) {
    case "warn":
      return COLORS.yellow;
    case "success":
      return COLORS.green;
    case "default":
    default:
      return COLORS.black;
  }
};

export const InfoPanel: React.FC<InfoPanelProps> = ({
  title,
  children,
  variant,
  style,
}) => {
  const borderColor = getBorderColor(variant);

  const panelStyle: React.CSSProperties = {
    border: `4px solid ${borderColor}`,
    padding: "2rem",
    boxSizing: "border-box",
    fontFamily: fontFamilyMono,
    fontSize: 12,
    color: COLORS.gray,
    lineHeight: 1.8,
  };

  const titleStyle: React.CSSProperties = {
    fontFamily: fontFamilyHeading,
    fontSize: 16,
    fontWeight: 700,
    marginBottom: "0.75rem",
    color: COLORS.black,
    letterSpacing: "-0.02em",
  };

  return (
    <div style={{ ...panelStyle, ...style }}>
      <div style={titleStyle}>{title}</div>
      {children}
    </div>
  );
};
