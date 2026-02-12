import React from "react";

interface WidgetSuccessProps {
  style?: React.CSSProperties;
}

/**
 * Success state panel for the DeadDrop widget, shown after a form
 * submission. Displays a green checkmark circle with confirmation text.
 * Shares the same outer shell dimensions and positioning as WidgetPanel.
 */
export const WidgetSuccess: React.FC<WidgetSuccessProps> = ({ style }) => {
  return (
    <div
      style={{
        position: "absolute",
        bottom: 92,
        right: 24,
        width: 370,
        borderRadius: 12,
        overflow: "hidden",
        boxShadow: "0 8px 30px rgba(0,0,0,0.18)",
        ...style,
      }}
    >
      {/* ---- Header ---- */}
      <div
        style={{
          backgroundColor: "#1a1a2e",
          padding: "18px 20px",
          display: "flex",
          alignItems: "center",
          justifyContent: "space-between",
        }}
      >
        <span
          style={{
            color: "#ffffff",
            fontSize: 16,
            fontWeight: 600,
            fontFamily: "system-ui, -apple-system, sans-serif",
          }}
        >
          Send us a message
        </span>
        <span
          style={{
            color: "rgba(255,255,255,0.6)",
            fontSize: 18,
            cursor: "pointer",
            lineHeight: 1,
            fontFamily: "system-ui, -apple-system, sans-serif",
          }}
        >
          {"\u2715"}
        </span>
      </div>

      {/* ---- Body ---- */}
      <div
        style={{
          backgroundColor: "#ffffff",
          padding: "40px 20px",
          display: "flex",
          flexDirection: "column",
          alignItems: "center",
          justifyContent: "center",
          fontFamily: "system-ui, -apple-system, sans-serif",
        }}
      >
        {/* Checkmark circle */}
        <div
          style={{
            width: 40,
            height: 40,
            borderRadius: "50%",
            backgroundColor: "#ecfdf5",
            border: "1px solid #a7f3d0",
            display: "flex",
            alignItems: "center",
            justifyContent: "center",
            marginBottom: 16,
          }}
        >
          <svg
            width={20}
            height={20}
            viewBox="0 0 24 24"
            fill="none"
            xmlns="http://www.w3.org/2000/svg"
          >
            <path
              d="M5 13l4 4L19 7"
              stroke="#22c55e"
              strokeWidth={2.5}
              strokeLinecap="round"
              strokeLinejoin="round"
            />
          </svg>
        </div>

        {/* Success text */}
        <span
          style={{
            fontSize: 16,
            fontWeight: 600,
            color: "#065f46",
            marginBottom: 6,
          }}
        >
          Message sent!
        </span>

        {/* Subtext */}
        <span
          style={{
            fontSize: 13,
            color: "#6b7280",
            textAlign: "center",
            lineHeight: 1.5,
            maxWidth: 260,
          }}
        >
          Thank you for reaching out. We'll get back to you soon.
        </span>
      </div>
    </div>
  );
};
