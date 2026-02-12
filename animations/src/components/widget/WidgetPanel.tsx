import React from "react";

interface WidgetPanelProps {
  isOpen: boolean;
  nameValue?: string;
  emailValue?: string;
  messageValue?: string;
  style?: React.CSSProperties;
}

const labelStyle: React.CSSProperties = {
  display: "block",
  fontSize: 13,
  fontWeight: 500,
  color: "#374151",
  marginBottom: 5,
};

const inputStyle: React.CSSProperties = {
  display: "block",
  width: "100%",
  padding: "10px 12px",
  border: "1px solid #d1d5db",
  borderRadius: 8,
  backgroundColor: "#f9fafb",
  fontSize: 14,
  fontFamily: "inherit",
  color: "#1f2937",
  outline: "none",
  boxSizing: "border-box",
};

/**
 * Contact form panel for the DeadDrop widget. Displays a header, three
 * form fields (name, email, message), and a submit button. Visibility
 * and positioning are controlled via the isOpen prop -- when closed the
 * panel is fully transparent and shifted down 20px.
 */
export const WidgetPanel: React.FC<WidgetPanelProps> = ({
  isOpen,
  nameValue = "",
  emailValue = "",
  messageValue = "",
  style,
}) => {
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
        opacity: isOpen ? 1 : 0,
        transform: isOpen ? "translateY(0px)" : "translateY(20px)",
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
          padding: 20,
          fontFamily: "system-ui, -apple-system, sans-serif",
        }}
      >
        {/* Name field */}
        <div style={{ marginBottom: 14 }}>
          <label style={labelStyle}>
            Name <span style={{ color: "#9ca3af", fontWeight: 400 }}>(optional)</span>
          </label>
          <input
            style={inputStyle}
            value={nameValue}
            readOnly
            placeholder="Your name"
          />
        </div>

        {/* Email field */}
        <div style={{ marginBottom: 14 }}>
          <label style={labelStyle}>
            Email <span style={{ color: "#9ca3af", fontWeight: 400 }}>(optional)</span>
          </label>
          <input
            style={inputStyle}
            value={emailValue}
            readOnly
            placeholder="you@example.com"
          />
        </div>

        {/* Message field */}
        <div style={{ marginBottom: 18 }}>
          <label style={labelStyle}>Message</label>
          <textarea
            style={{
              ...inputStyle,
              minHeight: 100,
              resize: "none",
              lineHeight: 1.5,
            }}
            value={messageValue}
            readOnly
            placeholder="How can we help?"
          />
        </div>

        {/* Submit button */}
        <button
          style={{
            display: "block",
            width: "100%",
            backgroundColor: "#1a1a2e",
            color: "#ffffff",
            border: "none",
            padding: 12,
            borderRadius: 8,
            fontSize: 15,
            fontWeight: 600,
            fontFamily: "system-ui, -apple-system, sans-serif",
            cursor: "pointer",
            textAlign: "center",
          }}
        >
          Send Message
        </button>
      </div>
    </div>
  );
};
