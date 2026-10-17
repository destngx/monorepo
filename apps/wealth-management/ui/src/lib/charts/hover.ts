// Chart readouts (WM-UI-15): a hovered column stays in the readout while the
// pointer moves from the chart onto the readout, so a masked value there can be
// clicked. Charts clear the hover when the pointer leaves the chart and its
// readout together, except for touch, where the last tap stays shown.
export function leavesReadout(event: PointerEvent): boolean {
  return event.pointerType !== 'touch';
}
