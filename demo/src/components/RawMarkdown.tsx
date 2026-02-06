import { useCurrentFrame, interpolate, spring, useVideoConfig } from "remotion";
import { COLORS, FONT } from "../styles";
import { RAW_LINES, FPS } from "../constants";

interface RawMarkdownProps {
  startFrame: number;
  endFrame: number;
}

const colorize = (line: string): React.ReactNode => {
  // Command line
  if (line.startsWith("$ ")) {
    return (
      <>
        <span style={{ color: COLORS.prompt }}>$ </span>
        <span style={{ color: COLORS.text }}>{line.slice(2)}</span>
      </>
    );
  }
  // Heading
  if (line.startsWith("## ")) {
    return (
      <span style={{ color: COLORS.green, fontWeight: "bold" }}>{line}</span>
    );
  }
  // Code fence with mermaid
  if (line.startsWith("```mermaid")) {
    return (
      <>
        <span style={{ color: COLORS.dimText }}>```</span>
        <span style={{ color: COLORS.mermaidKeyword }}>mermaid</span>
      </>
    );
  }
  if (line === "```") {
    return <span style={{ color: COLORS.dimText }}>{line}</span>;
  }
  // Mermaid keywords
  if (line.trimStart().startsWith("flowchart")) {
    return <span style={{ color: COLORS.mermaidKeyword }}>{line}</span>;
  }
  // Default
  return <span style={{ color: COLORS.text }}>{line}</span>;
};

export const RawMarkdown: React.FC<RawMarkdownProps> = ({
  startFrame,
  endFrame,
}) => {
  const frame = useCurrentFrame();
  const { fps } = useVideoConfig();

  if (frame < startFrame) return null;

  // Fade out near end
  const fadeOutStart = endFrame - Math.round(0.3 * FPS);
  const opacity =
    frame >= fadeOutStart
      ? interpolate(frame, [fadeOutStart, endFrame], [1, 0], {
          extrapolateRight: "clamp",
        })
      : 1;

  // Lines appear one by one
  const totalLines = RAW_LINES.length;
  const typeWindow = Math.round(1.5 * FPS); // lines appear over 1.5s
  const elapsed = frame - startFrame;
  const visibleCount = Math.min(
    totalLines,
    Math.floor((elapsed / typeWindow) * totalLines) + 1,
  );

  return (
    <div style={{ opacity }}>
      {RAW_LINES.slice(0, visibleCount).map((line, i) => (
        <div
          key={i}
          style={{
            fontFamily: FONT.family,
            fontSize: FONT.size,
            lineHeight: FONT.lineHeight,
            minHeight: line === "" ? FONT.size * 0.8 : undefined,
            whiteSpace: "pre",
          }}
        >
          {colorize(line)}
        </div>
      ))}
    </div>
  );
};
