import { useCurrentFrame } from "remotion";
import { COLORS, FONT } from "../styles";
import { Cursor } from "./Cursor";

interface TerminalLineProps {
  text: string;
  startFrame: number;
  endFrame: number;
  prompt?: string;
  color?: string;
}

export const TerminalLine: React.FC<TerminalLineProps> = ({
  text,
  startFrame,
  endFrame,
  prompt = "$ ",
  color = COLORS.text,
}) => {
  const frame = useCurrentFrame();

  if (frame < startFrame) return null;

  const typingDuration = endFrame - startFrame;
  const progress = Math.min((frame - startFrame) / typingDuration, 1);
  const charsToShow = Math.floor(progress * text.length);
  const displayedText = text.slice(0, charsToShow);
  const isTyping = progress < 1;

  return (
    <div
      style={{
        fontFamily: FONT.family,
        fontSize: FONT.size,
        lineHeight: FONT.lineHeight,
        color,
        whiteSpace: "pre",
      }}
    >
      <span style={{ color: COLORS.prompt }}>{prompt}</span>
      {displayedText}
      {isTyping && <Cursor />}
    </div>
  );
};
