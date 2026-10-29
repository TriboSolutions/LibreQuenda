// Digits in braille, drawn to scale for printing: the number sign, then a-j for
// 1-9 and 0. Standard sizes (as ONCE uses): 1.5 mm dots, 2.5 mm between the dots
// of a cell, 6 mm from cell to cell. On plain paper the dots come out flat; they
// only rise on swell (microcapsule) paper or from a braille embosser.

// dots of each cell, numbered 1-2-3 down the left column and 4-5-6 down the right
const NUMBER_SIGN = [3, 4, 5, 6]
const DIGITS: Record<string, number[]> = {
  '1': [1],
  '2': [1, 2],
  '3': [1, 4],
  '4': [1, 4, 5],
  '5': [1, 5],
  '6': [1, 2, 4],
  '7': [1, 2, 4, 5],
  '8': [1, 2, 5],
  '9': [2, 4],
  '0': [2, 4, 5],
}

const DOT = 1.5
const PITCH = 2.5
const CELL = 6
const MARGIN = 1

/** The cells of a number: the number sign, then one cell per digit. */
function brailleCells(digits: string) {
  return [NUMBER_SIGN, ...[...digits].filter((d) => d in DIGITS).map((d) => DIGITS[d])]
}

/** Decorative for screen readers: the same digits are printed next to it as text. */
export function BrailleCode({ code, className }: { code: string; className?: string }) {
  const cells = brailleCells(code)
  const width = MARGIN * 2 + (cells.length - 1) * CELL + PITCH + DOT
  const height = MARGIN * 2 + 2 * PITCH + DOT
  return (
    <svg
      aria-hidden
      className={className}
      width={`${width}mm`}
      height={`${height}mm`}
      viewBox={`0 0 ${width} ${height}`}
      fill="currentColor"
    >
      {cells.flatMap((dots, i) =>
        dots.map((dot) => (
          <circle
            key={`${i}-${dot}`}
            cx={MARGIN + DOT / 2 + i * CELL + (dot > 3 ? PITCH : 0)}
            cy={MARGIN + DOT / 2 + ((dot - 1) % 3) * PITCH}
            r={DOT / 2}
          />
        )),
      )}
    </svg>
  )
}
