import React from "react";
import { fontFamilyMono } from "../../fonts/load";
import { COLORS, BORDERS } from "../../constants/theme";

interface CodeBlockProps {
  children: React.ReactNode;
  style?: React.CSSProperties;
}

export const CodeBlock: React.FC<CodeBlockProps> = ({ children, style }) => {
  const blockStyle: React.CSSProperties = {
    background: COLORS.codeBg,
    color: COLORS.codeText,
    padding: "1rem 1.25rem",
    fontFamily: fontFamilyMono,
    fontSize: 12,
    lineHeight: 1.8,
    border: BORDERS.thick,
    wordBreak: "break-all",
    whiteSpace: "pre-wrap",
    boxSizing: "border-box",
  };

  return (
    <pre style={{ ...blockStyle, ...style, margin: 0 }}>
      {children}
    </pre>
  );
};

/* Syntax highlight span -- red keywords / tags */
interface CodeHLProps {
  children: React.ReactNode;
}

export const CodeHL: React.FC<CodeHLProps> = ({ children }) => (
  <span style={{ color: COLORS.codeHl }}>{children}</span>
);

/* Syntax highlight span -- gold values */
interface CodeValProps {
  children: React.ReactNode;
}

export const CodeVal: React.FC<CodeValProps> = ({ children }) => (
  <span style={{ color: COLORS.codeVal }}>{children}</span>
);
