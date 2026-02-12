import React from "react";
import { AbsoluteFill } from "remotion";
import { fontFamilyMono } from "../../fonts/load";

interface MockBrowserProps {
  url: string;
  children: React.ReactNode;
  style?: React.CSSProperties;
}

/**
 * Fake browser chrome wrapper that makes content look like it is being
 * viewed inside a web browser. Wraps every scene in the animation.
 */
export const MockBrowser: React.FC<MockBrowserProps> = ({
  url,
  children,
  style,
}) => {
  return (
    <AbsoluteFill
      style={{
        backgroundColor: "#e8e4dc",
        flexDirection: "column",
        ...style,
      }}
    >
      {/* ---- Browser chrome bar ---- */}
      <div
        style={{
          height: 52,
          backgroundColor: "#e0dcd4",
          display: "flex",
          flexDirection: "row",
          alignItems: "center",
          flexShrink: 0,
          borderBottom: "1px solid rgba(0,0,0,0.1)",
        }}
      >
        {/* Traffic-light dots */}
        <div
          style={{
            display: "flex",
            flexDirection: "row",
            alignItems: "center",
            gap: 8,
            marginLeft: 16,
            flexShrink: 0,
          }}
        >
          <div
            style={{
              width: 12,
              height: 12,
              borderRadius: "50%",
              backgroundColor: "#ff5f57",
            }}
          />
          <div
            style={{
              width: 12,
              height: 12,
              borderRadius: "50%",
              backgroundColor: "#ffbd2e",
            }}
          />
          <div
            style={{
              width: 12,
              height: 12,
              borderRadius: "50%",
              backgroundColor: "#28ca41",
            }}
          />
        </div>

        {/* URL bar */}
        <div
          style={{
            flex: 1,
            display: "flex",
            justifyContent: "center",
            alignItems: "center",
          }}
        >
          <div
            style={{
              backgroundColor: "#f5f0e8",
              borderRadius: 6,
              height: 30,
              flex: 1,
              maxWidth: 560,
              display: "flex",
              alignItems: "center",
              justifyContent: "center",
              padding: "0 16px",
            }}
          >
            <span
              style={{
                fontFamily: fontFamilyMono,
                fontSize: 12,
                color: "#888888",
                whiteSpace: "nowrap",
                overflow: "hidden",
                textOverflow: "ellipsis",
              }}
            >
              {url}
            </span>
          </div>
        </div>

        {/* Right spacer (balances the traffic lights) */}
        <div style={{ width: 16 + 12 * 3 + 8 * 2, flexShrink: 0 }} />
      </div>

      {/* ---- Content area ---- */}
      <div
        style={{
          flex: 1,
          position: "relative",
          overflow: "hidden",
          backgroundColor: "#f5f0e8",
        }}
      >
        {children}
      </div>
    </AbsoluteFill>
  );
};
