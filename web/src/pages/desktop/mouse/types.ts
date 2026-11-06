interface MouseMoveAbsoluteEvent {
  type: 'move';
  x: number;
  y: number;
}

interface MouseMoveRelativeEvent {
  type: 'move';
  deltaX: number;
  deltaY: number;
}

interface MouseButtonEvent {
  type: 'mousedown' | 'mouseup';
  button: number;
}

interface MouseWheelEvent {
  type: 'wheel';
  deltaY: number;
}

interface MouseReleaseEvent {
  type: 'release';
}

export type MouseAbsoluteEvent =
  | MouseMoveAbsoluteEvent
  | MouseButtonEvent
  | MouseWheelEvent
  | MouseReleaseEvent;

export type MouseRelativeEvent =
  | MouseMoveRelativeEvent
  | MouseButtonEvent
  | MouseWheelEvent
  | MouseReleaseEvent;
