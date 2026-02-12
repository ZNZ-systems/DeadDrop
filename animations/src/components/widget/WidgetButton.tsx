import React from "react";

interface WidgetButtonProps {
  style?: React.CSSProperties;
}

/**
 * Floating trigger button for the DeadDrop contact widget. Renders as a
 * 56px circle in the bottom-right corner with a white mail icon on a
 * dark background. Uses absolute positioning for Remotion scenes.
 */
export const WidgetButton: React.FC<WidgetButtonProps> = ({ style }) => {
  return (
    <div
      style={{
        position: "absolute",
        bottom: 24,
        right: 24,
        width: 56,
        height: 56,
        borderRadius: "50%",
        backgroundColor: "#1a1a2e",
        display: "flex",
        alignItems: "center",
        justifyContent: "center",
        boxShadow: "0 4px 14px rgba(0,0,0,0.25)",
        cursor: "pointer",
        ...style,
      }}
    >
      <svg
        width={26}
        height={26}
        viewBox="0 0 24 24"
        fill="none"
        xmlns="http://www.w3.org/2000/svg"
      >
        <path
          d="M20 4H4c-1.1 0-2 .9-2 2v12c0 1.1.9 2 2 2h16c1.1 0 2-.9 2-2V6c0-1.1-.9-2-2-2zm0 4l-8 5-8-5V6l8 5 8-5v2z"
          fill="#ffffff"
        />
      </svg>
    </div>
  );
};
