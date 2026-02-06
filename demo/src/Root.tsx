import { Composition } from "remotion";
import { Video } from "./Video";
import { FPS, TOTAL_FRAMES, WIDTH, HEIGHT } from "./constants";

export const RemotionRoot: React.FC = () => {
  return (
    <Composition
      id="GlowmDemo"
      component={Video}
      durationInFrames={TOTAL_FRAMES}
      fps={FPS}
      width={WIDTH}
      height={HEIGHT}
    />
  );
};
