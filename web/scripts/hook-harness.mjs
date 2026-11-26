import fs from 'node:fs';
import vm from 'node:vm';
import ts from 'typescript';

// Render a component and flush effects when their dependencies change.
// Tests call render again after completing the operation that updates state.
export function createHookHarness() {
  const slots = [];
  const layoutEffects = [];
  const effects = [];
  let cursor = 0;
  const sameDependencies = (left, right) =>
    left !== undefined &&
    right !== undefined &&
    left.length === right.length &&
    right.every((value, index) => Object.is(value, left[index]));

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
        if (!(slot in slots)) slots[slot] = typeof initial === 'function' ? initial() : initial;
        return [
          slots[slot],
          (value) => {
            slots[slot] = typeof value === 'function' ? value(slots[slot]) : value;
          }
        ];
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

export function loadComponent(file, name, modules) {
  const harness = createHookHarness();
  const jsx = (type, props) => ({ type, props });
  const imports = {
    react: harness.hooks,
    'react/jsx-runtime': { jsx, jsxs: jsx },
    'react-i18next': { useTranslation: () => ({ t: (key) => key }) },
    ...modules
  };
  const code = ts.transpileModule(fs.readFileSync(file, 'utf8'), {
    compilerOptions: {
      jsx: ts.JsxEmit.ReactJSX,
      module: ts.ModuleKind.CommonJS,
      target: ts.ScriptTarget.ES2022
    },
    fileName: file
  }).outputText;
  const exports = {};
  vm.runInNewContext(
    code,
    {
      exports,
      require(id) {
        if (!Object.hasOwn(imports, id)) throw new Error(`Unexpected module ${id}`);
        return imports[id];
      }
    },
    { filename: file }
  );

  return {
    cleanup: harness.cleanup,
    render(props) {
      const nodes = [];
      function visit(node) {
        if (Array.isArray(node)) return node.forEach(visit);
        if (!node || typeof node !== 'object') return;
        nodes.push(node);
        visit(node.props?.children);
      }
      visit(harness.render(() => exports[name](props)));
      return nodes;
    }
  };
}
