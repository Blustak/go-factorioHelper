(() => {
  const state = {
    recipes: [],
    items: [],
    fluids: [],
    machines: [],
    graph: { nodes: [], edges: [] },
    issues: [],
    selected: null,
    seq: 1,
    draggingNode: null,
    wiring: null,
    validateTimer: 0,
  };

  const recipeList = document.getElementById("recipe-list");
  const recipeSearch = document.getElementById("recipe-search");
  const nodesEl = document.getElementById("nodes");
  const wiresEl = document.getElementById("wires");
  const canvas = document.getElementById("canvas");
  const issuesEl = document.getElementById("issues");

  function nextId(prefix) {
    let id;
    do {
      id = prefix + state.seq++;
    } while (state.graph.nodes.some((n) => n.id === id) || state.graph.edges.some((e) => e.id === id));
    return id;
  }

  function recipeByName(name) {
    return state.recipes.find((r) => r.name === name);
  }

  function machinesFor(category) {
    if (!category) return state.machines.slice();
    return state.machines.filter((m) => (m.crafting_categories || []).includes(category));
  }

  function commodities() {
    return state.items.concat(state.fluids);
  }

  function labelOf(obj) {
    if (!obj) return "";
    return obj.localised_name || obj.name || "";
  }

  function recipeLabel(name) {
    return labelOf(recipeByName(name)) || name || "";
  }

  function commodityLabel(name, type) {
    const c = commodities().find((x) => x.name === name && x.type === (type || "item"));
    return labelOf(c) || name || "";
  }

  function matchesQuery(r, q) {
    if (!q) return true;
    const name = (r.name || "").toLowerCase();
    const display = (r.localised_name || "").toLowerCase();
    return name.includes(q) || display.includes(q);
  }

  function portsFor(node) {
    if (node.kind === "recipe") {
      const r = recipeByName(node.recipe);
      if (!r) return [];
      const ins = (r.ingredients || []).map((c, i) => ({
        id: "in:" + i,
        dir: "in",
        name: c.name,
        type: c.type || "item",
        label: labelOf(c),
        required: true,
      }));
      const outs = (r.products || []).map((c, i) => ({
        id: "out:" + i,
        dir: "out",
        name: c.name,
        type: c.type || "item",
        label: labelOf(c),
        required: false,
      }));
      return ins.concat(outs);
    }
    const type = node.prototype_type || "item";
    const itemLabel = commodityLabel(node.item_name, type);
    if (node.kind === "source") {
      return [{ id: "out:0", dir: "out", name: node.item_name, type, label: itemLabel, required: false }];
    }
    return [{ id: "in:0", dir: "in", name: node.item_name, type, label: itemLabel, required: true }];
  }

  function pruneEdges() {
    const valid = new Set();
    for (const n of state.graph.nodes) {
      for (const p of portsFor(n)) {
        valid.add(n.id + "/" + p.id);
      }
    }
    state.graph.edges = state.graph.edges.filter((e) =>
      valid.has(e.from_node + "/" + e.from_port) && valid.has(e.to_node + "/" + e.to_port)
    );
  }

  function outgoingCount(nodeId, portId) {
    return state.graph.edges.filter((e) => e.from_node === nodeId && e.from_port === portId).length;
  }

  function issueOn(nodeId, portId, edgeId) {
    return state.issues.find((i) =>
      (nodeId && i.node_id === nodeId && (!portId || i.port_id === portId)) ||
      (edgeId && i.edge_id === edgeId)
    );
  }

  async function loadCatalog() {
    const [recipes, items, fluids, machines] = await Promise.all([
      fetchJSON("/api/recipes"),
      fetchJSON("/api/items"),
      fetchJSON("/api/fluids"),
      fetchJSON("/api/machines"),
    ]);
    state.recipes = recipes || [];
    state.items = items || [];
    state.fluids = fluids || [];
    state.machines = machines || [];
  }

  async function fetchJSON(url, opts) {
    const res = await fetch(url, opts);
    if (!res.ok) {
      throw new Error(url + " " + res.status);
    }
    return res.json();
  }

  function scheduleValidate() {
    clearTimeout(state.validateTimer);
    state.validateTimer = setTimeout(validate, 120);
  }

  async function validate() {
    try {
      const res = await fetchJSON("/api/graph/validate", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(state.graph),
      });
      state.issues = res.issues || [];
      renderIssues(res.ok);
      paintValidation();
      drawWires();
    } catch (err) {
      issuesEl.replaceChildren(el("div", { className: "err", text: String(err) }));
    }
  }

  function renderIssues(ok) {
    issuesEl.replaceChildren();
    if (ok) {
      issuesEl.append(el("div", { className: "ok", text: "Graph is valid." }));
      return;
    }
    for (const issue of state.issues) {
      issuesEl.append(el("div", { className: "err", text: issue.message }));
    }
  }

  function renderSidebar() {
    const q = (recipeSearch.value || "").toLowerCase();
    recipeList.replaceChildren();
    for (const r of state.recipes) {
      if (!matchesQuery(r, q)) continue;
      const btn = el("button", { type: "button", text: labelOf(r) });
      btn.addEventListener("click", () => addRecipe(r.name));
      recipeList.append(el("li", {}, btn));
    }
  }

  function addRecipe(name) {
    const r = recipeByName(name);
    const machines = r ? machinesFor(r.category) : [];
    addNode({
      kind: "recipe",
      recipe: name,
      machine: machines[0] ? machines[0].name : "",
    });
  }

  function addIO(kind) {
    const first = commodities()[0];
    addNode({
      kind,
      item_name: first ? first.name : "",
      prototype_type: first ? first.type : "item",
    });
  }

  function addNode(partial) {
    const n = Object.assign({
      id: nextId("n"),
      x: 48 + (state.graph.nodes.length % 8) * 28,
      y: 48 + (state.graph.nodes.length % 8) * 20,
    }, partial);
    state.graph.nodes.push(n);
    state.selected = n.id;
    render();
    scheduleValidate();
  }

  function render() {
    renderNodes();
    requestAnimationFrame(() => {
      drawWires();
      paintValidation();
    });
  }

  function renderNodes() {
    nodesEl.replaceChildren();
    for (const node of state.graph.nodes) {
      nodesEl.append(renderNode(node));
    }
  }

  function renderNode(node) {
    const card = el("article", {
      className: "node " + node.kind + (state.selected === node.id ? " selected" : ""),
      dataset: { id: node.id },
    });
    card.style.left = node.x + "px";
    card.style.top = node.y + "px";

    const header = el("header", {},
      el("span", { text: nodeTitle(node) }),
      el("span", { className: "kind", text: node.kind }),
    );
    header.addEventListener("pointerdown", (ev) => startNodeDrag(ev, node));
    card.append(header);

    const body = el("div", { className: "body" });
    if (node.kind === "recipe") {
      body.append(machineSelect(node));
    } else {
      body.append(itemSelect(node));
    }
    body.append(portColumns(node));
    card.append(body);
    card.addEventListener("mousedown", () => {
      state.selected = node.id;
      syncSelection();
    });
    return card;
  }

  function nodeTitle(node) {
    if (node.kind === "recipe") return recipeLabel(node.recipe) || "recipe";
    return commodityLabel(node.item_name, node.prototype_type) || node.item_name || node.kind;
  }

  function machineSelect(node) {
    const r = recipeByName(node.recipe);
    const wrap = el("label", { className: "field", text: "Machine" });
    const sel = el("select");
    sel.append(el("option", { value: "", text: "(none)" }));
    for (const m of machinesFor(r ? r.category : "")) {
      const opt = el("option", { value: m.name, text: labelOf(m) });
      if (m.name === node.machine) opt.selected = true;
      sel.append(opt);
    }
    sel.addEventListener("change", () => {
      node.machine = sel.value;
      scheduleValidate();
    });
    wrap.append(sel);
    return wrap;
  }

  function itemSelect(node) {
    const wrap = el("label", { className: "field", text: "Item" });
    const sel = el("select");
    for (const c of commodities()) {
      const value = c.type + ":" + c.name;
      const opt = el("option", { value, text: labelOf(c) + " (" + c.type + ")" });
      if (c.name === node.item_name && c.type === (node.prototype_type || "item")) {
        opt.selected = true;
      }
      sel.append(opt);
    }
    sel.addEventListener("change", () => {
      const [type, ...rest] = sel.value.split(":");
      node.prototype_type = type;
      node.item_name = rest.join(":");
      pruneEdges();
      render();
      scheduleValidate();
    });
    wrap.append(sel);
    return wrap;
  }

  function portColumns(node) {
    const ports = portsFor(node);
    const wrap = el("div", { className: "ports" });
    const ins = el("div", { className: "col" });
    const outs = el("div", { className: "col" });
    for (const p of ports) {
      const row = el("div", {
        className: "port " + p.dir,
        dataset: { node: node.id, port: p.id, dir: p.dir },
      });
      const dot = el("span", { className: "dot" });
      const label = el("span", { text: p.label || p.name || p.id });
      row.append(dot, label);
      if (p.dir === "out" && outgoingCount(node.id, p.id) === 0) {
        row.classList.add("waste");
        row.title = "waste";
      }
      row.addEventListener("pointerdown", (ev) => startWire(ev, node, p));
      (p.dir === "in" ? ins : outs).append(row);
    }
    wrap.append(ins, outs);
    return wrap;
  }

  function syncSelection() {
    for (const card of nodesEl.querySelectorAll(".node")) {
      card.classList.toggle("selected", card.dataset.id === state.selected);
    }
  }

  function paintValidation() {
    for (const card of nodesEl.querySelectorAll(".node")) {
      const nodeId = card.dataset.id;
      card.classList.toggle("error", Boolean(issueOn(nodeId)));
      for (const row of card.querySelectorAll(".port")) {
        row.classList.toggle("error", Boolean(issueOn(nodeId, row.dataset.port)));
      }
    }
  }

  function canvasPoint(ev) {
    const rect = canvas.getBoundingClientRect();
    return {
      x: ev.clientX - rect.left + canvas.scrollLeft,
      y: ev.clientY - rect.top + canvas.scrollTop,
    };
  }

  function startNodeDrag(ev, node) {
    if (ev.button !== 0) return;
    ev.preventDefault();
    state.selected = node.id;
    syncSelection();
    const origin = canvasPoint(ev);
    state.draggingNode = {
      id: node.id,
      dx: origin.x - node.x,
      dy: origin.y - node.y,
    };
    headerCursor(ev.currentTarget, "grabbing");
  }

  function headerCursor(header, value) {
    header.style.cursor = value;
  }

  function startWire(ev, node, port) {
    if (ev.button !== 0) return;
    ev.preventDefault();
    ev.stopPropagation();
    const pt = canvasPoint(ev);
    state.wiring = { node, port, x: pt.x, y: pt.y };
    drawWires();
  }

  function portCenter(nodeId, portId) {
    const row = nodesEl.querySelector('.port[data-node="' + cssEscape(nodeId) + '"][data-port="' + cssEscape(portId) + '"] .dot');
    if (!row) return null;
    const canvasRect = canvas.getBoundingClientRect();
    const r = row.getBoundingClientRect();
    return {
      x: r.left - canvasRect.left + canvas.scrollLeft + r.width / 2,
      y: r.top - canvasRect.top + canvas.scrollTop + r.height / 2,
    };
  }

  function cssEscape(value) {
    if (window.CSS && CSS.escape) return CSS.escape(value);
    return String(value).replace(/"/g, '\\"');
  }

  function drawWires() {
    const ns = "http://www.w3.org/2000/svg";
    while (wiresEl.firstChild) wiresEl.removeChild(wiresEl.firstChild);
    for (const edge of state.graph.edges) {
      const a = portCenter(edge.from_node, edge.from_port);
      const b = portCenter(edge.to_node, edge.to_port);
      if (!a || !b) continue;
      const path = document.createElementNS(ns, "path");
      path.setAttribute("d", bezier(a, b));
      if (issueOn(null, null, edge.id)) path.classList.add("error");
      path.addEventListener("click", (ev) => {
        ev.stopPropagation();
        state.graph.edges = state.graph.edges.filter((e) => e.id !== edge.id);
        render();
        scheduleValidate();
      });
      path.style.pointerEvents = "stroke";
      wiresEl.append(path);
    }
    if (state.wiring) {
      const a = portCenter(state.wiring.node.id, state.wiring.port.id);
      if (a) {
        const b = { x: state.wiring.x, y: state.wiring.y };
        const path = document.createElementNS(ns, "path");
        path.setAttribute("d", state.wiring.port.dir === "out" ? bezier(a, b) : bezier(b, a));
        path.classList.add("preview");
        wiresEl.append(path);
      }
    }
  }

  function bezier(a, b) {
    const dx = Math.max(40, Math.abs(b.x - a.x) * 0.5);
    return "M " + a.x + " " + a.y + " C " + (a.x + dx) + " " + a.y + ", " + (b.x - dx) + " " + b.y + ", " + b.x + " " + b.y;
  }

  function finishWire(ev) {
    const wiring = state.wiring;
    if (!wiring) return;
    state.wiring = null;
    const target = ev.target.closest && ev.target.closest(".port");
    if (!target) {
      drawWires();
      return;
    }
    const from = wiring.port.dir === "out" ? wiring : {
      node: { id: target.dataset.node },
      port: { id: target.dataset.port, dir: target.dataset.dir },
    };
    const to = wiring.port.dir === "in" ? wiring : {
      node: { id: target.dataset.node },
      port: { id: target.dataset.port, dir: target.dataset.dir },
    };
    if (from.port.dir !== "out" || to.port.dir !== "in") {
      drawWires();
      return;
    }
    const exists = state.graph.edges.some((e) =>
      e.from_node === from.node.id && e.from_port === from.port.id &&
      e.to_node === to.node.id && e.to_port === to.port.id
    );
    if (!exists) {
      state.graph.edges.push({
        id: nextId("e"),
        from_node: from.node.id,
        from_port: from.port.id,
        to_node: to.node.id,
        to_port: to.port.id,
      });
    }
    render();
    scheduleValidate();
  }

  function el(tag, attrs, ...children) {
    const node = document.createElement(tag);
    if (attrs) {
      for (const [k, v] of Object.entries(attrs)) {
        if (k === "text") node.textContent = v;
        else if (k === "dataset") Object.assign(node.dataset, v);
        else if (k === "className") node.className = v;
        else node.setAttribute(k, v);
      }
    }
    for (const child of children) {
      if (child) node.append(child);
    }
    return node;
  }

  function deleteSelected() {
    if (!state.selected) return;
    const id = state.selected;
    state.graph.nodes = state.graph.nodes.filter((n) => n.id !== id);
    state.graph.edges = state.graph.edges.filter((e) => e.from_node !== id && e.to_node !== id);
    state.selected = null;
    render();
    scheduleValidate();
  }

  document.getElementById("add-source").addEventListener("click", () => addIO("source"));
  document.getElementById("add-sink").addEventListener("click", () => addIO("sink"));
  recipeSearch.addEventListener("input", renderSidebar);
  document.getElementById("download").addEventListener("click", () => {
    const blob = new Blob([JSON.stringify(state.graph, null, 2)], { type: "application/json" });
    const a = el("a", {
      href: URL.createObjectURL(blob),
      download: "supply-chain.json",
    });
    a.click();
    URL.revokeObjectURL(a.href);
  });
  document.getElementById("upload").addEventListener("change", async (ev) => {
    const file = ev.target.files && ev.target.files[0];
    if (!file) return;
    try {
      const parsed = JSON.parse(await file.text());
      state.graph = {
        nodes: parsed.nodes || [],
        edges: parsed.edges || [],
      };
      state.seq = 1;
      pruneEdges();
      render();
      scheduleValidate();
    } catch (err) {
      issuesEl.replaceChildren(el("div", { className: "err", text: String(err) }));
    }
    ev.target.value = "";
  });

  window.addEventListener("pointermove", (ev) => {
    if (state.draggingNode) {
      const node = state.graph.nodes.find((n) => n.id === state.draggingNode.id);
      if (!node) return;
      const pt = canvasPoint(ev);
      node.x = Math.max(0, pt.x - state.draggingNode.dx);
      node.y = Math.max(0, pt.y - state.draggingNode.dy);
      const card = nodesEl.querySelector('.node[data-id="' + cssEscape(node.id) + '"]');
      if (card) {
        card.style.left = node.x + "px";
        card.style.top = node.y + "px";
      }
      drawWires();
    }
    if (state.wiring) {
      const pt = canvasPoint(ev);
      state.wiring.x = pt.x;
      state.wiring.y = pt.y;
      drawWires();
    }
  });
  window.addEventListener("pointerup", (ev) => {
    if (state.draggingNode) {
      state.draggingNode = null;
      for (const h of nodesEl.querySelectorAll("header")) h.style.cursor = "";
    }
    if (state.wiring) finishWire(ev);
  });
  window.addEventListener("keydown", (ev) => {
    const tag = (ev.target && ev.target.tagName) || "";
    if (tag === "INPUT" || tag === "SELECT" || tag === "TEXTAREA") return;
    if (ev.key === "Delete" || ev.key === "Backspace") {
      ev.preventDefault();
      deleteSelected();
    }
    if (ev.key === "Escape") {
      state.wiring = null;
      drawWires();
    }
  });
  canvas.addEventListener("mousedown", (ev) => {
    if (ev.target === canvas || ev.target === nodesEl || ev.target === wiresEl) {
      state.selected = null;
      syncSelection();
    }
  });

  loadCatalog()
    .then(() => {
      renderSidebar();
      render();
      scheduleValidate();
    })
    .catch((err) => {
      issuesEl.replaceChildren(el("div", { className: "err", text: String(err) }));
    });
})();
