import React from "react";
import { fontFamilyMono } from "../../fonts/load";
import { COLORS, BORDERS } from "../../constants/theme";

interface InputProps {
  value: string;
  placeholder?: string;
  focused?: boolean;
  label?: string;
  hint?: string;
  style?: React.CSSProperties;
}

export const Input: React.FC<InputProps> = ({
  value,
  placeholder,
  focused = false,
  label,
  hint,
  style,
}) => {
  const wrapperStyle: React.CSSProperties = {
    marginBottom: 0,
    ...style,
  };

  const labelStyle: React.CSSProperties = {
    display: "block",
    fontFamily: fontFamilyMono,
    fontSize: 11,
    fontWeight: 700,
    textTransform: "uppercase",
    letterSpacing: "0.15em",
    marginBottom: "0.5rem",
    color: COLORS.black,
  };

  const inputStyle: React.CSSProperties = {
    width: "100%",
    padding: "14px 16px",
    background: COLORS.white,
    border: BORDERS.thick,
    borderColor: focused ? COLORS.red : COLORS.black,
    fontFamily: fontFamilyMono,
    fontSize: 13,
    lineHeight: 1.6,
    color: value ? COLORS.black : COLORS.gray,
    outline: "none",
    boxSizing: "border-box",
  };

  const hintStyle: React.CSSProperties = {
    fontFamily: fontFamilyMono,
    fontSize: 11,
    color: COLORS.gray,
    marginTop: "0.4rem",
    letterSpacing: "0.02em",
  };

  const displayText = value || placeholder || "";

  return (
    <div style={wrapperStyle}>
      {label && <div style={labelStyle}>{label}</div>}
      <div style={inputStyle}>
        {displayText}
      </div>
      {hint && <div style={hintStyle}>{hint}</div>}
    </div>
  );
};
