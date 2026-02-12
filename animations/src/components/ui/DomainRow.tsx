import React from "react";
import { fontFamilyHeading, fontFamilyMono } from "../../fonts/load";
import { COLORS, BORDERS } from "../../constants/theme";
import { Badge } from "./Badge";

interface DomainRowProps {
  name: string;
  verified: boolean;
  unreadCount?: number;
  hovered?: boolean;
  style?: React.CSSProperties;
}

export const DomainRow: React.FC<DomainRowProps> = ({
  name,
  verified,
  unreadCount = 0,
  hovered = false,
  style,
}) => {
  const rowStyle: React.CSSProperties = {
    padding: "1.25rem 1.5rem",
    borderBottom: BORDERS.thick,
    display: "flex",
    alignItems: "center",
    justifyContent: "space-between",
    textDecoration: "none",
    color: hovered ? COLORS.white : COLORS.black,
    background: hovered ? COLORS.black : "transparent",
    boxSizing: "border-box",
    fontFamily: fontFamilyMono,
  };

  const nameStyle: React.CSSProperties = {
    fontFamily: fontFamilyHeading,
    fontSize: 16,
    fontWeight: 700,
    letterSpacing: "-0.02em",
  };

  const leftStyle: React.CSSProperties = {
    display: "flex",
    alignItems: "center",
    gap: "0.75rem",
  };

  /* When hovered, verified badge becomes red bg + white text; unverified becomes yellow bg + black text */
  const hoveredVerifiedBadgeStyle: React.CSSProperties = hovered
    ? {
        background: COLORS.red,
        color: COLORS.white,
        borderColor: COLORS.red,
      }
    : {};

  const hoveredUnverifiedBadgeStyle: React.CSSProperties = hovered
    ? {
        background: COLORS.yellow,
        color: COLORS.black,
        borderColor: COLORS.yellow,
      }
    : {};

  const hoveredCountBadgeStyle: React.CSSProperties = hovered
    ? {
        background: COLORS.red,
        color: COLORS.white,
        borderColor: COLORS.red,
      }
    : {};

  return (
    <div style={{ ...rowStyle, ...style }}>
      <div style={leftStyle}>
        <span style={nameStyle}>{name}</span>
        {verified ? (
          <Badge
            text="Verified"
            variant="verified"
            style={hoveredVerifiedBadgeStyle}
          />
        ) : (
          <Badge
            text="Unverified"
            variant="unverified"
            style={hoveredUnverifiedBadgeStyle}
          />
        )}
      </div>
      {unreadCount > 0 && (
        <Badge
          text={`${unreadCount} unread`}
          variant="count"
          style={hoveredCountBadgeStyle}
        />
      )}
    </div>
  );
};
