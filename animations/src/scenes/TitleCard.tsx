import React from "react";
import { AbsoluteFill } from "remotion";

import { COLORS, SPRING_PRESETS } from "../constants/theme";
import { fontFamilyHeading, fontFamilyMono } from "../fonts/load";
import { FadeIn, ScaleIn } from "../components/animation";

/**
 * TitleCard -- Dramatic title reveal for the DeadDrop product demo.
 *
 * Duration: 60 frames (2.0s @ 30fps)
 *
 * Timeline:
 *   Frame  0-10  Diamond logo scales in with bounce
 *   Frame  8-18  Logo text fades up
 *   Frame 18-28  Subtitle fades up
 *   Frame 25-35  Red decorative line appears
 *   Frame 35-60  Hold on complete state
 */
export const TitleCard: React.FC = () => {
  return (
    <AbsoluteFill
      style={{
        backgroundColor: COLORS.bg,
        display: "flex",
        flexDirection: "column",
        alignItems: "center",
        justifyContent: "center",
      }}
    >
      {/* Diamond logo */}
      <ScaleIn
        startFrame={0}
        config={SPRING_PRESETS.bouncy}
        style={{
          display: "flex",
          alignItems: "center",
          justifyContent: "center",
          marginBottom: 24,
        }}
      >
        <svg
          width={48}
          height={48}
          viewBox="0 0 100 100"
          xmlns="http://www.w3.org/2000/svg"
        >
          <polygon
            points="50,10 90,50 50,90 10,50"
            fill={COLORS.red}
          />
        </svg>
      </ScaleIn>

      {/* Logo text: DEAD + DROP */}
      <FadeIn startFrame={8} durationFrames={10} direction="up">
        <div
          style={{
            fontFamily: fontFamilyHeading,
            fontSize: 64,
            fontWeight: 700,
            textTransform: "uppercase",
            letterSpacing: "-0.03em",
            lineHeight: 1,
          }}
        >
          <span style={{ color: COLORS.black }}>DEAD</span>
          <span style={{ color: COLORS.red }}>DROP</span>
        </div>
      </FadeIn>

      {/* Subtitle */}
      <FadeIn
        startFrame={18}
        durationFrames={10}
        direction="up"
        style={{ marginTop: 16 }}
      >
        <div
          style={{
            fontFamily: fontFamilyMono,
            fontSize: 16,
            fontWeight: 400,
            color: COLORS.gray,
            letterSpacing: "0.08em",
            textTransform: "uppercase",
          }}
        >
          Anonymous Contact Widget for Any Website
        </div>
      </FadeIn>

      {/* Decorative red line */}
      <FadeIn
        startFrame={25}
        durationFrames={10}
        direction="none"
        style={{ marginTop: 20 }}
      >
        <div
          style={{
            width: 80,
            height: 2,
            backgroundColor: COLORS.red,
          }}
        />
      </FadeIn>
    </AbsoluteFill>
  );
};
