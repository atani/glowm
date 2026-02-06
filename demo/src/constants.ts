export const FPS = 30;
export const DURATION_SEC = 12;
export const TOTAL_FRAMES = FPS * DURATION_SEC; // 360

export const WIDTH = 800;
export const HEIGHT = 500;

// Scene timing (in frames)
export const SCENES = {
  // Phase 1: Show raw markdown with Mermaid code (0-3.5s)
  rawStart: 0,
  rawEnd: Math.round(3.5 * FPS), // 105

  // Phase 2: Prompt appears, type command (4-6s)
  promptStart: Math.round(4 * FPS), // 120
  cmdStart: Math.round(4.5 * FPS), // 135
  cmdEnd: Math.round(6 * FPS), // 180

  // Phase 3: Rendered output reveal (6.5-9s)
  outputStart: Math.round(6.5 * FPS), // 195

  // Phase 4: Hold until end (9-12s)
} as const;

export const CMD = "glowm docs/sample.md";

// Raw markdown lines to display (showing the Mermaid section)
export const RAW_LINES = [
  "$ cat docs/sample.md",
  "",
  "## Mermaid",
  "",
  "```mermaid",
  "flowchart LR",
  "  A[Write Markdown] --> B[Run glowm]",
  "  B --> C{Terminal}",
  "  C -->|iTerm2/Kitty| D[Inline Mermaid]",
  "  C -->|Other| E[Mermaid code]",
  "  B --> F[PDF export]",
  "```",
];
