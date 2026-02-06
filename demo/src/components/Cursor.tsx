import { useCurrentFrame } from "remotion";
import { COLORS, FONT } from "../styles";

export const Cursor: React.FC<{ show?: boolean }> = ({ show = true }) => {
  const frame = useCurrentFrame();
  const visible = show && Math.floor(frame / 15) % 2 === 0;

  return (
    <span
      style={{
        display: "inline-block",
        width: FONT.size * 0.6,
        height: FONT.size * 1.2,
        backgroundColor: visible ? COLORS.cursor : "transparent",
        verticalAlign: "text-bottom",
        marginLeft: 2,
      }}
    />
  );
};
