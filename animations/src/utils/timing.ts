import { VIDEO } from "../constants/theme";

export const sec = (seconds: number): number =>
  Math.round(seconds * VIDEO.fps);

export const ms = (milliseconds: number): number =>
  Math.round((milliseconds / 1000) * VIDEO.fps);

export const charFrames = (
  text: string,
  framesPerChar: number = 2,
): number => {
  return text.length * framesPerChar;
};

export const getTypedText = (
  frame: number,
  fullText: string,
  framesPerChar: number = 2,
  startFrame: number = 0,
): string => {
  const elapsed = Math.max(0, frame - startFrame);
  const chars = Math.min(fullText.length, Math.floor(elapsed / framesPerChar));
  return fullText.slice(0, chars);
};

export const isInRange = (
  frame: number,
  start: number,
  end: number,
): boolean => {
  return frame >= start && frame < end;
};
