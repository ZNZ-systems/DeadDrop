import React from "react";
import {
  AbsoluteFill,
  useCurrentFrame,
  useVideoConfig,
  interpolate,
  spring,
} from "remotion";

import { MockBrowser } from "../components/layout";
import { AppShell } from "../components/layout";

import {
  Button,
  Input,
  Badge,
  FlashMessage,
  InfoPanel,
  CodeBlock,
  CodeHL,
  CodeVal,
} from "../components/ui";

import { Cursor, FadeIn } from "../components/animation";

import { COLORS } from "../constants/theme";
import { fontFamilyHeading, fontFamilyMono } from "../fonts/load";

/**
 * Scene 1 -- Register a Domain
 *
 * Duration: 135 frames (4.5s @ 30fps)
 *
 * Timeline:
 *   Phase 1  (0-14)   Form appears with FadeIn
 *   Phase 2  (15-56)  Cursor arrives, user types "mycoolproject.com"
 *   Phase 3  (57-74)  Cursor moves to button, hovers, clicks
 *   Phase 4  (75-89)  Form fades out, URL changes, detail page fades in
 *   Phase 5  (90-119) Domain detail with flash + verification panel
 *   Phase 6  (120-134) Static hold
 */
export const Scene1RegisterDomain: React.FC = () => {
  const frame = useCurrentFrame();
  const { fps } = useVideoConfig();

  // ---------- Timing constants ----------
  const TYPING_START = 17;
  const FRAMES_PER_CHAR = 2;
  const DOMAIN = "mycoolproject.com";

  // Phase boundaries
  const FORM_FADE_START = 0;
  const CURSOR_APPEAR = 15;
  const CURSOR_MOVE_TO_BTN = 57;
  const BTN_HOVER_START = 62;
  const BTN_CLICK_START = 68;
  const BTN_CLICK_END = 72;
  const PAGE_TRANSITION_START = 75;
  const PAGE_TRANSITION_END = 82;
  const DETAIL_FADE_START = 90;

  // ---------- Derived values ----------

  // Typed domain text
  const charsTyped = Math.min(
    DOMAIN.length,
    Math.max(0, Math.floor((frame - TYPING_START) / FRAMES_PER_CHAR)),
  );
  const currentInputValue = DOMAIN.slice(0, charsTyped);
  const inputFocused = frame >= CURSOR_APPEAR && frame < PAGE_TRANSITION_START;

  // Button state
  const buttonHovered =
    frame >= BTN_HOVER_START && frame < PAGE_TRANSITION_START;
  const cursorClicking =
    frame >= BTN_CLICK_START && frame <= BTN_CLICK_END;

  // Form page opacity (fades out during transition)
  const formOpacity = interpolate(
    frame,
    [PAGE_TRANSITION_START, PAGE_TRANSITION_END],
    [1, 0],
    { extrapolateLeft: "clamp", extrapolateRight: "clamp" },
  );

  // Detail page opacity (fades in)
  const detailOpacity = interpolate(
    frame,
    [DETAIL_FADE_START - 8, DETAIL_FADE_START],
    [0, 1],
    { extrapolateLeft: "clamp", extrapolateRight: "clamp" },
  );

  // URL changes at the transition midpoint
  const browserUrl =
    frame < PAGE_TRANSITION_END
      ? "deaddrop.io/domains/new"
      : "deaddrop.io/domains/mycoolproject.com";

  // ---------- Cursor position (interpolated across phases) ----------

  // Waypoints: input field -> button -> off-screen
  const cursorVisible = frame >= CURSOR_APPEAR && frame < PAGE_TRANSITION_START;

  const cursorX = interpolate(
    frame,
    [CURSOR_APPEAR, CURSOR_MOVE_TO_BTN, CURSOR_MOVE_TO_BTN + 6],
    [420, 420, 450],
    { extrapolateLeft: "clamp", extrapolateRight: "clamp" },
  );

  const cursorY = interpolate(
    frame,
    [CURSOR_APPEAR, CURSOR_MOVE_TO_BTN, CURSOR_MOVE_TO_BTN + 6],
    [380, 380, 478],
    { extrapolateLeft: "clamp", extrapolateRight: "clamp" },
  );

  // ---------- Detail page spring for entrance ----------
  const detailSlide = spring({
    frame,
    fps,
    config: { damping: 200, mass: 1, stiffness: 100, overshootClamping: false },
    delay: DETAIL_FADE_START,
  });

  const detailTranslateY = interpolate(detailSlide, [0, 1], [12, 0]);

  return (
    <AbsoluteFill>
      <MockBrowser url={browserUrl}>
        <AppShell email="demo@deaddrop.io">
          {/* ============ FORM PAGE ============ */}
          <div
            style={{
              opacity: formOpacity,
              position: "absolute",
              top: 0,
              left: 0,
              right: 0,
              bottom: 0,
              paddingTop: 80,
              paddingLeft: "2rem",
              paddingRight: "2rem",
              maxWidth: 900,
              marginLeft: "auto",
              marginRight: "auto",
              boxSizing: "border-box",
              pointerEvents: frame >= PAGE_TRANSITION_END ? "none" : "auto",
            }}
          >
            <FadeIn startFrame={FORM_FADE_START} durationFrames={14}>
              {/* Red tag */}
              <div
                style={{
                  fontFamily: fontFamilyMono,
                  fontSize: 11,
                  fontWeight: 700,
                  textTransform: "uppercase",
                  letterSpacing: "0.15em",
                  color: COLORS.red,
                  border: `2px solid ${COLORS.red}`,
                  padding: "6px 16px",
                  display: "inline-block",
                  marginBottom: "1.5rem",
                }}
              >
                NEW DOMAIN
              </div>

              {/* Page title */}
              <div
                style={{
                  fontFamily: fontFamilyHeading,
                  fontSize: "2.2rem",
                  fontWeight: 700,
                  letterSpacing: "-0.03em",
                  marginBottom: "2.5rem",
                  color: COLORS.black,
                }}
              >
                Add Domain
              </div>

              {/* Form */}
              <div style={{ maxWidth: 540 }}>
                <Input
                  label="DOMAIN NAME"
                  placeholder="example.com"
                  hint="Enter the domain name without http:// or www"
                  value={currentInputValue}
                  focused={inputFocused}
                  style={{ marginBottom: "1.5rem" }}
                />

                <Button
                  label="ADD DOMAIN"
                  variant="primary"
                  hovered={buttonHovered}
                  style={{ width: "100%", textAlign: "center" }}
                />
              </div>
            </FadeIn>
          </div>

          {/* ============ DETAIL PAGE ============ */}
          <div
            style={{
              opacity: detailOpacity,
              transform: `translateY(${detailTranslateY}px)`,
              position: "absolute",
              top: 0,
              left: 0,
              right: 0,
              bottom: 0,
              paddingTop: 80,
              paddingLeft: "2rem",
              paddingRight: "2rem",
              maxWidth: 900,
              marginLeft: "auto",
              marginRight: "auto",
              boxSizing: "border-box",
              pointerEvents:
                frame >= DETAIL_FADE_START ? "auto" : "none",
            }}
          >
            {/* Flash message */}
            <FlashMessage
              message="Domain added successfully"
              type="success"
              style={{ marginBottom: "1.5rem" }}
            />

            {/* Page header: title + badge */}
            <div
              style={{
                display: "flex",
                alignItems: "center",
                gap: "1rem",
                marginBottom: "2rem",
              }}
            >
              <span
                style={{
                  fontFamily: fontFamilyHeading,
                  fontSize: "2.2rem",
                  fontWeight: 700,
                  letterSpacing: "-0.03em",
                  color: COLORS.black,
                }}
              >
                mycoolproject.com
              </span>
              <Badge variant="unverified" text="Unverified" />
            </div>

            {/* Verification panel */}
            <InfoPanel title="DNS Verification Required" variant="warn">
              <div
                style={{
                  fontFamily: fontFamilyMono,
                  fontSize: 12,
                  color: COLORS.gray,
                  lineHeight: 1.8,
                  marginBottom: "1rem",
                }}
              >
                Add this TXT record to your domain's DNS settings:
              </div>
              <CodeBlock>
                <CodeHL>deaddrop-verify</CodeHL>=
                <CodeVal>a9c55678-1234-5678-abcd-ef0123456789</CodeVal>
              </CodeBlock>
              <div style={{ marginTop: "1.25rem" }}>
                <Button
                  label="CHECK VERIFICATION"
                  variant="outline"
                  size="sm"
                />
              </div>
            </InfoPanel>
          </div>
        </AppShell>

        {/* ============ CURSOR OVERLAY ============ */}
        <Cursor
          x={cursorX}
          y={cursorY}
          clicking={cursorClicking}
          visible={cursorVisible}
        />
      </MockBrowser>
    </AbsoluteFill>
  );
};
