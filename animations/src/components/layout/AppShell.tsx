import React from "react";
import { COLORS } from "../../constants/theme";
import { NavBar } from "../ui/NavBar";

interface AppShellProps {
  /** The email displayed in the nav bar's account area. */
  email?: string;
  children: React.ReactNode;
  style?: React.CSSProperties;
}

/**
 * DeadDrop application layout shell (nav + main content area).
 * Intended to be placed **inside** a MockBrowser wrapper.
 *
 * Depends on `../ui/NavBar` -- make sure that component exists.
 */
export const AppShell: React.FC<AppShellProps> = ({
  email,
  children,
  style,
}) => {
  return (
    <div
      style={{
        width: "100%",
        height: "100%",
        backgroundColor: COLORS.bg,
        display: "flex",
        flexDirection: "column",
        ...style,
      }}
    >
      {/* Fixed-style nav bar */}
      <NavBar email={email} />

      {/* Main content area */}
      <div
        style={{
          flex: 1,
          maxWidth: 900,
          width: "100%",
          marginLeft: "auto",
          marginRight: "auto",
          paddingTop: 80,
          paddingLeft: "2rem",
          paddingRight: "2rem",
          paddingBottom: "4rem",
          boxSizing: "border-box",
        }}
      >
        {children}
      </div>
    </div>
  );
};
