import React from "react";
import { COLORS, FONT } from "../styles";

interface TerminalProps {
  children: React.ReactNode;
}

const Dot: React.FC<{ color: string }> = ({ color }) => (
  <div
    style={{
      width: 12,
      height: 12,
      borderRadius: "50%",
      backgroundColor: color,
    }}
  />
);

export const Terminal: React.FC<TerminalProps> = ({ children }) => {
  return (
    <div
      style={{
        width: "100%",
        height: "100%",
        display: "flex",
        alignItems: "center",
        justifyContent: "center",
        backgroundColor: COLORS.outerBg,
        padding: 20,
      }}
    >
      <div
        style={{
          width: 760,
          height: 460,
          borderRadius: 10,
          overflow: "hidden",
          boxShadow: "0 8px 40px rgba(0,0,0,0.5)",
          display: "flex",
          flexDirection: "column",
        }}
      >
        {/* Title bar */}
        <div
          style={{
            backgroundColor: COLORS.titleBar,
            padding: "8px 14px",
            display: "flex",
            alignItems: "center",
            gap: 8,
            flexShrink: 0,
          }}
        >
          <Dot color={COLORS.dot.red} />
          <Dot color={COLORS.dot.yellow} />
          <Dot color={COLORS.dot.green} />
          <span
            style={{
              flex: 1,
              textAlign: "center",
              color: COLORS.titleBarText,
              fontSize: 13,
              fontFamily: FONT.family,
              opacity: 0.7,
            }}
          >
            Terminal
          </span>
          <div style={{ width: 52 }} />
        </div>
        {/* Terminal body */}
        <div
          style={{
            backgroundColor: COLORS.bg,
            padding: "14px 18px",
            flex: 1,
            overflow: "hidden",
            fontFamily: FONT.family,
            fontSize: FONT.size,
          }}
        >
          {children}
        </div>
      </div>
    </div>
  );
};
