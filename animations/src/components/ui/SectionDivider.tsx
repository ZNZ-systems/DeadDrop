import React from "react";
import { fontFamilyMono } from "../../fonts/load";
import { COLORS, BORDERS } from "../../constants/theme";

interface SectionDividerProps {
  num: string;
  label: string;
  style?: React.CSSProperties;
}

export const SectionDivider: React.FC<SectionDividerProps> = ({
  num,
  label,
  style,
}) => {
  const dividerStyle: React.CSSProperties = {
    borderTop: BORDERS.thick,
    borderBottom: BORDERS.thick,
    padding: "0.75rem 0",
    fontFamily: fontFamilyMono,
    fontSize: 11,
    fontWeight: 700,
    letterSpacing: "0.2em",
    textTransform: "uppercase",
    display: "flex",
    justifyContent: "space-between",
    alignItems: "center",
    boxSizing: "border-box",
  };

  const numStyle: React.CSSProperties = {
    color: COLORS.red,
  };

  const labelStyle: React.CSSProperties = {
    color: COLORS.black,
  };

  return (
    <div style={{ ...dividerStyle, ...style }}>
      <span style={numStyle}>{num}</span>
      <span style={labelStyle}>{label}</span>
    </div>
  );
};
