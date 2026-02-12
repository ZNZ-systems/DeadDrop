import React from "react";
import { AbsoluteFill } from "remotion";

import { COLORS } from "../constants/theme";
import { fontFamilyHeading, fontFamilyMono } from "../fonts/load";
import { FadeIn } from "../components/animation";

/**
 * OutroCard -- Closing card with CTA for the DeadDrop product demo.
 *
 * Duration: 60 frames (2.0s @ 30fps)
 *
 * Timeline:
 *   Frame  0-12  Diamond logo and tagline fade in
 *   Frame 15-25  URL fades in
 *   Frame 25-35  Footer text fades in
 *   Frame 35-60  Hold
 */
export const OutroCard: React.FC = () => {
  return (
    <AbsoluteFill
      style={{
        backgroundColor: COLORS.black,
        display: "flex",
        flexDirection: "column",
        alignItems: "center",
        justifyContent: "center",
      }}
    >
      {/* Diamond logo (smaller) */}
      <FadeIn
        startFrame={0}
        durationFrames={12}
        direction="none"
        style={{ marginBottom: 28 }}
      >
        <svg
          width={32}
          height={32}
          viewBox="0 0 100 100"
          xmlns="http://www.w3.org/2000/svg"
        >
          <polygon
            points="50,10 90,50 50,90 10,50"
            fill={COLORS.red}
          />
        </svg>
      </FadeIn>

      {/* Tagline */}
      <FadeIn startFrame={0} durationFrames={12} direction="up">
        <div
          style={{
            fontFamily: fontFamilyHeading,
            fontSize: 36,
            fontWeight: 700,
            color: COLORS.white,
            letterSpacing: "-0.02em",
            textAlign: "center",
            lineHeight: 1.3,
          }}
        >
          Your website. Their messages. Your inbox.
        </div>
      </FadeIn>

      {/* URL */}
      <FadeIn
        startFrame={15}
        durationFrames={10}
        direction="up"
        style={{ marginTop: 20 }}
      >
        <div
          style={{
            fontFamily: fontFamilyMono,
            fontSize: 20,
            fontWeight: 700,
            color: COLORS.red,
            letterSpacing: "0.05em",
            textTransform: "uppercase",
          }}
        >
          deaddrop.io
        </div>
      </FadeIn>

      {/* Footer text */}
      <FadeIn
        startFrame={25}
        durationFrames={10}
        direction="none"
        style={{ marginTop: 32 }}
      >
        <div
          style={{
            fontFamily: fontFamilyMono,
            fontSize: 11,
            color: "rgba(255,255,255,0.4)",
            letterSpacing: "0.15em",
            textTransform: "uppercase",
          }}
        >
          Open Source · Self-Hosted · Privacy-First
        </div>
      </FadeIn>
    </AbsoluteFill>
  );
};
