import React from "react";
import {
  AbsoluteFill,
  useCurrentFrame,
  interpolate,
} from "remotion";

import { MockBrowser } from "../components/layout";
import { AppShell } from "../components/layout";
import { MockDnsPanel } from "../components/layout";

import {
  Button,
  Badge,
  InfoPanel,
  CodeBlock,
  CodeHL,
  CodeVal,
} from "../components/ui";

import { Cursor, ScaleIn } from "../components/animation";

import { COLORS } from "../constants/theme";
import { fontFamilyHeading, fontFamilyMono } from "../fonts/load";

/**
 * Scene 2 -- DNS Verification
 *
 * Duration: 165 frames (5.5s @ 30fps)
 *
 * Timeline:
 *   Phase 1  (0-29)    Domain detail page shown, cursor copies TXT record
 *   Phase 2  (30-44)   Crossfade to DNS panel
 *   Phase 3  (45-104)  New TXT row slides in, typing the record value
 *   Phase 4  (105-114) Cursor clicks Save
 *   Phase 5  (115-139) Crossfade back to DeadDrop, cursor clicks CHECK button
 *   Phase 6  (140-164) Verification success - badge + panel transition
 */
export const Scene2DnsVerification: React.FC = () => {
  const frame = useCurrentFrame();

  // ---------- Timing constants ----------
  const CURSOR_APPEAR = 10;
  const COPIED_APPEAR = 20;
  const CROSSFADE_TO_DNS_START = 30;
  const CROSSFADE_TO_DNS_END = 44;
  const NEW_ROW_APPEAR = 48;
  const DNS_TYPING_START = 52;
  const DNS_TYPING_FPC = 1;
  const TXT_VALUE = "deaddrop-verify=a9c55678-1234-5678-abcd-ef0123456789";
  const DNS_TYPING_END = DNS_TYPING_START + TXT_VALUE.length * DNS_TYPING_FPC; // ~103
  const SAVE_MOVE_START = 105;
  const SAVE_CLICK_START = 108;
  const SAVE_CLICK_END = 112;
  const CROSSFADE_BACK_START = 115;
  const CROSSFADE_BACK_END = 124;
  const CHECK_MOVE_START = 125;
  const CHECK_CLICK_START = 130;
  const CHECK_CLICK_END = 134;
  const VERIFY_SUCCESS_START = 140;

  // ---------- Derived: which "page" is showing ----------

  // DeadDrop page opacity
  const deaddropOpacity1 = interpolate(
    frame,
    [CROSSFADE_TO_DNS_START, CROSSFADE_TO_DNS_END],
    [1, 0],
    { extrapolateLeft: "clamp", extrapolateRight: "clamp" },
  );

  const deaddropOpacity2 = interpolate(
    frame,
    [CROSSFADE_BACK_START, CROSSFADE_BACK_END],
    [0, 1],
    { extrapolateLeft: "clamp", extrapolateRight: "clamp" },
  );

  // Before first crossfade: show deaddrop. During DNS: hidden. After crossfade back: show deaddrop.
  const showDns = frame >= CROSSFADE_TO_DNS_START && frame < CROSSFADE_BACK_END;
  const deaddropOpacity = showDns
    ? frame < CROSSFADE_TO_DNS_END
      ? deaddropOpacity1
      : frame >= CROSSFADE_BACK_START
        ? deaddropOpacity2
        : 0
    : 1;

  // DNS panel opacity
  const dnsOpacityIn = interpolate(
    frame,
    [CROSSFADE_TO_DNS_START, CROSSFADE_TO_DNS_END],
    [0, 1],
    { extrapolateLeft: "clamp", extrapolateRight: "clamp" },
  );

  const dnsOpacityOut = interpolate(
    frame,
    [CROSSFADE_BACK_START, CROSSFADE_BACK_END],
    [1, 0],
    { extrapolateLeft: "clamp", extrapolateRight: "clamp" },
  );

  const dnsOpacity =
    frame < CROSSFADE_TO_DNS_START
      ? 0
      : frame < CROSSFADE_BACK_START
        ? dnsOpacityIn
        : dnsOpacityOut;

  // ---------- URL ----------
  const browserUrl =
    frame >= CROSSFADE_TO_DNS_START && frame < CROSSFADE_BACK_END
      ? "dash.cloudflare.com/dns/mycoolproject.com"
      : "deaddrop.io/domains/mycoolproject.com";

  // ---------- DNS record typing ----------
  const dnsCharsTyped = Math.min(
    TXT_VALUE.length,
    Math.max(0, Math.floor((frame - DNS_TYPING_START) / DNS_TYPING_FPC)),
  );
  const dnsTypedContent = TXT_VALUE.slice(0, dnsCharsTyped);

  // Show new row after NEW_ROW_APPEAR
  const showNewRow = frame >= NEW_ROW_APPEAR;

  const dnsRecords = [
    { type: "A", name: "mycoolproject.com", content: "76.76.21.21" },
    { type: "CNAME", name: "www", content: "mycoolproject.com" },
    ...(showNewRow
      ? [
          {
            type: "TXT",
            name: "mycoolproject.com",
            content: dnsTypedContent,
            isNew: true,
          },
        ]
      : []),
  ];

  // ---------- "Copied!" tooltip ----------
  const copiedVisible = frame >= COPIED_APPEAR && frame < CROSSFADE_TO_DNS_START;

  // ---------- Verification success transition ----------
  const isVerified = frame >= VERIFY_SUCCESS_START;

  // Panel border color interpolation (yellow -> green)
  const borderProgress = interpolate(
    frame,
    [VERIFY_SUCCESS_START, VERIFY_SUCCESS_START + 10],
    [0, 1],
    { extrapolateLeft: "clamp", extrapolateRight: "clamp" },
  );

  // Interpolate border color components: yellow (#eab308) -> green (#22c55e)
  const borderR = Math.round(interpolate(borderProgress, [0, 1], [0xea, 0x22]));
  const borderG = Math.round(interpolate(borderProgress, [0, 1], [0xb3, 0xc5]));
  const borderB = Math.round(interpolate(borderProgress, [0, 1], [0x08, 0x5e]));
  const panelBorderColor = `rgb(${borderR}, ${borderG}, ${borderB})`;

  // ---------- Cursor position (multi-phase) ----------

  // Phase 1: move toward code block
  // Phase 2-3: sit near DNS content field
  // Phase 4: move to Save
  // Phase 5: move to CHECK VERIFICATION button
  const cursorX = interpolate(
    frame,
    [
      CURSOR_APPEAR,         // 10 - appear
      20,                    // 20 - at code block
      CROSSFADE_TO_DNS_END,  // 44 - arrive in DNS panel
      DNS_TYPING_END,        // ~103 - still near content
      SAVE_MOVE_START,       // 105 - start moving to Save
      SAVE_MOVE_START + 3,   // 108 - at Save
      CROSSFADE_BACK_END,    // 124 - back on DeadDrop
      CHECK_MOVE_START,      // 125 - start moving to check btn
      CHECK_MOVE_START + 4,  // 129 - at check btn
    ],
    [
      500, // entering from right
      460, // at code block
      580, // DNS content field area
      580, // still there
      780, // moving to Save button
      780, // at Save
      340, // back on DeadDrop page
      340, // start towards check button
      340, // at check button
    ],
    { extrapolateLeft: "clamp", extrapolateRight: "clamp" },
  );

  const cursorY = interpolate(
    frame,
    [
      CURSOR_APPEAR,
      20,
      CROSSFADE_TO_DNS_END,
      DNS_TYPING_END,
      SAVE_MOVE_START,
      SAVE_MOVE_START + 3,
      CROSSFADE_BACK_END,
      CHECK_MOVE_START,
      CHECK_MOVE_START + 4,
    ],
    [
      350, // entering
      420, // at code block
      340, // DNS content row area
      340, // still there
      560, // Save button
      560, // at Save
      530, // back on DeadDrop
      530, // start towards check
      530, // at check button
    ],
    { extrapolateLeft: "clamp", extrapolateRight: "clamp" },
  );

  const cursorVisible = frame >= CURSOR_APPEAR && frame < VERIFY_SUCCESS_START;

  const cursorClicking =
    (frame >= SAVE_CLICK_START && frame <= SAVE_CLICK_END) ||
    (frame >= CHECK_CLICK_START && frame <= CHECK_CLICK_END);

  // ---------- Badge + panel content after verification ----------
  const badgeVariant = isVerified ? "verified" : "unverified";
  const badgeText = isVerified ? "Verified" : "Unverified";
  const panelVariant = isVerified ? "success" : "warn";
  const panelTitle = isVerified ? "Widget Embed Code" : "DNS Verification Required";

  return (
    <AbsoluteFill>
      <MockBrowser url={browserUrl}>
        {/* ============ DEADDROP PAGE ============ */}
        <div
          style={{
            position: "absolute",
            top: 0,
            left: 0,
            right: 0,
            bottom: 0,
            opacity: deaddropOpacity,
            pointerEvents: deaddropOpacity > 0.1 ? "auto" : "none",
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

              {/* Badge with bounce on verification */}
              {isVerified ? (
                <ScaleIn
                  startFrame={VERIFY_SUCCESS_START}
                  config={{ damping: 8, mass: 1, stiffness: 100, overshootClamping: false }}
                >
                  <Badge variant={badgeVariant} text={badgeText} />
                </ScaleIn>
              ) : (
                <Badge variant={badgeVariant} text={badgeText} />
              )}
            </div>

            {/* Info panel with animated border */}
            <InfoPanel
              title={panelTitle}
              variant={panelVariant}
              style={{
                borderColor: panelBorderColor,
                position: "relative",
              }}
            >
              {/* ---- Pre-verification content ---- */}
              <div
                style={{
                  opacity: isVerified ? 0 : 1,
                  position: isVerified ? "absolute" : "relative",
                  top: isVerified ? 0 : undefined,
                  left: isVerified ? 0 : undefined,
                  right: isVerified ? 0 : undefined,
                  pointerEvents: isVerified ? "none" : "auto",
                }}
              >
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
                <CodeBlock style={{ position: "relative" }}>
                  <CodeHL>deaddrop-verify</CodeHL>=
                  <CodeVal>
                    a9c55678-1234-5678-abcd-ef0123456789
                  </CodeVal>

                  {/* "Copied!" tooltip */}
                  {copiedVisible && (
                    <ScaleIn startFrame={COPIED_APPEAR}>
                      <div
                        style={{
                          position: "absolute",
                          top: -8,
                          right: 16,
                          fontFamily: fontFamilyMono,
                          fontSize: 10,
                          fontWeight: 700,
                          textTransform: "uppercase",
                          letterSpacing: "0.1em",
                          color: COLORS.green,
                          backgroundColor: COLORS.bg,
                          padding: "3px 10px",
                          border: `2px solid ${COLORS.green}`,
                        }}
                      >
                        COPIED!
                      </div>
                    </ScaleIn>
                  )}
                </CodeBlock>
                <div style={{ marginTop: "1.25rem" }}>
                  <Button
                    label="CHECK VERIFICATION"
                    variant="outline"
                    size="sm"
                    hovered={
                      frame >= CHECK_MOVE_START &&
                      frame < VERIFY_SUCCESS_START
                    }
                  />
                </div>
              </div>

              {/* ---- Post-verification content ---- */}
              <div
                style={{
                  opacity: isVerified ? 1 : 0,
                  position: isVerified ? "relative" : "absolute",
                  top: isVerified ? undefined : 0,
                  left: isVerified ? undefined : 0,
                  right: isVerified ? undefined : 0,
                  pointerEvents: isVerified ? "auto" : "none",
                }}
              >
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
                <CodeBlock>
                  <CodeHL>&lt;script</CodeHL>{" "}
                  src=<CodeVal>"deaddrop.io/static/widget.js"</CodeVal>{" "}
                  data-deaddrop-id=<CodeVal>"abc-123"</CodeVal>
                  <CodeHL>&gt;&lt;/script&gt;</CodeHL>
                </CodeBlock>
              </div>
            </InfoPanel>
          </AppShell>
        </div>

        {/* ============ DNS PANEL ============ */}
        <div
          style={{
            position: "absolute",
            top: 52, // below browser chrome bar
            left: 0,
            right: 0,
            bottom: 0,
            opacity: dnsOpacity,
            pointerEvents: dnsOpacity > 0.1 ? "auto" : "none",
          }}
        >
          <MockDnsPanel records={dnsRecords} />
        </div>

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
