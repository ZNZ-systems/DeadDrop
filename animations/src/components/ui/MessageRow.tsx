import React from "react";
import { fontFamilyMono } from "../../fonts/load";
import { COLORS, BORDERS } from "../../constants/theme";

interface MessageRowProps {
  senderName?: string;
  senderEmail?: string;
  body: string;
  time: string;
  isRead: boolean;
  showMarkRead?: boolean;
  style?: React.CSSProperties;
}

export const MessageRow: React.FC<MessageRowProps> = ({
  senderName,
  senderEmail,
  body,
  time,
  isRead,
  showMarkRead = true,
  style,
}) => {
  const isUnread = !isRead;

  const itemStyle: React.CSSProperties = {
    padding: "1.25rem 1.5rem",
    borderBottom: BORDERS.thick,
    boxSizing: "border-box",
    fontFamily: fontFamilyMono,
    ...(isUnread ? { borderLeft: `4px solid ${COLORS.red}` } : {}),
  };

  const metaStyle: React.CSSProperties = {
    display: "flex",
    alignItems: "center",
    gap: "0.75rem",
    marginBottom: "0.5rem",
    flexWrap: "wrap",
  };

  const senderNameStyle: React.CSSProperties = {
    fontWeight: 700,
    fontSize: 13,
    color: COLORS.black,
  };

  const senderEmailStyle: React.CSSProperties = {
    fontSize: 11,
    color: COLORS.gray,
  };

  const unreadDotStyle: React.CSSProperties = {
    display: "inline-block",
    width: 8,
    height: 8,
    background: COLORS.red,
  };

  const bodyStyle: React.CSSProperties = {
    fontSize: 12,
    color: COLORS.gray,
    lineHeight: 1.8,
    whiteSpace: "pre-wrap",
    marginBottom: "0.5rem",
  };

  const timeStyle: React.CSSProperties = {
    fontSize: 10,
    color: COLORS.gray,
    letterSpacing: "0.1em",
    textTransform: "uppercase",
  };

  const actionsStyle: React.CSSProperties = {
    display: "flex",
    gap: "1rem",
    marginTop: "0.75rem",
  };

  const actionBaseStyle: React.CSSProperties = {
    fontFamily: fontFamilyMono,
    fontSize: 10,
    fontWeight: 700,
    letterSpacing: "0.1em",
    textTransform: "uppercase",
    background: "none",
    border: "none",
    cursor: "pointer",
    padding: 0,
  };

  const actionReadStyle: React.CSSProperties = {
    ...actionBaseStyle,
    color: COLORS.black,
  };

  const actionDeleteStyle: React.CSSProperties = {
    ...actionBaseStyle,
    color: COLORS.red,
  };

  const shouldShowMarkRead = isUnread && showMarkRead;

  return (
    <div style={{ ...itemStyle, ...style }}>
      <div style={metaStyle}>
        {senderName && <span style={senderNameStyle}>{senderName}</span>}
        {senderEmail && <span style={senderEmailStyle}>{senderEmail}</span>}
        {isUnread && <span style={unreadDotStyle} />}
      </div>
      <div style={bodyStyle}>{body}</div>
      <span style={timeStyle}>{time}</span>
      <div style={actionsStyle}>
        {shouldShowMarkRead && (
          <span style={actionReadStyle}>Mark Read</span>
        )}
        <span style={actionDeleteStyle}>Delete</span>
      </div>
    </div>
  );
};
