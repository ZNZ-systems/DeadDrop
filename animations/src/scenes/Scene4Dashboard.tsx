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
  Badge,
  DomainRow,
  MessageRow,
  ListCard,
  SectionDivider,
} from "../components/ui";

import { Cursor, FadeIn } from "../components/animation";

import { COLORS, SPRING_PRESETS } from "../constants/theme";
import { fontFamilyHeading, fontFamilyMono } from "../fonts/load";

/**
 * Scene 4 -- Dashboard
 *
 * Duration: 150 frames (5.0s @ 30fps)
 *
 * Timeline:
 *   Phase 1  (0-29)     Dashboard view with domain list
 *   Phase 2  (30-54)    Hover and click on "mycoolproject.com"
 *   Phase 3  (55-89)    Transition to domain detail with messages
 *   Phase 4  (90-114)   Cursor moves to "Mark Read" on first message
 *   Phase 5  (115-149)  Message state changes from unread to read
 */
export const Scene4Dashboard: React.FC = () => {
  const frame = useCurrentFrame();
  const { fps } = useVideoConfig();

  // ========================================================================
  // PHASE 1 -- Dashboard view (frames 0-29)
  // ========================================================================
  const BADGE_BOUNCE_FRAME = 12;
  const UNREAD_INCREMENT_FRAME = 15;

  // ========================================================================
  // PHASE 2 -- Hover and click domain (frames 30-54)
  // ========================================================================
  const CURSOR_APPEAR = 30;
  const ROW_HOVER_START = 35;
  const ROW_CLICK = 45;
  const ROW_CLICK_END = 49;

  // ========================================================================
  // PHASE 3 -- Transition to domain detail (frames 55-89)
  // ========================================================================
  const DASHBOARD_FADE_START = 55;
  const DASHBOARD_FADE_END = 62;
  const DETAIL_FADE_START = 63;
  const DETAIL_FADE_END = 70;
  const MSG1_FADE = 65;
  const MSG2_FADE = 70;
  const MSG3_FADE = 75;

  // ========================================================================
  // PHASE 4 -- Mark message as read (frames 90-114)
  // ========================================================================
  const MARK_CURSOR_APPEAR = 90;
  const MARK_CURSOR_HOVER = 96;
  const MARK_CURSOR_CLICK = 100;
  const MARK_CURSOR_CLICK_END = 104;

  // ========================================================================
  // PHASE 5 -- Message state changes (frames 115-149)
  // ========================================================================
  const STATE_CHANGE_START = 115;
  const STATE_CHANGE_END = 125;

  // ---------- Derived: Unread count for mycoolproject.com ----------
  const mycoolUnread = frame < UNREAD_INCREMENT_FRAME ? 2 : 3;

  // ---------- Derived: Badge bounce animation ----------
  const badgeBounceProgress = spring({
    frame,
    fps,
    config: SPRING_PRESETS.snappy,
    delay: BADGE_BOUNCE_FRAME,
  });
  // Scale: 1 -> 1.15 -> 1 using spring overshoot
  const badgeScale = interpolate(badgeBounceProgress, [0, 0.5, 1], [1, 1.15, 1]);

  // ---------- Derived: Domain row hover state ----------
  const mycoolHovered = frame >= ROW_HOVER_START && frame < DASHBOARD_FADE_START;

  // ---------- Derived: Dashboard opacity ----------
  const dashboardOpacity = interpolate(
    frame,
    [DASHBOARD_FADE_START, DASHBOARD_FADE_END],
    [1, 0],
    { extrapolateLeft: "clamp", extrapolateRight: "clamp" },
  );

  // ---------- Derived: Detail page opacity ----------
  const detailOpacity = interpolate(
    frame,
    [DETAIL_FADE_START, DETAIL_FADE_END],
    [0, 1],
    { extrapolateLeft: "clamp", extrapolateRight: "clamp" },
  );

  // ---------- Derived: Browser URL ----------
  const browserUrl = frame < DASHBOARD_FADE_END
    ? "deaddrop.io"
    : "deaddrop.io/domains/mycoolproject.com";

  // ---------- Derived: Phase 2 cursor (dashboard) ----------
  const phase2CursorVisible = frame >= CURSOR_APPEAR && frame < DASHBOARD_FADE_START;

  const phase2CursorX = interpolate(
    frame,
    [CURSOR_APPEAR, ROW_HOVER_START, ROW_CLICK],
    [700, 500, 500],
    { extrapolateLeft: "clamp", extrapolateRight: "clamp" },
  );
  const phase2CursorY = interpolate(
    frame,
    [CURSOR_APPEAR, ROW_HOVER_START, ROW_CLICK],
    [250, 290, 290],
    { extrapolateLeft: "clamp", extrapolateRight: "clamp" },
  );
  const phase2CursorClicking = frame >= ROW_CLICK && frame < ROW_CLICK_END;

  // ---------- Derived: Phase 4 cursor (mark read) ----------
  const phase4CursorVisible = frame >= MARK_CURSOR_APPEAR && frame <= MARK_CURSOR_CLICK_END;

  const phase4CursorX = interpolate(
    frame,
    [MARK_CURSOR_APPEAR, MARK_CURSOR_HOVER, MARK_CURSOR_CLICK],
    [600, 330, 330],
    { extrapolateLeft: "clamp", extrapolateRight: "clamp" },
  );
  const phase4CursorY = interpolate(
    frame,
    [MARK_CURSOR_APPEAR, MARK_CURSOR_HOVER, MARK_CURSOR_CLICK],
    [250, 390, 390],
    { extrapolateLeft: "clamp", extrapolateRight: "clamp" },
  );
  const phase4CursorClicking = frame >= MARK_CURSOR_CLICK && frame < MARK_CURSOR_CLICK_END;

  // ---------- Derived: Message 1 read transition ----------
  const msg1IsRead = frame >= STATE_CHANGE_END;
  // For smooth transition of border/dot/button disappearance
  const msg1UnreadElementsOpacity = interpolate(
    frame,
    [STATE_CHANGE_START, STATE_CHANGE_END],
    [1, 0],
    { extrapolateLeft: "clamp", extrapolateRight: "clamp" },
  );

  // ---------- Pick active cursor ----------
  const cursorVisible = phase2CursorVisible || phase4CursorVisible;
  const cursorX = phase2CursorVisible ? phase2CursorX : phase4CursorX;
  const cursorY = phase2CursorVisible ? phase2CursorY : phase4CursorY;
  const cursorClicking = phase2CursorVisible
    ? phase2CursorClicking
    : phase4CursorClicking;

  // ========================================================================
  // RENDER
  // ========================================================================
  return (
    <AbsoluteFill>
      <MockBrowser url={browserUrl}>
        <AppShell email="demo@deaddrop.io">
          {/* ============ DASHBOARD PAGE ============ */}
          <div
            style={{
              opacity: dashboardOpacity,
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
              pointerEvents: frame >= DASHBOARD_FADE_END ? "none" : "auto",
            }}
          >
            <FadeIn startFrame={0} durationFrames={12}>
              {/* Page header */}
              <div
                style={{
                  display: "flex",
                  alignItems: "center",
                  justifyContent: "space-between",
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
                  Your Domains
                </span>
                <Button label="ADD DOMAIN" variant="primary" />
              </div>

              {/* Domain list */}
              <ListCard>
                {/* Row 1: mycoolproject.com -- badge bounces on unread increment */}
                <div
                  style={{
                    position: "relative",
                    transform: frame >= BADGE_BOUNCE_FRAME && frame < BADGE_BOUNCE_FRAME + 20
                      ? `scale(${badgeScale})`
                      : undefined,
                    transformOrigin: "right center",
                  }}
                >
                  <DomainRow
                    name="mycoolproject.com"
                    verified={true}
                    unreadCount={mycoolUnread}
                    hovered={mycoolHovered}
                  />
                </div>

                {/* Row 2: startup.dev */}
                <DomainRow
                  name="startup.dev"
                  verified={true}
                  unreadCount={1}
                />

                {/* Row 3: docs.example.org */}
                <DomainRow
                  name="docs.example.org"
                  verified={true}
                  unreadCount={0}
                />

                {/* Row 4: blog.janedoe.com */}
                <DomainRow
                  name="blog.janedoe.com"
                  verified={false}
                  unreadCount={0}
                />
              </ListCard>
            </FadeIn>
          </div>

          {/* ============ DOMAIN DETAIL PAGE ============ */}
          <div
            style={{
              opacity: detailOpacity,
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
              pointerEvents: frame >= DETAIL_FADE_START ? "auto" : "none",
              overflow: "auto",
            }}
          >
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

            {/* Section divider */}
            <SectionDivider
              num="01"
              label="Messages"
              style={{ marginBottom: 0 }}
            />

            {/* Messages list */}
            <ListCard>
              {/* Message 1: Jane Smith (transitions from unread to read) */}
              <FadeIn startFrame={MSG1_FADE} durationFrames={8}>
                <div style={{ position: "relative" }}>
                  {/* Smooth border-left transition overlay */}
                  {frame >= STATE_CHANGE_START && !msg1IsRead && (
                    <div
                      style={{
                        position: "absolute",
                        top: 0,
                        left: 0,
                        bottom: 0,
                        width: 4,
                        backgroundColor: COLORS.red,
                        opacity: msg1UnreadElementsOpacity,
                        zIndex: 2,
                      }}
                    />
                  )}
                  <MessageRow
                    senderName="Jane Smith"
                    senderEmail="jane@example.com"
                    body="Love the project! How can I contribute?"
                    time="Feb 12, 2026 2:30 PM"
                    isRead={msg1IsRead}
                    showMarkRead={frame < STATE_CHANGE_START}
                    style={
                      frame >= STATE_CHANGE_START && !msg1IsRead
                        ? {
                            borderLeft: `4px solid rgba(255, 34, 0, ${msg1UnreadElementsOpacity})`,
                          }
                        : undefined
                    }
                  />
                  {/* Fade out the unread dot and Mark Read button during transition */}
                  {frame >= STATE_CHANGE_START && frame < STATE_CHANGE_END && (
                    <div
                      style={{
                        position: "absolute",
                        top: 0,
                        left: 0,
                        right: 0,
                        bottom: 0,
                        pointerEvents: "none",
                        // This overlay effect is handled by the isRead/showMarkRead toggles
                      }}
                    />
                  )}
                </div>
              </FadeIn>

              {/* Message 2: Alex Chen */}
              <FadeIn startFrame={MSG2_FADE} durationFrames={8}>
                <MessageRow
                  senderName="Alex Chen"
                  senderEmail="alex@startup.dev"
                  body="Would love to discuss a partnership opportunity."
                  time="Feb 12, 2026 1:15 PM"
                  isRead={false}
                  showMarkRead={true}
                />
              </FadeIn>

              {/* Message 3: Sam Rivera */}
              <FadeIn startFrame={MSG3_FADE} durationFrames={8}>
                <MessageRow
                  senderName="Sam Rivera"
                  senderEmail="sam@example.com"
                  body="Thanks for the quick response! Looking forward to it."
                  time="Feb 11, 2026 4:45 PM"
                  isRead={true}
                  showMarkRead={false}
                />
              </FadeIn>
            </ListCard>
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
