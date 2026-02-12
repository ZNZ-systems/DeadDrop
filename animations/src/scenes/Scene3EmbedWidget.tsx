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
  Badge,
  InfoPanel,
  CodeBlock,
  CodeHL,
  CodeVal,
} from "../components/ui";

import { Cursor, ScaleIn, SlideIn } from "../components/animation";

import { WidgetButton, WidgetPanel, WidgetSuccess } from "../components/widget";

import { COLORS, SPRING_PRESETS } from "../constants/theme";
import { fontFamilyHeading, fontFamilyMono } from "../fonts/load";

/**
 * Scene 3 -- Embed Widget
 *
 * Duration: 150 frames (5.0s @ 30fps)
 *
 * Timeline:
 *   Phase 1  (0-29)    Embed code panel on domain detail page
 *   Phase 2  (30-59)   Crossfade to user's mock website
 *   Phase 3  (60-74)   WidgetButton appears on the mock site
 *   Phase 4  (75-124)  Widget interaction -- fill form fields
 *   Phase 5  (125-149) Success state
 */
export const Scene3EmbedWidget: React.FC = () => {
  const frame = useCurrentFrame();
  const { fps } = useVideoConfig();

  // ========================================================================
  // PHASE 1 -- Embed code panel (frames 0-29)
  // ========================================================================
  const CURSOR_APPEAR = 8;
  const CURSOR_CLICK = 16;
  const CURSOR_CLICK_END = 20;
  const COPIED_APPEAR = 22;

  // ========================================================================
  // PHASE 2 -- Crossfade to user website (frames 30-59)
  // ========================================================================
  const CROSSFADE_START = 30;
  const CROSSFADE_END = 38;
  const SITE_FADE_IN_START = 34;
  const SITE_FADE_IN_END = 44;
  const SCRIPT_OVERLAY_APPEAR = 48;
  const SCRIPT_OVERLAY_GONE = 58;

  // ========================================================================
  // PHASE 3 -- Widget appears (frames 60-74)
  // ========================================================================
  const WIDGET_BTN_APPEAR = 60;

  // ========================================================================
  // PHASE 4 -- Widget interaction (frames 75-124)
  // ========================================================================
  const WIDGET_CLICK = 75;
  const PANEL_OPEN = 78;
  const NAME_TYPE_START = 82;
  const EMAIL_TYPE_START = 92;
  const MSG_TYPE_START = 104;
  const SEND_CURSOR_MOVE = 122;
  const SEND_CLICK = 124;

  // ========================================================================
  // PHASE 5 -- Success (frames 125-149)
  // ========================================================================
  const SUCCESS_TRANSITION_START = 125;
  const SUCCESS_TRANSITION_END = 130;

  // ---------- Derived: Phase detection ----------
  const isDeadDropPage = frame < CROSSFADE_END;
  const isMockSite = frame >= SITE_FADE_IN_START;

  // ---------- Derived: DeadDrop page opacity ----------
  const deadDropOpacity = interpolate(
    frame,
    [CROSSFADE_START, CROSSFADE_END],
    [1, 0],
    { extrapolateLeft: "clamp", extrapolateRight: "clamp" },
  );

  // ---------- Derived: Mock website opacity ----------
  const mockSiteOpacity = interpolate(
    frame,
    [SITE_FADE_IN_START, SITE_FADE_IN_END],
    [0, 1],
    { extrapolateLeft: "clamp", extrapolateRight: "clamp" },
  );

  // ---------- Derived: Script overlay opacity ----------
  const scriptOverlayOpacity = interpolate(
    frame,
    [SCRIPT_OVERLAY_APPEAR, SCRIPT_OVERLAY_APPEAR + 4, SCRIPT_OVERLAY_GONE - 4, SCRIPT_OVERLAY_GONE],
    [0, 1, 1, 0],
    { extrapolateLeft: "clamp", extrapolateRight: "clamp" },
  );

  // ---------- Derived: Browser URL ----------
  const browserUrl = frame < CROSSFADE_END
    ? "deaddrop.io/domains/mycoolproject.com"
    : "mycoolproject.com";

  // ---------- Derived: Phase 1 cursor ----------
  const phase1CursorVisible = frame >= CURSOR_APPEAR && frame < CROSSFADE_START;
  const phase1CursorX = interpolate(
    frame,
    [CURSOR_APPEAR, CURSOR_CLICK],
    [600, 480],
    { extrapolateLeft: "clamp", extrapolateRight: "clamp" },
  );
  const phase1CursorY = interpolate(
    frame,
    [CURSOR_APPEAR, CURSOR_CLICK],
    [300, 440],
    { extrapolateLeft: "clamp", extrapolateRight: "clamp" },
  );
  const phase1CursorClicking = frame >= CURSOR_CLICK && frame < CURSOR_CLICK_END;

  // ---------- Derived: "COPIED!" text ----------
  const copiedVisible = frame >= COPIED_APPEAR && frame < CROSSFADE_START;

  // ---------- Derived: WidgetButton scale-in ----------
  const widgetBtnProgress = spring({
    frame,
    fps,
    config: SPRING_PRESETS.snappy,
    delay: WIDGET_BTN_APPEAR,
  });
  const widgetBtnScale = interpolate(widgetBtnProgress, [0, 1], [0, 1]);
  const widgetBtnVisible = frame >= WIDGET_BTN_APPEAR;

  // ---------- Derived: Widget panel open state ----------
  const panelOpenProgress = spring({
    frame,
    fps,
    config: SPRING_PRESETS.smooth,
    delay: PANEL_OPEN,
  });
  const isPanelOpen = frame >= PANEL_OPEN && frame < SUCCESS_TRANSITION_START;

  // ---------- Derived: Typed field values ----------
  const nameText = "Jane Smith";
  const nameChars = Math.min(
    nameText.length,
    Math.max(0, Math.floor((frame - NAME_TYPE_START) / 2)),
  );
  const nameValue = frame >= NAME_TYPE_START ? nameText.slice(0, nameChars) : "";

  const emailText = "jane@example.com";
  const emailChars = Math.min(
    emailText.length,
    Math.max(0, Math.floor((frame - EMAIL_TYPE_START) / 2)),
  );
  const emailValue = frame >= EMAIL_TYPE_START ? emailText.slice(0, emailChars) : "";

  const msgText = "Love the project! How can I contribute?";
  const msgChars = Math.min(
    msgText.length,
    Math.max(0, Math.floor((frame - MSG_TYPE_START) / 2)),
  );
  const messageValue = frame >= MSG_TYPE_START ? msgText.slice(0, msgChars) : "";

  // ---------- Derived: Phase 3-4 cursor ----------
  const phase34CursorVisible = frame >= WIDGET_BTN_APPEAR && frame <= SEND_CLICK;

  const phase34CursorX = interpolate(
    frame,
    [
      WIDGET_BTN_APPEAR,     // start: off to the left
      WIDGET_CLICK - 2,      // approach widget button
      WIDGET_CLICK,          // at widget button
      PANEL_OPEN + 2,        // stay near button briefly
      SEND_CURSOR_MOVE,      // move toward send button
      SEND_CLICK,            // at send button
    ],
    [
      700,     // start position
      830,     // near widget button (bottom-right area)
      830,     // click on widget button
      830,     // hold
      790,     // moving to send button
      790,     // at send button
    ],
    { extrapolateLeft: "clamp", extrapolateRight: "clamp" },
  );

  const phase34CursorY = interpolate(
    frame,
    [
      WIDGET_BTN_APPEAR,
      WIDGET_CLICK - 2,
      WIDGET_CLICK,
      PANEL_OPEN + 2,
      SEND_CURSOR_MOVE,
      SEND_CLICK,
    ],
    [
      500,     // start
      720,     // near widget button
      720,     // click
      720,     // hold
      620,     // near send button area
      620,     // at send button
    ],
    { extrapolateLeft: "clamp", extrapolateRight: "clamp" },
  );

  const phase34CursorClicking =
    (frame >= WIDGET_CLICK && frame < WIDGET_CLICK + 4) ||
    (frame >= SEND_CLICK && frame <= SEND_CLICK + 3);

  // ---------- Derived: Success transition ----------
  const panelFadeOut = interpolate(
    frame,
    [SUCCESS_TRANSITION_START, SUCCESS_TRANSITION_END],
    [1, 0],
    { extrapolateLeft: "clamp", extrapolateRight: "clamp" },
  );
  const successFadeIn = interpolate(
    frame,
    [SUCCESS_TRANSITION_START, SUCCESS_TRANSITION_END],
    [0, 1],
    { extrapolateLeft: "clamp", extrapolateRight: "clamp" },
  );
  const showSuccess = frame >= SUCCESS_TRANSITION_START;

  // ---------- Pick active cursor ----------
  const cursorVisible = phase1CursorVisible || phase34CursorVisible;
  const cursorX = phase1CursorVisible ? phase1CursorX : phase34CursorX;
  const cursorY = phase1CursorVisible ? phase1CursorY : phase34CursorY;
  const cursorClicking = phase1CursorVisible
    ? phase1CursorClicking
    : phase34CursorClicking;

  // ========================================================================
  // RENDER
  // ========================================================================
  return (
    <AbsoluteFill>
      <MockBrowser url={browserUrl}>
        {/* ============ DEADDROP DOMAIN DETAIL PAGE ============ */}
        {isDeadDropPage && (
          <div
            style={{
              opacity: deadDropOpacity,
              position: "absolute",
              top: 0,
              left: 0,
              right: 0,
              bottom: 0,
              pointerEvents: frame >= CROSSFADE_END ? "none" : "auto",
            }}
          >
            <AppShell email="demo@deaddrop.io">
              {/* Page header */}
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
                <Badge variant="verified" text="Verified" />
              </div>

              {/* Embed code panel */}
              <InfoPanel title="Widget Embed Code" variant="success">
                <div
                  style={{
                    fontFamily: fontFamilyMono,
                    fontSize: 12,
                    color: COLORS.gray,
                    lineHeight: 1.8,
                    marginBottom: "1rem",
                  }}
                >
                  Add this script tag to any page on mycoolproject.com:
                </div>
                <div style={{ position: "relative" }}>
                  <CodeBlock>
                    <CodeHL>&lt;script</CodeHL> src=
                    <CodeVal>"deaddrop.io/static/widget.js"</CodeVal>{" "}
                    data-deaddrop-id=
                    <CodeVal>"abc-123"</CodeVal>
                    <CodeHL>&gt;&lt;/script&gt;</CodeHL>
                  </CodeBlock>

                  {/* "COPIED!" label */}
                  {copiedVisible && (
                    <ScaleIn startFrame={COPIED_APPEAR}>
                      <div
                        style={{
                          position: "absolute",
                          top: -28,
                          left: "50%",
                          transform: "translateX(-50%)",
                          fontFamily: fontFamilyMono,
                          fontSize: 10,
                          fontWeight: 700,
                          textTransform: "uppercase",
                          color: COLORS.green,
                          letterSpacing: "0.15em",
                          whiteSpace: "nowrap",
                        }}
                      >
                        COPIED!
                      </div>
                    </ScaleIn>
                  )}
                </div>
              </InfoPanel>
            </AppShell>
          </div>
        )}

        {/* ============ MOCK WEBSITE ============ */}
        {isMockSite && (
          <div
            style={{
              opacity: mockSiteOpacity,
              position: "absolute",
              top: 0,
              left: 0,
              right: 0,
              bottom: 0,
              backgroundColor: "#ffffff",
              overflow: "hidden",
            }}
          >
            {/* ---- Site header ---- */}
            <div
              style={{
                display: "flex",
                alignItems: "center",
                justifyContent: "space-between",
                padding: "1.5rem 3rem",
                borderBottom: "1px solid #e5e5e5",
                backgroundColor: "#ffffff",
              }}
            >
              <span
                style={{
                  fontFamily: fontFamilyHeading,
                  fontSize: 20,
                  fontWeight: 700,
                  color: "#0a0a0a",
                  letterSpacing: "-0.02em",
                }}
              >
                mycoolproject
              </span>
              <div
                style={{
                  display: "flex",
                  gap: "2rem",
                  alignItems: "center",
                }}
              >
                {["About", "Features", "Blog"].map((link) => (
                  <span
                    key={link}
                    style={{
                      fontFamily: fontFamilyHeading,
                      fontSize: 13,
                      color: "#555555",
                      cursor: "pointer",
                    }}
                  >
                    {link}
                  </span>
                ))}
              </div>
            </div>

            {/* ---- Hero section ---- */}
            <div
              style={{
                display: "flex",
                flexDirection: "column",
                alignItems: "center",
                justifyContent: "center",
                padding: "4rem 2rem 3rem",
                textAlign: "center",
              }}
            >
              <h1
                style={{
                  fontFamily: fontFamilyHeading,
                  fontSize: 42,
                  fontWeight: 700,
                  color: "#0a0a0a",
                  letterSpacing: "-0.03em",
                  margin: "0 0 1rem 0",
                }}
              >
                Welcome to My Cool Project
              </h1>
              <p
                style={{
                  fontSize: 16,
                  color: "#666666",
                  margin: "0 0 2rem 0",
                  fontFamily: fontFamilyHeading,
                  lineHeight: 1.6,
                }}
              >
                Building amazing things, one line at a time.
              </p>
              <div
                style={{
                  backgroundColor: "#0a0a0a",
                  color: "#ffffff",
                  padding: "14px 32px",
                  borderRadius: 0,
                  fontSize: 14,
                  fontWeight: 600,
                  fontFamily: fontFamilyHeading,
                  cursor: "pointer",
                }}
              >
                Get Started
              </div>
            </div>

            {/* ---- Content cards ---- */}
            <div
              style={{
                display: "flex",
                gap: "1.5rem",
                padding: "0 3rem 3rem",
                justifyContent: "center",
              }}
            >
              {[
                { color: "#ff2200", title: "Lightning Fast", desc: "Optimized for speed at every layer of the stack." },
                { color: "#22c55e", title: "Open Source", desc: "Fully transparent codebase with an active community." },
                { color: "#3b82f6", title: "Secure by Default", desc: "Enterprise-grade security built into the core." },
              ].map((card) => (
                <div
                  key={card.title}
                  style={{
                    flex: 1,
                    maxWidth: 280,
                    border: "1px solid #e5e5e5",
                    borderRadius: 0,
                    overflow: "hidden",
                  }}
                >
                  <div
                    style={{
                      height: 4,
                      backgroundColor: card.color,
                    }}
                  />
                  <div style={{ padding: "1.25rem" }}>
                    <div
                      style={{
                        fontFamily: fontFamilyHeading,
                        fontSize: 16,
                        fontWeight: 700,
                        color: "#0a0a0a",
                        marginBottom: "0.5rem",
                      }}
                    >
                      {card.title}
                    </div>
                    <div
                      style={{
                        fontSize: 13,
                        color: "#888888",
                        lineHeight: 1.6,
                        fontFamily: fontFamilyHeading,
                      }}
                    >
                      {card.desc}
                    </div>
                  </div>
                </div>
              ))}
            </div>

            {/* ---- Script tag overlay ---- */}
            {frame >= SCRIPT_OVERLAY_APPEAR && frame <= SCRIPT_OVERLAY_GONE && (
              <div
                style={{
                  position: "absolute",
                  bottom: 80,
                  left: "50%",
                  transform: "translateX(-50%)",
                  opacity: scriptOverlayOpacity,
                  zIndex: 10,
                }}
              >
                <SlideIn startFrame={SCRIPT_OVERLAY_APPEAR} direction="down">
                  <div
                    style={{
                      backgroundColor: "#0a0a0a",
                      color: "#aaaaaa",
                      fontFamily: fontFamilyMono,
                      fontSize: 11,
                      padding: "12px 20px",
                      borderRadius: 6,
                      boxShadow: "0 8px 24px rgba(0,0,0,0.3)",
                      whiteSpace: "nowrap",
                    }}
                  >
                    <span style={{ color: "#ff2200" }}>&lt;script</span>{" "}
                    <span style={{ color: "#aaaaaa" }}>src=</span>
                    <span style={{ color: "#ffd700" }}>"..."</span>{" "}
                    <span style={{ color: "#aaaaaa" }}>data-deaddrop-id=</span>
                    <span style={{ color: "#ffd700" }}>"abc-123"</span>
                    <span style={{ color: "#ff2200" }}>&gt;&lt;/script&gt;</span>
                  </div>
                </SlideIn>
              </div>
            )}

            {/* ---- Widget button ---- */}
            {widgetBtnVisible && (
              <div
                style={{
                  transform: `scale(${widgetBtnScale})`,
                  transformOrigin: "bottom right",
                }}
              >
                <WidgetButton />
              </div>
            )}

            {/* ---- Widget panel (form) ---- */}
            {isPanelOpen && (
              <div style={{ opacity: frame >= SUCCESS_TRANSITION_START ? panelFadeOut : 1 }}>
                <WidgetPanel
                  isOpen={panelOpenProgress > 0.1}
                  nameValue={nameValue}
                  emailValue={emailValue}
                  messageValue={messageValue}
                />
              </div>
            )}

            {/* ---- Widget success ---- */}
            {showSuccess && (
              <div style={{ opacity: successFadeIn }}>
                <WidgetSuccess />
              </div>
            )}
          </div>
        )}

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
