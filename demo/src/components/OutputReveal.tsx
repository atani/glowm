import {
  useCurrentFrame,
  Img,
  staticFile,
  interpolate,
  spring,
  useVideoConfig,
} from "remotion";
import { FPS } from "../constants";

interface OutputRevealProps {
  startFrame: number;
}

export const OutputReveal: React.FC<OutputRevealProps> = ({ startFrame }) => {
  const frame = useCurrentFrame();
  const { fps } = useVideoConfig();

  if (frame < startFrame) return null;

  // Phase A: Fade in (0-0.8s after start)
  const revealProgress = spring({
    frame: frame - startFrame,
    fps,
    config: { damping: 20, stiffness: 80 },
  });
  const opacity = interpolate(revealProgress, [0, 1], [0, 1]);
  const slideUp = interpolate(revealProgress, [0, 1], [20, 0]);

  // Phase B: Scroll image up to reveal Mermaid section (1.5s-3.5s after start)
  const scrollStart = startFrame + Math.round(1.5 * FPS);
  const scrollEnd = startFrame + Math.round(3.5 * FPS);
  // Scroll by ~40% of image height to reveal Mermaid diagram at bottom
  const scrollAmount = interpolate(frame, [scrollStart, scrollEnd], [0, 40], {
    extrapolateLeft: "clamp",
    extrapolateRight: "clamp",
  });

  return (
    <div
      style={{
        opacity,
        transform: `translateY(${slideUp}px)`,
        marginTop: 8,
        borderRadius: 4,
        overflow: "hidden",
        boxShadow: "0 2px 12px rgba(0,0,0,0.3)",
        maxHeight: 380,
      }}
    >
      <Img
        src={staticFile("README-demo.png")}
        style={{
          width: "100%",
          display: "block",
          transform: `translateY(-${scrollAmount}%)`,
        }}
      />
    </div>
  );
};
