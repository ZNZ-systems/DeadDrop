import React from "react";
import { BORDERS } from "../../constants/theme";

interface ListCardProps {
  children: React.ReactNode;
  style?: React.CSSProperties;
}

export const ListCard: React.FC<ListCardProps> = ({ children, style }) => {
  const cardStyle: React.CSSProperties = {
    border: BORDERS.thick,
    boxSizing: "border-box",
  };

  return (
    <div style={{ ...cardStyle, ...style }}>
      {children}
    </div>
  );
};
