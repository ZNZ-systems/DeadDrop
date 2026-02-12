import React from "react";
import { fontFamilyHeading, fontFamilyMono } from "../../fonts/load";
import { COLORS, BORDERS } from "../../constants/theme";

interface NavBarProps {
  email?: string;
  style?: React.CSSProperties;
}

export const NavBar: React.FC<NavBarProps> = ({ email, style }) => {
  const navStyle: React.CSSProperties = {
    padding: "1rem 2rem",
    display: "flex",
    alignItems: "center",
    justifyContent: "space-between",
    background: COLORS.bg,
    borderBottom: BORDERS.thick,
    boxSizing: "border-box",
  };

  const logoStyle: React.CSSProperties = {
    fontFamily: fontFamilyHeading,
    fontSize: 18,
    fontWeight: 700,
    textTransform: "uppercase",
    letterSpacing: "-0.02em",
    color: COLORS.black,
    textDecoration: "none",
  };

  const logoRedStyle: React.CSSProperties = {
    color: COLORS.red,
  };

  const navRightStyle: React.CSSProperties = {
    display: "flex",
    alignItems: "center",
    gap: "1.5rem",
  };

  const navEmailStyle: React.CSSProperties = {
    fontFamily: fontFamilyMono,
    fontSize: 11,
    color: COLORS.gray,
    letterSpacing: "0.05em",
  };

  const navLinkStyle: React.CSSProperties = {
    fontFamily: fontFamilyMono,
    fontSize: 11,
    fontWeight: 700,
    textDecoration: "none",
    color: COLORS.black,
    textTransform: "uppercase",
    letterSpacing: "0.1em",
  };

  const navCtaStyle: React.CSSProperties = {
    fontFamily: fontFamilyMono,
    fontSize: 11,
    fontWeight: 700,
    padding: "10px 24px",
    background: COLORS.black,
    color: COLORS.white,
    border: "none",
    textDecoration: "none",
    textTransform: "uppercase",
    letterSpacing: "0.1em",
    cursor: "pointer",
  };

  const logoutBtnStyle: React.CSSProperties = {
    fontFamily: fontFamilyMono,
    fontSize: 11,
    fontWeight: 700,
    padding: "10px 24px",
    background: "transparent",
    color: COLORS.red,
    border: `2px solid ${COLORS.red}`,
    textDecoration: "none",
    textTransform: "uppercase",
    letterSpacing: "0.1em",
    cursor: "pointer",
  };

  return (
    <nav style={{ ...navStyle, ...style }}>
      <div style={logoStyle}>
        Dead<span style={logoRedStyle}>Drop</span>
      </div>
      <div style={navRightStyle}>
        {email ? (
          <>
            <span style={navEmailStyle}>{email}</span>
            <div style={logoutBtnStyle}>Logout</div>
          </>
        ) : (
          <>
            <span style={navLinkStyle}>Login</span>
            <div style={navCtaStyle}>Sign Up</div>
          </>
        )}
      </div>
    </nav>
  );
};
