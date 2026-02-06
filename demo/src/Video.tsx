import { useCurrentFrame, interpolate } from "remotion";
import { Terminal } from "./components/Terminal";
import { TerminalLine } from "./components/TerminalLine";
import { Cursor } from "./components/Cursor";
import { RawMarkdown } from "./components/RawMarkdown";
import { OutputReveal } from "./components/OutputReveal";
import { SCENES, CMD, FPS } from "./constants";
import { COLORS, FONT } from "./styles";

export const Video: React.FC = () => {
  const frame = useCurrentFrame();

  // Phase 1 visible until fade out completes
  const showRaw = frame < SCENES.rawEnd;
  // Phase 2+ visible after raw fades
  const showCmd = frame >= SCENES.promptStart;

  return (
    <Terminal>
      {/* Phase 1: Raw markdown with Mermaid code */}
      {showRaw && (
        <RawMarkdown startFrame={SCENES.rawStart} endFrame={SCENES.rawEnd} />
      )}

      {/* Phase 2: Prompt + type command */}
      {showCmd && (
        <>
          {/* Blinking cursor before typing starts */}
          {frame < SCENES.cmdStart && (
            <div
              style={{
                fontFamily: FONT.family,
                fontSize: FONT.size,
                lineHeight: FONT.lineHeight,
                color: COLORS.text,
              }}
            >
              <span style={{ color: COLORS.prompt }}>$ </span>
              <Cursor />
            </div>
          )}

          {/* Typing the command */}
          {frame >= SCENES.cmdStart && (
            <TerminalLine
              text={CMD}
              startFrame={SCENES.cmdStart}
              endFrame={SCENES.cmdEnd}
            />
          )}

          {/* Phase 3: Rendered output */}
          <OutputReveal startFrame={SCENES.outputStart} />
        </>
      )}
    </Terminal>
  );
};
