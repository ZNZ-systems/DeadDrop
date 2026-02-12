import React from "react";
import { fontFamilyHeading, fontFamilyMono } from "../../fonts/load";

interface DnsRecord {
  type: string;
  name: string;
  content: string;
  isNew?: boolean;
}

interface MockDnsPanelProps {
  records: DnsRecord[];
  style?: React.CSSProperties;
}

/**
 * Fake Cloudflare-style DNS management panel used in Scene 2
 * to show users adding TXT records for domain verification.
 */
export const MockDnsPanel: React.FC<MockDnsPanelProps> = ({
  records,
  style,
}) => {
  const gridColumns = "80px 1fr 2fr 100px";

  return (
    <div
      style={{
        width: "100%",
        height: "100%",
        backgroundColor: "#ffffff",
        display: "flex",
        flexDirection: "column",
        ...style,
      }}
    >
      {/* ---- Header bar ---- */}
      <div
        style={{
          backgroundColor: "#1a1a2e",
          padding: "1rem 2rem",
          display: "flex",
          justifyContent: "space-between",
          alignItems: "center",
          flexShrink: 0,
        }}
      >
        <span
          style={{
            fontFamily: fontFamilyHeading,
            fontSize: 16,
            fontWeight: 700,
            color: "#ffffff",
          }}
        >
          DNS Management
        </span>
        <span
          style={{
            fontFamily: fontFamilyMono,
            fontSize: 12,
            color: "rgba(255,255,255,0.7)",
          }}
        >
          mycoolproject.com
        </span>
      </div>

      {/* ---- Sub-header ---- */}
      <div
        style={{
          backgroundColor: "#f7f7f8",
          borderBottom: "1px solid #e5e5e5",
          padding: "0.75rem 2rem",
          flexShrink: 0,
        }}
      >
        <span
          style={{
            fontFamily: fontFamilyHeading,
            fontSize: 13,
            fontWeight: 700,
            color: "#333333",
          }}
        >
          DNS Records
        </span>
      </div>

      {/* ---- Table header ---- */}
      <div
        style={{
          display: "grid",
          gridTemplateColumns: gridColumns,
          padding: "0.5rem 2rem",
          backgroundColor: "#f7f7f8",
          borderBottom: "1px solid #e5e5e5",
          flexShrink: 0,
        }}
      >
        {["TYPE", "NAME", "CONTENT", ""].map((col, i) => (
          <span
            key={i}
            style={{
              fontFamily: fontFamilyMono,
              fontSize: 11,
              fontWeight: 700,
              textTransform: "uppercase",
              letterSpacing: "0.1em",
              color: "#666666",
            }}
          >
            {col}
          </span>
        ))}
      </div>

      {/* ---- Record rows ---- */}
      <div style={{ flex: 1, overflow: "hidden" }}>
        {records.map((record, index) => (
          <div
            key={index}
            style={{
              display: "grid",
              gridTemplateColumns: gridColumns,
              padding: "1rem 2rem",
              borderBottom: "1px solid #e5e5e5",
              alignItems: "center",
              backgroundColor: record.isNew ? "#fffde7" : "transparent",
            }}
          >
            {/* Type badge */}
            <div>
              <span
                style={{
                  fontFamily: fontFamilyMono,
                  fontSize: 12,
                  fontWeight: 700,
                  backgroundColor: "#e8f0fe",
                  color: "#1a73e8",
                  padding: "2px 8px",
                  borderRadius: 4,
                  display: "inline-block",
                }}
              >
                {record.type}
              </span>
            </div>

            {/* Name */}
            <span
              style={{
                fontFamily: fontFamilyMono,
                fontSize: 13,
                color: "#0a0a0a",
              }}
            >
              {record.name}
            </span>

            {/* Content */}
            <span
              style={{
                fontFamily: fontFamilyMono,
                fontSize: 13,
                color: "#555555",
                wordBreak: "break-all",
              }}
            >
              {record.content}
            </span>

            {/* Actions placeholder */}
            <div />
          </div>
        ))}
      </div>

      {/* ---- Bottom bar ---- */}
      <div
        style={{
          padding: "1rem 2rem",
          display: "flex",
          justifyContent: "flex-end",
          borderTop: "1px solid #e5e5e5",
          flexShrink: 0,
        }}
      >
        <div
          style={{
            backgroundColor: "#1a73e8",
            color: "#ffffff",
            padding: "8px 24px",
            borderRadius: 6,
            fontFamily: fontFamilyMono,
            fontSize: 12,
            fontWeight: 700,
            cursor: "pointer",
          }}
        >
          Save
        </div>
      </div>
    </div>
  );
};
