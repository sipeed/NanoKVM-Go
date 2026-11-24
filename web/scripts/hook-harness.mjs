// Render a component and flush effects when their dependencies change.
// Tests call render again after completing the operation that updates state.
export function createHookHarness() {
  const slots = [];
  const layoutEffects = [];
  const effects = [];
  let cursor = 0;
  const sameDependencies = (left, right) =>
    left?.length === right.length && right.every((value, index) => Object.is(value, left[index]));

  function schedule(queue, effect, dependencies) {
    const slot = cursor++;
    const previous = slots[slot];
    if (sameDependencies(previous?.dependencies, dependencies)) return;
    const entry = { dependencies };
    slots[slot] = entry;
    queue.push(() => {
      previous?.cleanup?.();
      entry.cleanup = effect();
    });
  }

  return {
    hooks: {
      useState(initial) {
        const slot = cursor++;
        if (!(slot in slots)) slots[slot] = initial;
        return [slots[slot], (value) => { slots[slot] = value; }];
      },
      useRef(initial) {
        const slot = cursor++;
        if (!(slot in slots)) slots[slot] = { current: initial };
        return slots[slot];
      },
      useCallback(callback, dependencies) {
        const slot = cursor++;
        if (!sameDependencies(slots[slot]?.dependencies, dependencies)) {
          slots[slot] = { callback, dependencies };
        }
        return slots[slot].callback;
      },
      useLayoutEffect: (effect, deps) => schedule(layoutEffects, effect, deps),
      useEffect: (effect, deps) => schedule(effects, effect, deps)
    },
    render(component) {
      cursor = 0;
      const tree = component();
      for (const effect of layoutEffects.splice(0)) effect();
      for (const effect of effects.splice(0)) effect();
      return tree;
    },
    cleanup() {
      slots.forEach((slot) => slot?.cleanup?.());
    }
  };
}
