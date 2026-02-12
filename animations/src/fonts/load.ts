import { loadFont as loadSpaceGrotesk } from "@remotion/google-fonts/SpaceGrotesk";
import { loadFont as loadSpaceMono } from "@remotion/google-fonts/SpaceMono";

const spaceGrotesk = loadSpaceGrotesk("normal", {
  weights: ["400", "500", "600", "700"],
  subsets: ["latin"],
});

const spaceMono = loadSpaceMono("normal", {
  weights: ["400", "700"],
  subsets: ["latin"],
});

export const fontFamilyHeading = spaceGrotesk.fontFamily;
export const fontFamilyMono = spaceMono.fontFamily;
