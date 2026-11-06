(() => {
  'use strict';

  const LOCAL_LIMIT = 1200;
  const TABLE_LIMIT = 200;
  const RATE_WINDOW_MS = 2000;
  const STATUS_POLL_MS = 1000;
  const STREAM_RECONNECT_MS = 500;
  const RENDER_INTERVAL_MS = 100;
  const POINT_COLORS = ['#ff637d', '#39d8ff', '#54e598', '#ffc857', '#a98bff', '#52ead8', '#ff9859', '#e97ac2', '#a5dc62', '#78a7ff'];
  const FILTER_IDS = ['filter-api', 'filter-event', 'filter-pointer', 'filter-kind', 'filter-output', 'filter-session', 'filter-text'];
  const ELEMENT_IDS = [
    'connection', 'auth-warning', 'pause', 'clear', 'copy', 'export',
    'metric-capture-rate', 'metric-capture-total', 'metric-router-rate', 'metric-router-total',
    'metric-output-rate', 'metric-output-total', 'metric-routing-ratio', 'metric-client-latency',
    'metric-router-latency', 'metric-hid-latency', 'metric-errors', 'selected-summary',
    'node-capture', 'browser-event-json', 'node-mapping', 'node-route', 'router-flow-graph', 'node-encode',
    'node-dispatch', 'visible-count', ...FILTER_IDS, 'trace-body', 'point-canvas',
    'point-count', 'point-legend', 'raw-json', 'routes', 'sessions', 'performance',
    'sources', 'debug-health'
  ];
  const state = {
    traces: [],
    transactions: [],
    transactionsVersion: 0,
    transactionsDirty: false,
    seen: new Set(),
    arrivals: new Map(),
    status: null,
    renderedStatus: undefined,
    renderedStatusSignature: '',
    selectedKey: '',
    autoFollow: true,
    ignoredUntil: 0,
    paused: false,
    authorized: true,
    statusTimer: 0,
    statusRequestActive: false,
    streamAbort: null,
    streamConnected: false,
    streamError: null,
    streamReconnectTimer: 0,
    renderPending: false,
    renderTimer: 0,
    lastRenderAt: 0,
    parseErrors: 0,
    renderedTableSignature: '',
    renderedSelectedSignature: '',
    renderedPointSignature: ''
  };
  const elements = Object.fromEntries(ELEMENT_IDS.map((id) => [id, document.getElementById(id)]));

  elements.pause.addEventListener('click', () => {
    state.paused = !state.paused;
    elements.pause.textContent = state.paused ? '继续' : '暂停';
    updateConnection();
  });
  elements.clear.addEventListener('click', clearLocalTraces);
  elements.copy.addEventListener('click', copySnapshot);
  elements.export.addEventListener('click', exportSnapshot);
  for (const id of FILTER_IDS) {
    elements[id].addEventListener('input', renderTraceTable);
    elements[id].addEventListener('change', renderTraceTable);
  }
  window.addEventListener('resize', () => renderPointCanvas(selectedTransaction(), true));

  function scheduleStatusPoll(delay = STATUS_POLL_MS) {
    clearTimeout(state.statusTimer);
    state.statusTimer = window.setTimeout(pollStatus, delay);
  }

  async function pollStatus() {
    if (state.statusRequestActive) return scheduleStatusPoll();
    state.statusRequestActive = true;
    try {
      const response = await requestJSON('/api/hid/input-router/status');
      state.authorized = true;
      state.status = response.data || null;
      elements['auth-warning'].classList.add('hidden');
      renderAll();
    } catch (error) {
      if (error?.status === 401) {
        state.authorized = false;
        elements['auth-warning'].classList.remove('hidden');
      }
      updateConnection();
    } finally {
      state.statusRequestActive = false;
      scheduleStatusPoll();
    }
  }

  async function startTraceStream() {
    clearTimeout(state.streamReconnectTimer);
    const abort = new AbortController();
    state.streamAbort = abort;
    try {
      const response = await fetch('/api/hid/input-router/trace-stream', {
        credentials: 'same-origin',
        headers: { Accept: 'application/x-ndjson' },
        cache: 'no-store',
        signal: abort.signal
      });
      if (!response.ok) {
        const error = new Error(`HTTP ${response.status}`);
        error.status = response.status;
        throw error;
      }
      if (!response.body) throw new Error('trace stream unavailable');
      state.authorized = true;
      state.streamConnected = true;
      state.streamError = null;
      elements['auth-warning'].classList.add('hidden');
      updateConnection();
      await readTraceStream(response.body);
    } catch (error) {
      if (error?.name === 'AbortError') return;
      state.streamError = error;
      if (error?.status === 401) {
        state.authorized = false;
        elements['auth-warning'].classList.remove('hidden');
      }
    } finally {
      if (state.streamAbort !== abort) return;
      state.streamConnected = false;
      state.streamAbort = null;
      updateConnection();
      state.streamReconnectTimer = window.setTimeout(startTraceStream, STREAM_RECONNECT_MS);
    }
  }

  async function readTraceStream(body) {
    const reader = body.getReader();
    const decoder = new TextDecoder();
    let pending = '';
    while (true) {
      const { value, done } = await reader.read();
      pending += decoder.decode(value || new Uint8Array(), { stream: !done });
      const lines = pending.split('\n');
      pending = done ? '' : lines.pop() || '';
      const records = [];
      for (const line of lines) {
        if (!line.trim()) continue;
        try {
          records.push(JSON.parse(line));
        } catch {
          state.parseErrors += 1;
        }
      }
      if (records.length) {
        ingestTraces(records);
        scheduleRender();
      }
      if (done) return;
    }
  }

  function scheduleRender() {
    if (state.renderPending) return;
    state.renderPending = true;
    const delay = Math.max(0, RENDER_INTERVAL_MS - (performance.now() - state.lastRenderAt));
    state.renderTimer = window.setTimeout(() => {
      requestAnimationFrame(() => {
        state.renderPending = false;
        state.lastRenderAt = performance.now();
        renderAll();
      });
    }, delay);
  }

  async function requestJSON(url) {
    const response = await fetch(url, {
      credentials: 'same-origin',
      headers: { Accept: 'application/json' },
      cache: 'no-store'
    });
    if (!response.ok) {
      const error = new Error(`HTTP ${response.status}`);
      error.status = response.status;
      throw error;
    }
    const payload = await response.json();
    if (payload.code !== 0) throw new Error(payload.msg || `API ${payload.code}`);
    return payload;
  }

  function ingestTraces(records) {
    if (!Array.isArray(records) || state.paused) return;
    const latest = records.at(-1)?.sequence || 0;
    const knownLatest = state.traces.at(-1)?.sequence || 0;
    if (
      latest &&
      ((knownLatest && latest < knownLatest) ||
        (!knownLatest && state.ignoredUntil && latest < state.ignoredUntil))
    ) {
      resetLocalState();
    }
    const arrivedAt = Date.now();
    let changed = false;
    for (const record of records) {
      if (!record || !Number.isFinite(record.sequence) || record.sequence <= state.ignoredUntil) continue;
      if (state.seen.has(record.sequence)) continue;
      state.seen.add(record.sequence);
      state.arrivals.set(record.sequence, arrivedAt);
      state.traces.push(record);
      changed = true;
    }
    if (!changed) return;
    state.traces.sort((left, right) => left.sequence - right.sequence);
    if (state.traces.length > LOCAL_LIMIT) {
      const removed = state.traces.splice(0, state.traces.length - LOCAL_LIMIT);
      for (const record of removed) {
        state.seen.delete(record.sequence);
        state.arrivals.delete(record.sequence);
      }
    }
    state.transactionsDirty = true;
  }

  function resetLocalState() {
    state.traces = [];
    state.transactions = [];
    state.transactionsDirty = false;
    state.seen.clear();
    state.arrivals.clear();
    state.selectedKey = '';
    state.ignoredUntil = 0;
    state.autoFollow = true;
    state.renderedTableSignature = '';
    state.renderedSelectedSignature = '';
    state.renderedPointSignature = '';
  }

  function clearLocalTraces() {
    const latest = state.traces.at(-1)?.sequence || state.ignoredUntil;
    resetLocalState();
    state.ignoredUntil = latest;
    renderAll();
  }

  function rebuildTransactions() {
    const transactions = new Map();
    for (const trace of state.traces) {
      const captureSequence = Number(trace.capture_sequence || 0);
      const session = Number(trace.session_id || 0);
      const key = captureSequence
        ? `browser:${session}:${captureSequence}`
        : `router:${trace.sequence}`;
      let transaction = transactions.get(key);
      if (!transaction) {
        transaction = {
          key,
          captureSequence,
          session,
          firstSequence: trace.sequence,
          lastSequence: trace.sequence,
          timestamp: trace.t,
          input: null,
          outputs: [],
          records: []
        };
        transactions.set(key, transaction);
      }
      transaction.lastSequence = trace.sequence;
      transaction.timestamp = Math.min(transaction.timestamp || trace.t, trace.t || transaction.timestamp);
      transaction.records.push(trace);
      if (trace.browser_event) transaction.input = trace.browser_event;
      if (trace.stage !== 'browser-capture') transaction.outputs.push(trace);
    }
    state.transactions = [...transactions.values()]
      .sort((left, right) => left.lastSequence - right.lastSequence)
      .map((transaction) => {
        transaction.kinds = unique(transaction.outputs.map((output) => output.kind));
        transaction.routes = unique(transaction.outputs.map((output) => output.route));
        transaction.endpoints = unique(transaction.routes.map(endpointName));
        transaction.errors = unique(transaction.outputs.map((output) => output.error));
        transaction.search = JSON.stringify(serializeTransaction(transaction)).toLowerCase();
        return transaction;
      });
    state.transactionsVersion = (state.transactionsVersion || 0) + 1;
  }

  function ensureTransactions() {
    if (!state.transactionsDirty) return;
    rebuildTransactions();
    state.transactionsDirty = false;
    if (state.autoFollow || !transactionByKey(state.selectedKey)) {
      state.selectedKey = state.transactions.at(-1)?.key || '';
    }
  }

  function transactionByKey(key) {
    return state.transactions.find((transaction) => transaction.key === key) || null;
  }

  function selectedTransaction() {
    ensureTransactions();
    return transactionByKey(state.selectedKey) || state.transactions.at(-1) || null;
  }

  function renderAll() {
    ensureTransactions();
    updateConnection();
    renderStatus();
    updateFilterOptions();
    renderTraceTable();
    renderSelected(selectedTransaction());
    renderPerformance();
  }

  function updateConnection() {
    const node = elements.connection;
    node.className = 'status';
    if (!state.authorized) {
      node.classList.add('status-error');
      node.textContent = '需要登录';
    } else if (state.streamError) {
      node.classList.add('status-error');
      node.textContent = '连接失败';
    } else if (state.paused) {
      node.classList.add('status-paused');
      node.textContent = '已暂停';
    } else if (state.streamConnected) {
      node.classList.add('status-live');
      node.textContent = '实时';
    } else {
      node.classList.add('status-waiting');
      node.textContent = '正在连接';
    }
  }

  function renderStatus() {
    const runtime = state.status?.runtime;
    const signature = JSON.stringify(runtime ? {
      errors: [runtime.dispatch_failed || 0, runtime.backpressure_dropped || 0, totalRouteValue(runtime.routes, 'write_retries')],
      routes: runtime.routes || [],
      sessions: runtime.sessions || [],
      sources: runtime.sources || []
    } : null);
    if (state.renderedStatusSignature === signature) return;
    state.renderedStatus = state.status;
    state.renderedStatusSignature = signature;
    if (!runtime) {
      renderRoutes([]);
      renderSessions([]);
      renderSources([]);
      return;
    }
    const errors = [runtime.dispatch_failed || 0, runtime.backpressure_dropped || 0, totalRouteValue(runtime.routes, 'write_retries')];
    setText('metric-errors', errors.join(' / '));
    renderRoutes(runtime.routes || []);
    renderSessions(runtime.sessions || []);
    renderSources(runtime.sources || []);
  }

  function renderRoutes(routes) {
    if (!routes.length) {
      elements.routes.replaceChildren(create('div', 'route-card route-meta', '等待 Router 状态'));
      return;
    }
    elements.routes.replaceChildren(...routes.map((route) => {
      const card = create('article', 'route-card');
      card.append(create('div', 'route-name', `${route.route || route.kind || 'unknown'} → ${endpointName(route.route)}`));
      const meta = create('div', 'route-meta');
      for (const [label, value] of [
        ['OWNER', route.owner_session || '—'], ['OUTPUT', route.dispatched || 0],
        ['FAIL', route.failed || 0], ['DROP', route.backpressure_dropped || 0]
      ]) {
        const cell = create('div');
        cell.append(create('span', '', label), create('strong', '', value));
        meta.append(cell);
      }
      card.append(meta);
      return card;
    }));
  }

  function renderSessions(sessions) {
    if (!sessions.length) {
      elements.sessions.replaceChildren(create('div', 'session-card session-detail', '暂无活动输入会话'));
      return;
    }
    elements.sessions.replaceChildren(...sessions.map((session) => {
      const card = create('article', 'session-card');
      const title = create('div', 'session-title');
      title.append(create('span', '', `SESSION ${session.id}`), create('span', '', `${session.events} EVENTS`));
      const sources = (session.sources || []).map((source) => `${source.source}:${source.events}`).join(' · ') || '—';
      const routes = (session.routes || []).join(' · ') || '—';
      card.append(title, create('div', 'session-detail', `SOURCE SEQ ${session.last_source_sequence || 0}\n${sources}\n${routes}`));
      return card;
    }));
  }

  function renderSources(sources) {
    if (!sources.length) {
      elements.sources.replaceChildren(create('div', 'session-detail', '暂无 Router source 统计'));
      return;
    }
    const maximum = Math.max(1, ...sources.map((source) => source.events || 0));
    elements.sources.replaceChildren(...sources.map((source) => {
      const row = create('div', 'source-row');
      const bar = create('div', 'source-bar');
      const fill = create('span');
      fill.style.width = `${Math.max(1, (source.events || 0) / maximum * 100)}%`;
      bar.append(fill);
      row.append(
        create('span', 'source-name', `Router source · ${source.source}`),
        bar,
        create('span', 'source-count', source.events || 0)
      );
      return row;
    }));
  }

  function updateFilterOptions() {
    syncSelectOptions(elements['filter-api'], unique(state.transactions.map((transaction) => transaction.input?.api)));
    syncSelectOptions(elements['filter-event'], unique(state.transactions.map((transaction) => transaction.input?.type)));
    syncSelectOptions(elements['filter-pointer'], unique(state.transactions.map((transaction) => browserInputDevice(transaction.input))));
    syncSelectOptions(elements['filter-kind'], unique(state.transactions.flatMap((transaction) => transaction.kinds)));
    syncSelectOptions(elements['filter-output'], unique(state.transactions.flatMap((transaction) => transaction.endpoints)));
  }

  function syncSelectOptions(select, values) {
    const selected = select.value;
    const current = Array.from(select.options).slice(1).map((option) => option.value);
    if (current.join('\0') === values.join('\0')) return;
    select.replaceChildren(new Option('全部', ''), ...values.map((value) => new Option(value, value)));
    if (values.includes(selected)) select.value = selected;
  }

  function filteredTransactions() {
    ensureTransactions();
    const api = elements['filter-api'].value;
    const eventType = elements['filter-event'].value;
    const pointerType = elements['filter-pointer'].value;
    const kind = elements['filter-kind'].value;
    const output = elements['filter-output'].value;
    const session = elements['filter-session'].value.trim();
    const search = elements['filter-text'].value.trim().toLowerCase();
    return state.transactions.filter((transaction) => {
      if (api && transaction.input?.api !== api) return false;
      if (eventType && transaction.input?.type !== eventType) return false;
      if (pointerType && browserInputDevice(transaction.input) !== pointerType) return false;
      if (kind && !transaction.kinds.includes(kind)) return false;
      if (output && !transaction.endpoints.includes(output)) return false;
      if (session && String(transaction.session || '') !== session) return false;
      return !search || transaction.search.includes(search);
    });
  }

  function renderTraceTable() {
    const filtered = filteredTransactions();
    setText('visible-count', `${filtered.length} / ${state.transactions.length}`);
    const visible = filtered.slice(-TABLE_LIMIT).reverse();
    const filterSignature = FILTER_IDS.map((id) => elements[id].value).join('\0');
    const tableSignature = [
      state.transactionsVersion,
      state.selectedKey,
      filterSignature,
      visible.map((transaction) => `${transaction.key}:${transaction.lastSequence}`).join('\0')
    ].join('|');
    if (state.renderedTableSignature === tableSignature) return;
    state.renderedTableSignature = tableSignature;

    if (!visible.length) {
      const row = create('tr');
      const message = state.transactions.length ? '没有符合过滤条件的事务' : '等待浏览器输入或 Router 事件';
      const cell = create('td', 'empty-row', message);
      cell.colSpan = 10;
      row.append(cell);
      elements['trace-body'].replaceChildren(row);
      return;
    }

    const existingRows = new Map(
      [...elements['trace-body'].querySelectorAll('tr[data-key]')].map((row) => [row.dataset.key, row])
    );
    const rows = visible.map((transaction) => {
      const row = existingRows.get(transaction.key) || createTransactionRow();
      updateTransactionRow(row, transaction);
      return row;
    });
    for (let index = 0; index < rows.length; index += 1) {
      const current = elements['trace-body'].children[index];
      if (current !== rows[index]) {
        elements['trace-body'].insertBefore(rows[index], current || null);
      }
    }
    while (elements['trace-body'].children.length > rows.length) {
      elements['trace-body'].lastElementChild?.remove();
    }
  }

  function createTransactionRow() {
    const row = create('tr');
    row.addEventListener('click', () => {
      const transaction = transactionByKey(row.dataset.key || '');
      if (!transaction) return;
      state.selectedKey = transaction.key;
      state.autoFollow = transaction.key === state.transactions.at(-1)?.key;
      state.renderedTableSignature = '';
      state.renderedSelectedSignature = '';
      renderTraceTable();
      renderSelected(transaction, true);
    });
    return row;
  }

  function updateTransactionRow(row, transaction) {
    const classes = [];
    if (transaction.errors.length) classes.push('trace-error');
    if (transaction.input && !transaction.outputs.length) classes.push('capture-only');
    if (transaction.key === state.selectedKey) classes.push('selected');
    row.className = classes.join(' ');
    row.dataset.key = transaction.key;
    const input = transaction.input;
    const values = [
      transactionLabel(transaction),
      formatClock(transaction.timestamp),
      input ? `${input.api}.${input.type}` : '—',
      browserInputDevice(input) || unique(transaction.outputs.map((item) => item.source)).join(', ') || '—',
      input ? browserInputSummary(input) : fallbackInputSummary(transaction),
      transaction.kinds.join(', ') || '—',
      routeRelation(transaction),
      hidOutputSummary(transaction),
      resultSummary(transaction),
      latencySummary(transaction)
    ];
    const rowSignature = JSON.stringify(values);
    if (row.dataset.signature === rowSignature) return;
    row.dataset.signature = rowSignature;
    values.forEach((value, index) => {
      let cell = row.children[index];
      if (!cell) {
        cell = create('td');
        row.append(cell);
      }
      updateNodeText(cell, value);
      cell.title = displayValue(value);
    });
    while (row.children.length > values.length) row.lastElementChild?.remove();
  }

  function renderSelected(transaction, force = false) {
    const selectedSignature = selectedFlowSignature(transaction);
    if (!force && state.renderedSelectedSignature === selectedSignature) {
      renderPointCanvas(transaction);
      return;
    }
    state.renderedSelectedSignature = selectedSignature;
    const nodes = Object.fromEntries(
      [...document.querySelectorAll('.pipeline-node')].map((node) => [node.dataset.stage, node])
    );
    for (const node of Object.values(nodes)) node.classList.remove('active', 'error', 'pending');
    if (!transaction) {
      setText('selected-summary', '等待浏览器输入事件');
      for (const id of ['node-capture', 'node-mapping', 'node-route', 'node-encode', 'node-dispatch']) {
        elements[id].replaceChildren();
      }
      elements['router-flow-graph'].replaceChildren();
      setText('browser-event-json', '等待真实浏览器输入事件。');
      setText('raw-json', '等待选择事件。');
      renderPointCanvas(null);
      return;
    }
    const input = transaction.input;
    const outputs = transaction.outputs;
    const reports = outputs.filter((output) => output.report);
    const routed = outputs.filter((output) => output.route);
    const latest = outputs.at(-1);
    setText(
      'selected-summary',
      `${transactionLabel(transaction)} · ${formatClock(transaction.timestamp)} · SESSION ${transaction.session || '—'} · TRACE ${transaction.firstSequence}..${transaction.lastSequence}`
    );
    fillDefinitionList(
      elements['node-capture'],
      input ? browserInputDefinition(input) : [['API', '无关联浏览器输入事件', 'warn'], ['Trace', transaction.lastSequence]]
    );
    setText('browser-event-json', input ? JSON.stringify(input, null, 2) : '该事件没有关联的浏览器原始输入。');

    fillDefinitionList(elements['node-mapping'], outputs.length ? [
      ['Router Source', unique(outputs.map((output) => output.source)).join('\n') || '—'],
      ['Mode', unique(outputs.map((output) => output.mode)).join('\n') || '—'],
      ['Semantic Event', outputs.map((output) => `${output.kind || 'unknown'} · ${semanticSummary(output)}`).join('\n')],
      ['Client Processing', formatLatencyPair(outputs.map((output) => output.client_delay_us))]
    ] : [['State', input ? '未产生语义事件（过滤、合帧或等待）' : '—', 'warn']]);

    fillDefinitionList(elements['node-route'], routed.length ? [
      ['Relations', routed.map((output) => `${output.kind} → ${output.route}`).join('\n'), 'accent'],
      ['Session / Source Seq', routed.map((output) => `${output.session_id || '—'} / ${output.source_sequence || '—'}`).join('\n')],
      ['Owner', routed.map((output) => `${output.route}: ${routeOwner(output.route)}`).join('\n')],
      ['Router Stage', unique(routed.map((output) => output.stage)).join('\n')]
    ] : [['Route', '无 HID 路由', 'warn']]);
    renderRouterFlowGraph(transaction);

    fillDefinitionList(elements['node-encode'], reports.length ? [
      ['Reports', reports.map((output) => `${output.route} · ${reportIdentity(output)} · ${reportBytes(output)} B`).join('\n'), 'accent'],
      ['Hex', reports.map((output) => `${output.route}\n${output.report}`).join('\n')],
      ['Meaning', reports.map((output) => output.meaning || '—').join('\n')]
    ] : [['Report', routed.length ? '路由未生成 HID 报告' : '—', routed.length ? 'bad' : 'warn']]);

    fillDefinitionList(elements['node-dispatch'], outputs.length ? [
      ['Output', unique(outputs.map((output) => endpointName(output.route))).join('\n') || '—', 'accent'],
      ['Result', outputs.map((output) => `${output.stage || 'unknown'}${output.error ? ` · ${output.error}` : ' · OK'}`).join('\n'), transaction.errors.length ? 'bad' : 'good'],
      ['Router', formatLatencyPair(outputs.map((output) => output.router_duration_us))],
      ['HID Write', formatLatencyPair(outputs.map((output) => output.hid_write_us))],
      ['Server Time', formatTimestamp(latest?.t)]
    ] : [['Output', '未写入 hidgX', 'warn']]);

    nodes.capture.classList.add(input ? 'active' : 'pending');
    nodes.mapping.classList.add(outputs.length ? 'active' : 'pending');
    nodes.route.classList.add(routed.length ? 'active' : 'pending');
    nodes.encode.classList.add(reports.length ? 'active' : 'pending');
    nodes.dispatch.classList.add(transaction.errors.length ? 'error' : outputs.length ? 'active' : 'pending');
    setText('raw-json', JSON.stringify(serializeTransaction(transaction), null, 2));
    renderPointCanvas(transaction);
  }

  function selectedFlowSignature(transaction) {
    if (!transaction) return 'empty';
    const owners = transaction.outputs
      .map((output) => `${output.route || ''}:${routeOwner(output.route)}`)
      .join('|');
    return [
      transaction.key,
      transaction.lastSequence,
      transaction.outputs.length,
      owners
    ].join(':');
  }

  function fillDefinitionList(list, entries) {
    entries.forEach(([term, value, className], index) => {
      let row = list.children[index];
      if (!row) {
        row = create('div');
        row.append(create('dt'), create('dd'));
        list.append(row);
      }
      const termNode = row.querySelector('dt');
      const valueNode = row.querySelector('dd');
      updateNodeText(termNode, term);
      updateNodeText(valueNode, value);
      valueNode.className = className || '';
    });
    while (list.children.length > entries.length) list.lastElementChild?.remove();
  }

  function renderRouterFlowGraph(transaction) {
    if (!transaction.outputs.length) {
      const message = `${transaction.input?.api || 'Browser Event'} → ∅`;
      const current = elements['router-flow-graph'].firstElementChild;
      if (elements['router-flow-graph'].children.length !== 1 || !current?.classList.contains('router-flow-empty')) {
        elements['router-flow-graph'].replaceChildren(create('div', 'router-flow-empty', message));
      } else {
        updateNodeText(current, message);
      }
      return;
    }
    transaction.outputs.forEach((output, index) => {
      let row = elements['router-flow-graph'].children[index];
      if (!row?.classList.contains('router-flow-row')) {
        row = create('div');
        row.append(
          create('span', 'router-flow-event'),
          create('span', 'router-flow-arrow', '→'),
          create('span', 'router-flow-output')
        );
        elements['router-flow-graph'].insertBefore(row, elements['router-flow-graph'].children[index] || null);
      }
      row.className = `router-flow-row${output.error ? ' flow-error' : ''}`;
      updateNodeText(row.querySelector('.router-flow-event'), output.kind || 'decode');
      updateNodeText(
        row.querySelector('.router-flow-output'),
        output.route ? `${endpointName(output.route)} · ${output.route.split('/')[1]}` : output.stage || 'rejected'
      );
      row.title = output.error || `${output.kind || 'unknown'} → ${output.route || output.stage || 'unknown'}`;
    });
    while (elements['router-flow-graph'].children.length > transaction.outputs.length) {
      elements['router-flow-graph'].lastElementChild?.remove();
    }
  }

  function renderPointCanvas(transaction, force = false) {
    const canvas = elements['point-canvas'];
    const rect = canvas.getBoundingClientRect();
    const scale = Math.max(1, window.devicePixelRatio || 1);
    const width = rect.width;
    const height = rect.height;
    const selectedOutput = [...(transaction?.outputs || [])].reverse().find((output) => output.points?.length) || null;
    const relevant = selectedOutput ? state.traces
      .filter((trace) => trace.kind === selectedOutput.kind && trace.session_id === selectedOutput.session_id && trace.points?.length)
      .slice(-80) : [];
    const pointSignature = JSON.stringify({
      transaction: transaction?.key || '',
      output: selectedOutput?.sequence || 0,
      traces: relevant.map((trace) => trace.sequence),
      points: selectedOutput?.points || [],
      size: [width, height, scale]
    });
    if (!force && state.renderedPointSignature === pointSignature) return;
    state.renderedPointSignature = pointSignature;

    const pixelWidth = Math.max(1, Math.round(width * scale));
    const pixelHeight = Math.max(1, Math.round(height * scale));
    if (canvas.width !== pixelWidth) canvas.width = pixelWidth;
    if (canvas.height !== pixelHeight) canvas.height = pixelHeight;
    const context = canvas.getContext('2d');
    context.setTransform(scale, 0, 0, scale, 0, 0);
    context.clearRect(0, 0, width, height);
    context.strokeStyle = '#182637';
    context.lineWidth = 1;
    for (let step = 1; step < 10; step += 1) {
      const x = width * step / 10;
      const y = height * step / 10;
      context.beginPath(); context.moveTo(x, 0); context.lineTo(x, height); context.stroke();
      context.beginPath(); context.moveTo(0, y); context.lineTo(width, y); context.stroke();
    }
    context.strokeStyle = '#39516c';
    context.strokeRect(.5, .5, width - 1, height - 1);
    const trails = new Map();
    for (const output of relevant) {
      for (const point of output.points || []) {
        const id = Number(point.id || 0);
        if (!trails.has(id)) trails.set(id, []);
        trails.get(id).push(point);
      }
    }
    for (const [id, points] of trails) {
      const color = POINT_COLORS[Math.abs(id) % POINT_COLORS.length];
      context.strokeStyle = color;
      context.globalAlpha = .4;
      context.lineWidth = 2;
      context.beginPath();
      points.forEach((point, index) => {
        const x = clamp(point.x) * width;
        const y = clamp(point.y) * height;
        if (index) context.lineTo(x, y); else context.moveTo(x, y);
      });
      context.stroke();
    }
    context.globalAlpha = 1;
    const points = selectedOutput?.points || [];
    for (const point of points) {
      const id = Number(point.id || 0);
      const x = clamp(point.x) * width;
      const y = clamp(point.y) * height;
      const color = POINT_COLORS[Math.abs(id) % POINT_COLORS.length];
      context.fillStyle = `${color}55`;
      context.strokeStyle = color;
      context.lineWidth = 3;
      context.beginPath();
      context.arc(x, y, point.touching ? 15 : 11, 0, Math.PI * 2);
      context.fill();
      context.stroke();
      context.fillStyle = '#ffffff';
      context.font = 'bold 11px ui-monospace, monospace';
      context.textAlign = 'center';
      context.textBaseline = 'middle';
      context.fillText(id ? String(id) : 'P', x, y);
    }
    setText('point-count', `${points.length} 点`);
    if (!points.length) {
      const current = elements['point-legend'].firstElementChild;
      if (elements['point-legend'].children.length !== 1 || current?.dataset.empty !== 'true') {
        const empty = create('span', 'legend-item', '当前 HID 输出没有绝对坐标');
        empty.dataset.empty = 'true';
        elements['point-legend'].replaceChildren(empty);
      }
      return;
    }
    points.forEach((point, index) => {
      const id = Number(point.id || 0);
      let item = elements['point-legend'].children[index];
      if (!item || item.dataset.empty === 'true') {
        item = create('span', 'legend-item');
        item.append(create('span', 'legend-dot'), document.createTextNode(''));
        elements['point-legend'].insertBefore(item, elements['point-legend'].children[index] || null);
      }
      item.dataset.empty = 'false';
      const dot = item.querySelector('.legend-dot');
      dot.style.background = POINT_COLORS[Math.abs(id) % POINT_COLORS.length];
      const label = `#${id || 'P'} x=${number(point.x, 4)} y=${number(point.y, 4)} ${point.touching ? 'DOWN' : point.in_range ? 'HOVER' : 'UP'}`;
      if (item.lastChild?.nodeValue !== label) item.lastChild.nodeValue = label;
    });
    while (elements['point-legend'].children.length > points.length) {
      elements['point-legend'].lastElementChild?.remove();
    }
  }

  function renderPerformance() {
    const now = Date.now();
    const recent = state.traces.filter((trace) => now - (state.arrivals.get(trace.sequence) || 0) <= RATE_WINDOW_MS);
    const captures = recent.filter((trace) => trace.stage === 'browser-capture');
    const routerEvents = recent.filter((trace) => trace.stage !== 'browser-capture');
    const hidOutputs = routerEvents.filter(successfulHIDOutput);
    const linkedHIDOutputs = hidOutputs.filter((trace) => trace.capture_sequence);
    const localCaptures = state.traces.filter((trace) => trace.stage === 'browser-capture').length;
    const localRouter = state.traces.length - localCaptures;
    const localOutputs = state.traces.filter(successfulHIDOutput).length;
    const runtime = state.status?.runtime;
    const rateDivisor = RATE_WINDOW_MS / 1000;
    setText('metric-capture-rate', `${rate(captures.length, rateDivisor)}/s`);
    setText('metric-capture-total', `${localCaptures} 条 · 设备 ${runtime?.browser_captured ?? '—'}`);
    setText('metric-router-rate', `${rate(routerEvents.length, rateDivisor)}/s`);
    setText('metric-router-total', `${localRouter} 条 · 设备 ${runtime?.received ?? '—'}`);
    setText('metric-output-rate', `${rate(hidOutputs.length, rateDivisor)}/s`);
    setText('metric-output-total', `${localOutputs} 条 · 设备 ${runtime?.dispatched ?? '—'}`);
    setText('metric-routing-ratio', captures.length ? `${(linkedHIDOutputs.length / captures.length).toFixed(2)}×` : '—');
    setText('metric-client-latency', formatLatencyPair(routerEvents.map((trace) => trace.client_delay_us)));
    setText('metric-router-latency', formatLatencyPair(routerEvents.map((trace) => trace.router_duration_us)));
    setText('metric-hid-latency', formatLatencyPair(routerEvents.map((trace) => trace.hid_write_us)));

    const captureTransactions = state.transactions.filter((transaction) => transaction.input);
    const linkedTransactions = captureTransactions.filter((transaction) => transaction.outputs.length);
    const coalesced = captureTransactions.reduce((total, transaction) => total + Number(transaction.input?.content?.coalescedEventCount || 0), 0);
    const fanout = captureTransactions.length
      ? captureTransactions.reduce((total, transaction) => total + transaction.outputs.length, 0) / captureTransactions.length
      : 0;
    const performanceCards = [
      ['INPUT → ROUTER', `${linkedTransactions.length} / ${captureTransactions.length}`, `${percentage(linkedTransactions.length, captureTransactions.length)} 已关联`],
      ['ROUTER FAN-OUT', `${fanout.toFixed(2)}×`, '每个浏览器事件的语义事件数'],
      ['COALESCED', coalesced, 'PointerEvent 合并样本数']
    ];
    if (elements.performance.children.length !== performanceCards.length) {
      elements.performance.replaceChildren(...performanceCards.map(([label, value, detail]) => performanceCard(label, value, detail)));
    } else {
      [...elements.performance.children].forEach((card, index) => {
        const [label, value, detail] = performanceCards[index];
        updateNodeText(card.querySelector('span'), label);
        updateNodeText(card.querySelector('strong'), value);
        updateNodeText(card.querySelector('.session-detail'), detail);
      });
    }
    const health = [
      `Trace stream ${state.streamConnected ? 'CONNECTED' : 'OFFLINE'}`,
      `浏览器内存 ${state.traces.length}/${LOCAL_LIMIT} records · ${state.transactions.length} transactions`,
      `JSON parse errors ${state.parseErrors}`,
      `stream drops ${runtime?.dropped_stream_records ?? 0}`,
      `capture rejects ${runtime?.browser_capture_rejected ?? 0}`,
      runtime?.last_error ? `last error ${runtime.last_error}` : 'last error NONE'
    ];
    elements['debug-health'].textContent = health.join(' · ');
  }

  function performanceCard(label, value, detail) {
    const card = create('article', 'performance-card');
    card.append(create('span', '', label), create('strong', '', value), create('div', 'session-detail', detail));
    return card;
  }

  function updateNodeText(node, value) {
    if (node && node.textContent !== displayValue(value)) node.textContent = displayValue(value);
  }

  function dashboardSnapshot() {
    return {
      schema: 'nanokvm-input-debug-dashboard/v2',
      exportedAt: new Date().toISOString(),
      status: state.status,
      selectedKey: state.selectedKey,
      filters: Object.fromEntries(FILTER_IDS.map((id) => [id.replace('filter-', ''), elements[id].value])),
      transactions: filteredTransactions().map(serializeTransaction)
    };
  }

  async function copySnapshot() {
    try {
      await navigator.clipboard.writeText(`${JSON.stringify(dashboardSnapshot(), null, 2)}\n`);
      flashButton(elements.copy, '已复制');
    } catch {
      flashButton(elements.copy, '复制失败');
    }
  }

  function exportSnapshot() {
    const blob = new Blob([`${JSON.stringify(dashboardSnapshot(), null, 2)}\n`], { type: 'application/json' });
    const link = document.createElement('a');
    const url = URL.createObjectURL(blob);
    link.href = url;
    link.download = `nanokvm-input-debug-${Date.now()}.json`;
    link.click();
    window.setTimeout(() => URL.revokeObjectURL(url), 0);
    flashButton(elements.export, '已导出');
  }

  function serializeTransaction(transaction) {
    return {
      transaction: {
        key: transaction.key,
        pointerEventSequence: transaction.captureSequence || null,
        traceSequences: transaction.records.map((record) => record.sequence),
        sessionId: transaction.session || null,
        timestamp: transaction.timestamp
      },
      input: transaction.input,
      router: transaction.outputs
    };
  }

  function transactionLabel(transaction) {
    return transaction.captureSequence ? `P#${transaction.captureSequence}` : `T#${transaction.lastSequence}`;
  }

  function browserInputDevice(input) {
    if (!input) return '';
    return input.api === 'TouchEvent' ? 'touch' : input.content?.pointerType || 'pointer';
  }

  function browserInputDefinition(input) {
    const content = input.content || {};
    if (input.api === 'TouchEvent') {
      const active = content.targetTouches || [];
      const changed = content.changedTouches || [];
      return [
        ['API', input.api, 'accent'],
        ['Type', input.type, 'accent'],
        ['Contacts', `${active.length} active / ${changed.length} changed`],
        ['Active Snapshot', touchPointsSummary(active)],
        ['Changed', touchPointsSummary(changed)],
        ['Target', content.targetId || '—']
      ];
    }
    return [
      ['API', input.api, 'accent'],
      ['Type', input.type, 'accent'],
      ['Pointer', `${content.pointerType} #${content.pointerId}${content.isPrimary ? ' primary' : ''}`],
      ['Client', coordinate(content.clientX, content.clientY)],
      ['Movement', coordinate(content.movementX, content.movementY)],
      ['Buttons / Pressure', `button=${content.button} buttons=${hexNumber(content.buttons)} pressure=${number(content.pressure, 3)}`],
      ['Coalesced', content.coalescedEventCount || 0]
    ];
  }

  function browserInputSummary(input) {
    if (input.api === 'TouchEvent') {
      const active = input.content?.targetTouches || [];
      const changed = input.content?.changedTouches || [];
      return `${active.length} active · ${touchPointsSummary(active)} · ${changed.length} changed`;
    }
    return pointerSummary(input.content);
  }

  function pointerSummary(content = {}) {
    const coalesced = content.coalescedEventCount ? ` coal=${content.coalescedEventCount}` : '';
    return `#${content.pointerId ?? '—'} client=${coordinate(content.clientX, content.clientY)} Δ=${coordinate(content.movementX, content.movementY)} buttons=${hexNumber(content.buttons)} p=${number(content.pressure, 3)}${coalesced}`;
  }

  function touchPointsSummary(points = []) {
    return points.map((point) => `#${point.identifier} ${coordinate(point.clientX, point.clientY)}`).join(' · ') || '—';
  }

  function fallbackInputSummary(transaction) {
    const latest = transaction.outputs.at(-1);
    return latest ? semanticSummary(latest) : '—';
  }

  function semanticSummary(output) {
    const semantic = output.semantic || {};
    if (output.points?.length) {
      return output.points.map((point) => `#${point.id || 'P'} (${number(point.x, 4)},${number(point.y, 4)}) ${point.touching ? 'DOWN' : point.in_range ? 'HOVER' : 'UP'}`).join(' · ');
    }
    if (output.kind === 'relative-pointer') {
      return `buttons=${hexNumber(semantic.buttons)} Δ=(${semantic.dx || 0},${semantic.dy || 0}) wheel=${semantic.wheel || 0}`;
    }
    if (output.kind === 'keyboard-state') {
      return `modifiers=${hexNumber(semantic.modifiers)} keys=[${(semantic.keys || []).join(',')}]`;
    }
    return compactJSON(semantic);
  }

  function routeRelation(transaction) {
    const relations = unique(transaction.outputs
      .filter((output) => output.kind || output.route)
      .map((output) => `${output.kind || 'unknown'} → ${output.route || 'rejected'}`));
    return relations.join(' | ') || (transaction.input ? `${transaction.input.api} → ∅` : '—');
  }

  function hidOutputSummary(transaction) {
    const reports = transaction.outputs.filter((output) => output.route && output.report);
    return unique(reports.map((output) => `${endpointName(output.route)} ${reportIdentity(output)} ${reportBytes(output)}B`)).join(' | ') || '—';
  }

  function resultSummary(transaction) {
    if (transaction.errors.length) return transaction.errors.join(' | ');
    if (!transaction.outputs.length) return '无 HID（过滤 / 合帧）';
    return unique(transaction.outputs.map((output) => output.stage)).join(' | ') || '—';
  }

  function latencySummary(transaction) {
    if (!transaction.outputs.length) return '—';
    const client = formatLatency(maximum(transaction.outputs.map((output) => output.client_delay_us)));
    const router = formatLatency(maximum(transaction.outputs.map((output) => output.router_duration_us)));
    const hid = formatLatency(maximum(transaction.outputs.map((output) => output.hid_write_us)));
    return `Client ${client} · Router ${router} · HID ${hid}`;
  }

  function endpointName(route) {
    return String(route || '').split('/')[0] || '—';
  }

  function reportIdentity(output) {
    if (!output?.report) return '—';
    return endpointName(output.route) === 'hidg2' ? `ID 0x${output.report.slice(0, 2).toUpperCase()}` : 'NO-ID';
  }

  function reportBytes(output) {
    return output?.report ? Math.floor(output.report.length / 2) : 0;
  }

  function routeOwner(route) {
    const match = state.status?.runtime?.routes?.find((item) => item.route === route);
    return match?.owner_session || '—';
  }

  function successfulHIDOutput(trace) {
    return Boolean(trace.route && trace.report && (trace.stage === 'dispatched' || trace.stage === 'dispatched-retry'));
  }

  function totalRouteValue(routes = [], key) {
    return routes.reduce((total, route) => total + Number(route[key] || 0), 0);
  }

  function formatLatencyPair(values) {
    const finite = values.map(Number).filter((value) => Number.isFinite(value) && value >= 0);
    if (!finite.length) return '—';
    return `${formatLatency(percentile(finite, .5))} / ${formatLatency(percentile(finite, .95))}`;
  }

  function percentile(values, quantile) {
    const ordered = [...values].sort((left, right) => left - right);
    return ordered[Math.max(0, Math.ceil(ordered.length * quantile) - 1)];
  }

  function maximum(values) {
    const finite = values.map(Number).filter((value) => Number.isFinite(value) && value >= 0);
    return finite.length ? Math.max(...finite) : null;
  }

  function formatLatency(microseconds) {
    if (!Number.isFinite(microseconds)) return '—';
    if (microseconds < 1000) return `${Math.round(microseconds)}µs`;
    if (microseconds < 1000000) return `${(microseconds / 1000).toFixed(microseconds < 10000 ? 2 : 1)}ms`;
    return `${(microseconds / 1000000).toFixed(2)}s`;
  }

  function rate(count, seconds) {
    const value = count / seconds;
    return value >= 100 ? value.toFixed(0) : value.toFixed(1);
  }

  function percentage(value, total) {
    return total ? `${(value / total * 100).toFixed(1)}%` : '—';
  }

  function unique(values) {
    return [...new Set(values.filter((value) => value !== undefined && value !== null && value !== ''))].sort();
  }

  function create(tag, className = '', textContent) {
    const node = document.createElement(tag);
    if (className) node.className = className;
    if (textContent !== undefined) node.textContent = String(textContent);
    return node;
  }

  function setText(id, value) {
    const textContent = displayValue(value);
    if (elements[id].textContent !== textContent) elements[id].textContent = textContent;
  }

  function displayValue(value) {
    if (value === undefined || value === null || value === '') return '—';
    return String(value);
  }

  function compactJSON(value) {
    const json = JSON.stringify(value);
    return json === '{}' || json === '[]' ? '—' : json;
  }

  function coordinate(x, y) {
    return `(${number(x, 2)}, ${number(y, 2)})`;
  }

  function hexNumber(value) {
    const numberValue = Number(value || 0);
    return `0x${numberValue.toString(16).padStart(2, '0').toUpperCase()}`;
  }

  function formatClock(timestamp) {
    return timestamp ? new Date(timestamp).toLocaleTimeString([], {
      hour: '2-digit', minute: '2-digit', second: '2-digit', fractionalSecondDigits: 3, hour12: false
    }) : '—';
  }

  function formatTimestamp(timestamp) {
    return timestamp ? new Date(timestamp).toISOString() : '—';
  }

  function number(value, digits = 3) {
    const numeric = Number(value);
    return Number.isFinite(numeric) ? numeric.toFixed(digits) : '—';
  }

  function clamp(value) {
    return Math.max(0, Math.min(1, Number(value) || 0));
  }

  function flashButton(button, label) {
    const previous = button.textContent;
    button.textContent = label;
    window.setTimeout(() => { button.textContent = previous; }, 900);
  }

  window.inputDebugDashboard = { snapshot: dashboardSnapshot, clear: clearLocalTraces };
  window.addEventListener('pagehide', () => {
    clearTimeout(state.statusTimer);
    clearTimeout(state.streamReconnectTimer);
    clearTimeout(state.renderTimer);
    state.streamAbort?.abort();
  });
  renderAll();
  pollStatus();
  startTraceStream();
})();
