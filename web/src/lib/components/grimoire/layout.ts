export function circleLayout(
  count: number,
  centerX: number,
  centerY: number,
  radius: number,
): { x: number; y: number }[] {
  return Array.from({ length: count }, (_, i) => {
    const angle = (2 * Math.PI * i) / count - Math.PI / 2;
    return {
      x: centerX + radius * Math.cos(angle),
      y: centerY + radius * Math.sin(angle),
    };
  });
}

// Reminder token attachment constants
export const ORBIT_RADIUS = 76;
export const ATTACH_THRESHOLD = 80;
export const DETACH_THRESHOLD = 100;

export function orbitPosition(
  playerX: number,
  playerY: number,
  angle: number,
): { x: number; y: number } {
  return {
    x: playerX + ORBIT_RADIUS * Math.cos(angle),
    y: playerY + ORBIT_RADIUS * Math.sin(angle),
  };
}

export function distributeAngles(count: number): number[] {
  return Array.from(
    { length: count },
    (_, i) => (2 * Math.PI * i) / count - Math.PI / 2,
  );
}

export function angleFromPosition(
  px: number,
  py: number,
  cx: number,
  cy: number,
): number {
  return Math.atan2(py - cy, px - cx);
}

export function distanceBetween(
  x1: number,
  y1: number,
  x2: number,
  y2: number,
): number {
  return Math.hypot(x2 - x1, y2 - y1);
}
