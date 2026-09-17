package graphview

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"os"
)

const htmlPageTemplate = `<!DOCTYPE html>
<html lang="pt-BR">
<head>
  <meta charset="UTF-8">
  <meta http-equiv="Content-Type" content="text/html; charset=UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>{{ .Title }}</title>
  <style>
    :root {
      --bg: #090d16;
      --bg-surface: #0f172a;
      --border: #1e293b;
      --text: #f8fafc;
      --text-muted: #94a3b8;
      --accent: #38bdf8;
      --accent-glow: rgba(56, 189, 248, 0.25);
    }
    * { box-sizing: border-box; margin: 0; padding: 0; }
    body {
      background-color: var(--bg);
      color: var(--text);
      font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif;
      overflow: hidden;
      width: 100vw;
      height: 100vh;
      user-select: none;
    }
    header {
      position: absolute;
      top: 16px;
      left: 16px;
      z-index: 10;
      background: rgba(15, 23, 42, 0.85);
      backdrop-filter: blur(8px);
      border: 1px solid var(--border);
      border-radius: 12px;
      padding: 12px 18px;
      display: flex;
      flex-direction: column;
      gap: 8px;
      max-width: 440px;
      box-shadow: 0 10px 25px -5px rgba(0, 0, 0, 0.5);
    }
    .title-row { display: flex; align-items: center; justify-content: space-between; gap: 12px; }
    .title { font-size: 16px; font-weight: 700; color: var(--text); }
    .badge {
      font-size: 11px;
      padding: 2px 8px;
      border-radius: 9999px;
      background: #1e293b;
      color: var(--accent);
      border: 1px solid rgba(56, 189, 248, 0.3);
    }
    .stats-row {
      display: flex;
      gap: 12px;
      font-size: 12px;
      color: var(--text-muted);
    }
    .stats-row span strong { color: var(--text); }
    .search-row {
      display: flex;
      gap: 8px;
      margin-top: 4px;
    }
    .search-input {
      flex: 1;
      background: #090d16;
      border: 1px solid var(--border);
      border-radius: 8px;
      padding: 6px 12px;
      color: var(--text);
      font-size: 13px;
      outline: none;
      transition: border-color 0.2s;
    }
    .search-input:focus { border-color: var(--accent); }
    .filter-chips {
      display: flex;
      flex-wrap: wrap;
      gap: 6px;
      margin-top: 2px;
    }
    .chip {
      font-size: 11px;
      padding: 3px 8px;
      border-radius: 6px;
      background: #1e293b;
      color: var(--text-muted);
      cursor: pointer;
      transition: all 0.2s;
      border: 1px solid transparent;
    }
    .chip.active {
      color: var(--text);
      font-weight: 600;
      border-color: currentColor;
    }
    #canvas-container {
      width: 100vw;
      height: 100vh;
      cursor: grab;
    }
    #canvas-container.panning { cursor: grabbing; }
    svg { width: 100%; height: 100%; }
    .edge {
      stroke-opacity: 0.5;
      stroke-width: 1.5;
      transition: stroke-opacity 0.2s, stroke-width 0.2s;
    }
    .edge.inferred { stroke-dasharray: 4, 4; }
    .edge.highlighted { stroke-opacity: 1 !important; stroke-width: 3 !important; }
    .node-group { cursor: pointer; transition: opacity 0.2s; }
    .node-circle {
      stroke: #090d16;
      stroke-width: 2px;
      transition: r 0.2s, stroke-width 0.2s, filter 0.2s;
    }
    .node-group:hover .node-circle {
      stroke-width: 3.5px;
      stroke: #fff;
      filter: drop-shadow(0 0 8px var(--accent));
    }
    .node-group.hub .node-circle {
      stroke: #eab308;
      stroke-width: 2.5px;
    }
    .node-label {
      font-size: 11px;
      font-weight: 500;
      fill: #cbd5e1;
      text-anchor: middle;
      pointer-events: none;
      paint-order: stroke;
      stroke: #090d16;
      stroke-width: 3px;
      stroke-linecap: round;
      stroke-linejoin: round;
      opacity: 0;
      transition: opacity 0.2s ease, fill 0.2s ease;
    }
    .node-group.hub .node-label,
    .node-group[data-important="true"] .node-label,
    .node-group:hover .node-label,
    .node-group.highlighted .node-label,
    body.show-all-labels .node-label {
      opacity: 1;
    }
    .node-group:hover .node-label {
      font-weight: 600;
      fill: #38bdf8;
    }
    .controls {
      position: absolute;
      bottom: 20px;
      right: 20px;
      display: flex;
      flex-direction: column;
      gap: 6px;
      z-index: 10;
    }
    .ctrl-btn {
      width: 36px;
      height: 36px;
      background: rgba(15, 23, 42, 0.9);
      border: 1px solid var(--border);
      border-radius: 8px;
      color: var(--text);
      font-size: 18px;
      display: flex;
      align-items: center;
      justify-content: center;
      cursor: pointer;
      backdrop-filter: blur(8px);
      transition: background 0.2s, border-color 0.2s;
    }
    .ctrl-btn:hover { background: #1e293b; border-color: var(--accent); }
    #sidebar {
      position: absolute;
      top: 16px;
      right: 16px;
      bottom: 16px;
      width: 380px;
      background: rgba(15, 23, 42, 0.95);
      backdrop-filter: blur(12px);
      border: 1px solid var(--border);
      border-radius: 12px;
      padding: 20px;
      z-index: 20;
      display: none;
      flex-direction: column;
      gap: 16px;
      overflow-y: auto;
      box-shadow: -10px 0 25px -5px rgba(0, 0, 0, 0.5);
    }
    #sidebar.open { display: flex; }
    .sidebar-header {
      display: flex;
      justify-content: space-between;
      align-items: flex-start;
      gap: 12px;
    }
    .sidebar-title { font-size: 17px; font-weight: 700; color: var(--text); word-break: break-word; }
    .close-btn {
      background: none;
      border: none;
      color: var(--text-muted);
      font-size: 20px;
      cursor: pointer;
      padding: 0 4px;
    }
    .close-btn:hover { color: var(--text); }
    .meta-card {
      background: #090d16;
      border: 1px solid var(--border);
      border-radius: 8px;
      padding: 12px;
      display: flex;
      flex-direction: column;
      gap: 8px;
      font-size: 13px;
    }
    .meta-item { display: flex; justify-content: space-between; }
    .meta-label { color: var(--text-muted); }
    .meta-value { font-weight: 600; }
    .links-section { display: flex; flex-direction: column; gap: 8px; }
    .links-title { font-size: 13px; font-weight: 600; text-transform: uppercase; letter-spacing: 0.05em; color: var(--text-muted); }
    .links-list { display: flex; flex-direction: column; gap: 4px; }
    .link-item {
      padding: 6px 10px;
      background: #090d16;
      border: 1px solid var(--border);
      border-radius: 6px;
      font-size: 12px;
      cursor: pointer;
      color: var(--accent);
      transition: all 0.2s;
      text-decoration: none;
      word-break: break-all;
    }
    .link-item:hover { background: #1e293b; border-color: var(--accent); }
    .obsidian-btn {
      padding: 10px;
      background: #4f46e5;
      color: #fff;
      border: none;
      border-radius: 8px;
      font-weight: 600;
      font-size: 13px;
      cursor: pointer;
      text-align: center;
      text-decoration: none;
      transition: background 0.2s;
    }
    .obsidian-btn:hover { background: #4338ca; }
    .triptych-btn {
      padding: 10px;
      background: #0284c7;
      color: #fff;
      border: none;
      border-radius: 8px;
      font-weight: 600;
      font-size: 13px;
      cursor: pointer;
      text-align: center;
      transition: background 0.2s;
    }
    .triptych-btn:hover { background: #0369a1; }
    .triptych-modal {
      display: none;
      position: fixed;
      inset: 0;
      background: rgba(3, 7, 18, 0.85);
      backdrop-filter: blur(8px);
      z-index: 1000;
      justify-content: center;
      align-items: center;
      padding: 24px;
    }
    .triptych-modal.open { display: flex; }
    .triptych-container {
      background: #090d16;
      border: 1px solid #1e293b;
      border-radius: 12px;
      width: 95%;
      max-width: 1200px;
      max-height: 90vh;
      display: flex;
      flex-direction: column;
      box-shadow: 0 25px 50px -12px rgba(0, 0, 0, 0.5);
      overflow: hidden;
    }
    .triptych-header {
      padding: 16px 20px;
      border-bottom: 1px solid #1e293b;
      display: flex;
      justify-content: space-between;
      align-items: center;
      background: #0f172a;
    }
    .triptych-title { font-size: 16px; font-weight: 700; color: #f8fafc; }
    .triptych-grid {
      display: grid;
      grid-template-columns: 1fr 1.2fr 1fr;
      gap: 16px;
      padding: 20px;
      overflow-y: auto;
      background: #090d16;
    }
    @media (max-width: 900px) {
      .triptych-grid { grid-template-columns: 1fr; }
    }
    .triptych-col {
      background: #0d131f;
      border: 1px solid #1e293b;
      border-radius: 8px;
      padding: 16px;
      display: flex;
      flex-direction: column;
      gap: 12px;
    }
    .col-header {
      font-size: 14px;
      font-weight: 700;
      color: #94a3b8;
      border-bottom: 1px solid #1e293b;
      padding-bottom: 8px;
      display: flex;
      justify-content: space-between;
      align-items: center;
    }
    .tp-list { display: flex; flex-direction: column; gap: 8px; max-height: 55vh; overflow-y: auto; }
    .tp-card {
      background: #111827;
      border: 1px solid #1f2937;
      border-radius: 6px;
      padding: 10px;
      cursor: pointer;
      transition: all 0.2s;
    }
    .tp-card:hover { border-color: #38bdf8; background: #1e293b; }
    .tp-card-title { font-size: 13px; font-weight: 600; color: #38bdf8; }
    .tp-card-meta { font-size: 11px; color: #94a3b8; margin-top: 4px; display: flex; justify-content: space-between; }
    .tp-badge-crit { background: rgba(239, 68, 68, 0.2); color: #f87171; padding: 2px 6px; border-radius: 4px; font-size: 10px; font-weight: 700; }
    .tp-badge-high { background: rgba(249, 115, 22, 0.2); color: #fb923c; padding: 2px 6px; border-radius: 4px; font-size: 10px; font-weight: 700; }
    .tp-badge-med { background: rgba(234, 179, 8, 0.2); color: #facc15; padding: 2px 6px; border-radius: 4px; font-size: 10px; font-weight: 700; }
    .tp-badge-low { background: rgba(34, 197, 94, 0.2); color: #4ade80; padding: 2px 6px; border-radius: 4px; font-size: 10px; font-weight: 700; }
  </style>
</head>
<body>
  <header>
    <div class="title-row">
      <div class="title">{{ .Title }}</div>
      {{ if .Repository }}<span class="badge">{{ .Repository }}</span>{{ end }}
    </div>
    <div class="stats-row">
      <span>Nós: <strong>{{ .Stats.TotalNodes }}</strong></span>
      <span>Arestas: <strong>{{ .Stats.TotalEdges }}</strong></span>
      <span>Hubs: <strong>{{ .Stats.HubCount }}</strong></span>
      <span>Densidade: <strong>{{ .Stats.Density }}</strong></span>
      {{ if .FormattedUpdatedAt }}<span>Atualizado em: <strong>{{ .FormattedUpdatedAt }}</strong></span>{{ end }}
    </div>
    <div class="search-row">
      <input type="text" id="search-input" class="search-input" placeholder="Filtrar notas...">
    </div>
    <div class="filter-chips" id="filter-chips">
      <span class="chip active" data-type="all">Todos</span>
      <span class="chip" data-type="concept" style="color: #3b82f6;">Conceitos</span>
      <span class="chip" data-type="decision" style="color: #f43f5e;">Decisões</span>
      <span class="chip" data-type="guide" style="color: #10b981;">Guias</span>
      <span class="chip" data-type="synthesis" style="color: #ec4899;">Sínteses</span>
      <span class="chip" data-type="reference" style="color: #a855f7;">Referências</span>
      <span class="chip" data-type="other" style="color: #64748b;">Outros</span>
    </div>
  </header>

  <div class="controls">
    <button class="ctrl-btn" id="btn-toggle-labels" title="Alternar Rótulos (Ocultar / Mostrar Todos)">🏷️</button>
    <button class="ctrl-btn" id="btn-zoom-in" title="Aumentar Zoom">+</button>
    <button class="ctrl-btn" id="btn-zoom-out" title="Diminuir Zoom">−</button>
    <button class="ctrl-btn" id="btn-reset" title="Centralizar Grafo">⟲</button>
  </div>

  <div id="canvas-container">
    <svg id="graph-svg">
      <defs>
        <marker id="arrow" viewBox="0 0 10 10" refX="22" refY="5" markerWidth="6" markerHeight="6" orient="auto-start-reverse">
          <path d="M 0 1 L 10 5 L 0 9 z" fill="#64748b" />
        </marker>
        <marker id="arrow-highlight" viewBox="0 0 10 10" refX="22" refY="5" markerWidth="7" markerHeight="7" orient="auto-start-reverse">
          <path d="M 0 1 L 10 5 L 0 9 z" fill="#38bdf8" />
        </marker>
      </defs>
      <g id="viewport">
        <g id="edges-layer"></g>
        <g id="nodes-layer"></g>
      </g>
    </svg>
  </div>

  <div id="sidebar">
    <div class="sidebar-header">
      <div class="sidebar-title" id="sb-title">Nota Selecionada</div>
      <button class="close-btn" id="sb-close">&times;</button>
    </div>
    <div class="meta-card">
      <div class="meta-item">
        <span class="meta-label">Tipo:</span>
        <span class="meta-value" id="sb-type">concept</span>
      </div>
      <div class="meta-item">
        <span class="meta-label">PageRank:</span>
        <span class="meta-value" id="sb-pagerank">0.00</span>
      </div>
      <div class="meta-item">
        <span class="meta-label">Entrada (In-links):</span>
        <span class="meta-value" id="sb-in">0</span>
      </div>
      <div class="meta-item">
        <span class="meta-label">Saída (Out-links):</span>
        <span class="meta-value" id="sb-out">0</span>
      </div>
      <div class="meta-item" id="sb-updated-row">
        <span class="meta-label">Última atualização:</span>
        <span class="meta-value" id="sb-updated">-</span>
      </div>
    </div>
    <div class="links-section">
      <div class="links-title">Conexões de Entrada</div>
      <div class="links-list" id="sb-in-links"></div>
    </div>
    <div class="links-section">
      <div class="links-title">Conexões de Saída</div>
      <div class="links-list" id="sb-out-links"></div>
    </div>
    <div style="display:flex; flex-direction:column; gap:8px; margin-top:auto;">
      <button id="sb-inspect-btn" class="triptych-btn" onclick="openTriptychModal()">🔬 Inspecionar Tríptico</button>
      <a href="#" id="sb-obsidian-link" class="obsidian-btn" target="_blank">Abrir no Obsidian</a>
    </div>
  </div>

  <!-- Modal do Tríptico (3 Colunas) -->
  <div id="triptych-modal" class="triptych-modal">
    <div class="triptych-container">
      <div class="triptych-header">
        <div class="triptych-title">🔬 Visualização Cirúrgica em 3 Colunas (Triptych Node Inspector)</div>
        <button class="close-btn" onclick="closeTriptychModal()">✕</button>
      </div>
      <div class="triptych-grid">
        <div class="triptych-col">
          <div class="col-header">
            <span>⬅️ Chamadores (In-links)</span>
            <span id="tp-in-count" class="badge">0</span>
          </div>
          <div id="tp-in-list" class="tp-list"></div>
        </div>
        <div class="triptych-col">
          <div class="col-header">🎯 Nó Central & Métricas</div>
          <div id="tp-center-card" style="display:flex; flex-direction:column; gap:10px; font-size:13px;">
            <div style="font-size:16px; font-weight:700; color:#38bdf8;" id="tp-title">Título</div>
            <div class="meta-item"><span class="meta-label">ID Canônico:</span><span id="tp-id" class="meta-value"></span></div>
            <div class="meta-item"><span class="meta-label">Tipo:</span><span id="tp-type" class="meta-value"></span></div>
            <div class="meta-item"><span class="meta-label">PageRank:</span><span id="tp-pagerank" class="meta-value"></span></div>
            <div class="meta-item"><span class="meta-label">Comunidade:</span><span id="tp-community" class="meta-value"></span></div>
            <div class="meta-item"><span class="meta-label">Total Conexões:</span><span id="tp-degree" class="meta-value"></span></div>
            <a href="#" id="tp-obsidian-btn" class="obsidian-btn" target="_blank" style="margin-top:12px;">Abrir no Obsidian</a>
          </div>
        </div>
        <div class="triptych-col">
          <div class="col-header">
            <span>➡️ Referências (Out-links)</span>
            <span id="tp-out-count" class="badge">0</span>
          </div>
          <div id="tp-out-list" class="tp-list"></div>
        </div>
      </div>
    </div>
  </div>

  <script>
    const DATA = {{ .DataJSON }};
    const nodes = DATA.nodes;
    const edges = DATA.edges;

    const width = window.innerWidth;
    const height = window.innerHeight;

    // Inicialização espalhada em anel elíptico proporcional à tela
    const nodeMap = new Map();
    const rx = Math.min(width, height) * 0.35;
    nodes.forEach((n, i) => {
      const angle = (2 * Math.PI * i) / (nodes.length || 1);
      const spread = rx * (0.6 + 0.4 * Math.random());
      n.x = (width / 2) + spread * Math.cos(angle);
      n.y = (height / 2) + spread * Math.sin(angle);
      n.vx = (Math.random() - 0.5) * 2;
      n.vy = (Math.random() - 0.5) * 2;
      nodeMap.set(n.id, n);
    });

    // Mapeamento de arestas com referências diretas
    const links = [];
    edges.forEach(e => {
      const source = nodeMap.get(e.source);
      const target = nodeMap.get(e.target);
      if (source && target) {
        links.push({ ...e, source, target });
      }
    });

    const svg = document.getElementById('graph-svg');
    const viewport = document.getElementById('viewport');
    const edgesLayer = document.getElementById('edges-layer');
    const nodesLayer = document.getElementById('nodes-layer');

    // Renderizar Arestas SVG
    links.forEach(l => {
      const line = document.createElementNS('http://www.w3.org/2000/svg', 'line');
      line.setAttribute('class', 'edge' + (l.epistemic_status === 'INFERRED' ? ' inferred' : ''));
      line.setAttribute('stroke', l.color || '#475569');
      line.setAttribute('marker-end', 'url(#arrow)');
      l.element = line;
      edgesLayer.appendChild(line);
    });

    // Renderizar Nós SVG
    nodes.forEach(n => {
      const g = document.createElementNS('http://www.w3.org/2000/svg', 'g');
      g.setAttribute('class', 'node-group' + (n.is_hub ? ' hub' : ''));
      g.setAttribute('data-id', n.id);
      g.setAttribute('data-type', n.type);

      // Identifica nós com maior relevância para exibir rótulo por padrão
      const degree = (n.in_degree || 0) + (n.out_degree || 0);
      if (n.is_hub || degree >= 3 || (n.pagerank && n.pagerank > 0.015)) {
        g.setAttribute('data-important', 'true');
      }

      const circle = document.createElementNS('http://www.w3.org/2000/svg', 'circle');
      circle.setAttribute('class', 'node-circle');
      circle.setAttribute('r', n.radius || 10);
      circle.setAttribute('fill', n.color);

      // Tooltip nativo
      const svgTitle = document.createElementNS('http://www.w3.org/2000/svg', 'title');
      svgTitle.textContent = (n.title || n.id) + ' (Grau: ' + degree + ', PR: ' + (n.pagerank ? n.pagerank.toFixed(4) : '0') + ')';
      g.appendChild(svgTitle);

      const text = document.createElementNS('http://www.w3.org/2000/svg', 'text');
      text.setAttribute('class', 'node-label');
      text.setAttribute('dy', (n.radius || 10) + 14);

      // Limpeza e encurtamento do título para o nó no canvas
      let displayTitle = n.title || n.id;
      const lastSlash = Math.max(displayTitle.lastIndexOf('/'), displayTitle.lastIndexOf('\\'));
      if (lastSlash !== -1) {
        displayTitle = displayTitle.slice(lastSlash + 1);
      }
      if (displayTitle.length > 22) {
        displayTitle = displayTitle.slice(0, 20) + '…';
      }
      text.textContent = displayTitle;

      g.appendChild(circle);
      g.appendChild(text);
      n.element = g;

      // Eventos de clique e arraste
      g.addEventListener('click', (ev) => {
        ev.stopPropagation();
        selectNode(n);
      });

      g.addEventListener('mousedown', (ev) => {
        ev.stopPropagation();
        startDrag(n, ev);
      });

      nodesLayer.appendChild(g);
    });

    // Simulação Force-Directed Graph com anti-colisão física
    let alpha = 1.0;
    const alphaDecay = 0.008;
    const alphaMin = 0.001;

    function tickSimulation() {
      if (alpha < alphaMin) return;

      const k = 150; // distância ideal de mola
      const repStrength = 4200; // força de repulsão

      // 1. Repulsão Many-Body (Coulomb) + Anti-colisão física
      for (let i = 0; i < nodes.length; i++) {
        const a = nodes[i];
        for (let j = i + 1; j < nodes.length; j++) {
          const b = nodes[j];
          let dx = b.x - a.x;
          let dy = b.y - a.y;
          let dist = Math.sqrt(dx * dx + dy * dy) || 1;
          if (dist > 550) continue;

          let f = (repStrength / (dist * dist)) * alpha;
          let fx = (dx / dist) * f;
          let fy = (dy / dist) * f;

          // Anti-colisão: mantém espaçamento físico entre círculos
          const minDist = (a.radius || 10) + (b.radius || 10) + 26;
          if (dist < minDist) {
            const overlap = (minDist - dist) / minDist;
            const push = overlap * 3.5 * alpha;
            fx += (dx / dist) * push;
            fy += (dy / dist) * push;
          }

          if (!a.pinned) { a.vx -= fx; a.vy -= fy; }
          if (!b.pinned) { b.vx += fx; b.vy += fy; }
        }
      }

      // 2. Atração por Arestas (Hooke)
      for (let i = 0; i < links.length; i++) {
        const l = links[i];
        let dx = l.target.x - l.source.x;
        let dy = l.target.y - l.source.y;
        let dist = Math.sqrt(dx * dx + dy * dy) || 1;

        let force = (dist - k) * 0.04 * alpha;
        let fx = (dx / dist) * force;
        let fy = (dy / dist) * force;

        if (!l.source.pinned) { l.source.vx += fx; l.source.vy += fy; }
        if (!l.target.pinned) { l.target.vx -= fx; l.target.vy -= fy; }
      }

      // 3. Centralização Suave e Amortecimento
      const centerX = width / 2;
      const centerY = height / 2;
      for (let i = 0; i < nodes.length; i++) {
        const n = nodes[i];
        if (n.pinned) continue;

        n.vx += (centerX - n.x) * 0.0012 * alpha;
        n.vy += (centerY - n.y) * 0.0012 * alpha;

        n.vx *= 0.82;
        n.vy *= 0.82;

        n.x += n.vx;
        n.y += n.vy;
      }

      alpha *= (1 - alphaDecay);

      // Atualizar SVG
      links.forEach(l => {
        l.element.setAttribute('x1', l.source.x);
        l.element.setAttribute('y1', l.source.y);
        l.element.setAttribute('x2', l.target.x);
        l.element.setAttribute('y2', l.target.y);
      });

      nodes.forEach(n => {
        n.element.setAttribute('transform', 'translate(' + n.x + ',' + n.y + ')');
      });

      requestAnimationFrame(tickSimulation);
    }

    requestAnimationFrame(tickSimulation);

    function restartSimulation() {
      alpha = 0.5;
      requestAnimationFrame(tickSimulation);
    }

    // Zoom e Pan
    let zoom = 1.0;
    let panX = 0;
    let panY = 0;
    let isPanning = false;
    let startPanX = 0;
    let startPanY = 0;

    function updateTransform() {
      viewport.setAttribute('transform', 'translate(' + panX + ',' + panY + ') scale(' + zoom + ')');
    }

    const container = document.getElementById('canvas-container');
    container.addEventListener('wheel', (ev) => {
      ev.preventDefault();
      const zoomFactor = ev.deltaY < 0 ? 1.1 : 0.9;
      zoom = Math.max(0.2, Math.min(4.0, zoom * zoomFactor));
      updateTransform();
    }, { passive: false });

    container.addEventListener('mousedown', (ev) => {
      if (ev.target === svg || ev.target === container) {
        isPanning = true;
        startPanX = ev.clientX - panX;
        startPanY = ev.clientY - panY;
        container.classList.add('panning');
      }
    });

    window.addEventListener('mousemove', (ev) => {
      if (isPanning) {
        panX = ev.clientX - startPanX;
        panY = ev.clientY - startPanY;
        updateTransform();
      }
      if (draggedNode) {
        draggedNode.x = (ev.clientX - panX) / zoom;
        draggedNode.y = (ev.clientY - panY) / zoom;
        draggedNode.pinned = true;
        restartSimulation();
      }
    });

    window.addEventListener('mouseup', () => {
      if (isPanning) {
        isPanning = false;
        container.classList.remove('panning');
      }
      if (draggedNode) {
        draggedNode.pinned = false;
        draggedNode = null;
      }
    });

    // Drag & Drop
    let draggedNode = null;
    function startDrag(n, ev) {
      draggedNode = n;
      restartSimulation();
    }

    // Controles
    const btnToggleLabels = document.getElementById('btn-toggle-labels');
    if (btnToggleLabels) {
      btnToggleLabels.addEventListener('click', () => {
        document.body.classList.toggle('show-all-labels');
        const active = document.body.classList.contains('show-all-labels');
        btnToggleLabels.style.borderColor = active ? 'var(--accent)' : 'var(--border)';
        btnToggleLabels.style.background = active ? 'rgba(56, 189, 248, 0.2)' : 'rgba(15, 23, 42, 0.9)';
      });
    }

    document.getElementById('btn-zoom-in').addEventListener('click', () => {
      zoom = Math.min(4.0, zoom * 1.2);
      updateTransform();
    });
    document.getElementById('btn-zoom-out').addEventListener('click', () => {
      zoom = Math.max(0.2, zoom / 1.2);
      updateTransform();
    });
    document.getElementById('btn-reset').addEventListener('click', () => {
      zoom = 1.0;
      panX = 0;
      panY = 0;
      updateTransform();
      restartSimulation();
    });

    // Painel Lateral e Seleção
    const sidebar = document.getElementById('sidebar');
    let selectedNode = null;

    function selectNode(n) {
      selectedNode = n;
      document.getElementById('sb-title').textContent = n.title;
      document.getElementById('sb-type').textContent = n.type;
      document.getElementById('sb-pagerank').textContent = n.pagerank.toFixed(4);
      document.getElementById('sb-in').textContent = n.in_degree;
      document.getElementById('sb-out').textContent = n.out_degree;

      const updRow = document.getElementById('sb-updated-row');
      const updVal = document.getElementById('sb-updated');
      if (n.updated_at && n.updated_at > 0) {
        const d = new Date(n.updated_at * 1000);
        updVal.textContent = d.toLocaleDateString('pt-BR') + ' ' + d.toLocaleTimeString('pt-BR', {hour: '2-digit', minute: '2-digit'});
        if (updRow) updRow.style.display = 'flex';
      } else {
        updVal.textContent = '-';
        if (updRow) updRow.style.display = 'none';
      }

      // Deep link para Obsidian
      const obsLink = 'obsidian://open?file=' + encodeURIComponent(n.id);
      document.getElementById('sb-obsidian-link').setAttribute('href', obsLink);

      // In-links
      const inList = document.getElementById('sb-in-links');
      inList.innerHTML = '';
      const inEdges = links.filter(l => l.target.id === n.id);
      if (inEdges.length === 0) {
        inList.innerHTML = '<span style="font-size:12px;color:var(--text-muted);">Nenhum link de entrada</span>';
      } else {
        inEdges.forEach(l => {
          const item = document.createElement('div');
          item.className = 'link-item';
          item.textContent = l.source.title + ' (' + l.relation + ')';
          item.onclick = () => selectNode(l.source);
          inList.appendChild(item);
        });
      }

      // Out-links
      const outList = document.getElementById('sb-out-links');
      outList.innerHTML = '';
      const outEdges = links.filter(l => l.source.id === n.id);
      if (outEdges.length === 0) {
        outList.innerHTML = '<span style="font-size:12px;color:var(--text-muted);">Nenhum link de saída</span>';
      } else {
        outEdges.forEach(l => {
          const item = document.createElement('div');
          item.className = 'link-item';
          item.textContent = l.target.title + ' (' + l.relation + ')';
          item.onclick = () => selectNode(l.target);
          outList.appendChild(item);
        });
      }

      // Destaque visual
      highlightConnections(n);
      sidebar.classList.add('open');
    }

    function highlightConnections(centerNode) {
      const connected = new Set();
      connected.add(centerNode.id);

      links.forEach(l => {
        if (l.source.id === centerNode.id || l.target.id === centerNode.id) {
          connected.add(l.source.id);
          connected.add(l.target.id);
          l.element.classList.add('highlighted');
          l.element.setAttribute('stroke', '#38bdf8');
          l.element.setAttribute('marker-end', 'url(#arrow-highlight)');
        } else {
          l.element.classList.remove('highlighted');
          l.element.setAttribute('stroke', l.color || '#475569');
          l.element.setAttribute('marker-end', 'url(#arrow)');
        }
      });

      nodes.forEach(n => {
        if (connected.has(n.id)) {
          n.element.style.opacity = '1';
          n.element.classList.add('highlighted');
        } else {
          n.element.style.opacity = '0.2';
          n.element.classList.remove('highlighted');
        }
      });
    }

    function clearHighlight() {
      links.forEach(l => {
        l.element.classList.remove('highlighted');
        l.element.setAttribute('stroke', l.color || '#475569');
        l.element.setAttribute('marker-end', 'url(#arrow)');
      });
      nodes.forEach(n => {
        n.element.style.opacity = '1';
        n.element.classList.remove('highlighted');
      });
    }

    document.getElementById('sb-close').addEventListener('click', () => {
      sidebar.classList.remove('open');
      clearHighlight();
    });

    container.addEventListener('click', (ev) => {
      if (ev.target === svg || ev.target === container) {
        sidebar.classList.remove('open');
        clearHighlight();
      }
    });

    // Busca em tempo real
    const searchInput = document.getElementById('search-input');
    searchInput.addEventListener('input', () => {
      const query = searchInput.value.trim().toLowerCase();
      if (!query) {
        clearHighlight();
        return;
      }
      nodes.forEach(n => {
        const match = n.title.toLowerCase().includes(query) || n.id.toLowerCase().includes(query);
        n.element.style.opacity = match ? '1' : '0.15';
      });
    });

    // Filtros por tipo de nota
    const chips = document.querySelectorAll('.filter-chips .chip');
    chips.forEach(chip => {
      chip.addEventListener('click', () => {
        chips.forEach(c => c.classList.remove('active'));
        chip.classList.add('active');
        const filterType = chip.getAttribute('data-type');
        nodes.forEach(n => {
          if (filterType === 'all' || n.type === filterType) {
            n.element.style.display = 'block';
          } else {
            n.element.style.display = 'none';
          }
        });
        links.forEach(l => {
          const srcVisible = filterType === 'all' || l.source.type === filterType;
          const tgtVisible = filterType === 'all' || l.target.type === filterType;
          l.element.style.display = (srcVisible && tgtVisible) ? 'block' : 'none';
        });
      });
    });

    // Funções do Visualizador Cirúrgico (Triptych Modal)
    function openTriptychModal() {
      if (!selectedNode) return;
      renderTriptych(selectedNode);
      document.getElementById('triptych-modal').classList.add('open');
    }

    function closeTriptychModal() {
      document.getElementById('triptych-modal').classList.remove('open');
    }

    function renderTriptych(n) {
      document.getElementById('tp-title').textContent = n.title;
      document.getElementById('tp-id').textContent = n.id;
      document.getElementById('tp-type').textContent = n.type;
      document.getElementById('tp-pagerank').textContent = n.pagerank.toFixed(4);
      document.getElementById('tp-community').textContent = n.community_id > 0 ? ('Cluster #' + n.community_id + ' (' + (n.community_label || 'Geral') + ')') : 'Não agrupado';
      document.getElementById('tp-degree').textContent = (n.in_degree + n.out_degree) + ' (In: ' + n.in_degree + ' | Out: ' + n.out_degree + ')';

      const obsLink = 'obsidian://open?file=' + encodeURIComponent(n.id);
      document.getElementById('tp-obsidian-btn').setAttribute('href', obsLink);

      // Inbound
      const inList = document.getElementById('tp-in-list');
      inList.innerHTML = '';
      const inEdges = links.filter(l => l.target.id === n.id);
      document.getElementById('tp-in-count').textContent = inEdges.length;

      if (inEdges.length === 0) {
        inList.innerHTML = '<div style="font-size:12px;color:var(--text-muted);padding:8px;">Nenhum nó apontando para este documento</div>';
      } else {
        inEdges.forEach(l => {
          const card = document.createElement('div');
          card.className = 'tp-card';
          
          let rel = l.relation || 'links_to';
          let badgeClass = 'tp-badge-low';
          let badgeText = 'BAIXO';
          let relLower = rel.toLowerCase();
          if (relLower.includes('implement') || relLower.includes('depend') || relLower.includes('contradict') || relLower.includes('block')) {
            badgeClass = 'tp-badge-crit';
            badgeText = 'CRÍTICO';
          } else if (relLower.includes('link') || relLower.includes('refer')) {
            badgeClass = 'tp-badge-med';
            badgeText = 'MÉDIO';
          }

          card.innerHTML = '<div style="display:flex; justify-content:space-between; align-items:center;">' +
            '<span class="tp-card-title">' + l.source.title + '</span>' +
            '<span class="' + badgeClass + '">' + badgeText + '</span>' +
            '</div>' +
            '<div class="tp-card-meta">' +
            '<span>Relação: <code>' + rel + '</code></span>' +
            '<span>PR: ' + (l.source.pagerank || 0).toFixed(4) + '</span>' +
            '</div>';
          card.onclick = () => {
            selectNode(l.source);
            renderTriptych(l.source);
          };
          inList.appendChild(card);
        });
      }

      // Outbound
      const outList = document.getElementById('tp-out-list');
      outList.innerHTML = '';
      const outEdges = links.filter(l => l.source.id === n.id);
      document.getElementById('tp-out-count').textContent = outEdges.length;

      if (outEdges.length === 0) {
        outList.innerHTML = '<div style="font-size:12px;color:var(--text-muted);padding:8px;">Este documento não referencia outras notas</div>';
      } else {
        outEdges.forEach(l => {
          const card = document.createElement('div');
          card.className = 'tp-card';
          card.innerHTML = '<div style="display:flex; justify-content:space-between; align-items:center;">' +
            '<span class="tp-card-title">' + l.target.title + '</span>' +
            '<span style="font-size:11px; color:#38bdf8;">✓ Válido</span>' +
            '</div>' +
            '<div class="tp-card-meta">' +
            '<span>Relação: <code>' + (l.relation || 'links_to') + '</code></span>' +
            '<span>PR: ' + (l.target.pagerank || 0).toFixed(4) + '</span>' +
            '</div>';
          card.onclick = () => {
            selectNode(l.target);
            renderTriptych(l.target);
          };
          outList.appendChild(card);
        });
      }
    }

    window.addEventListener('keydown', (e) => {
      if (e.key === 'Escape') closeTriptychModal();
    });
  </script>
</body>
</html>`

// TemplateData empacota os dados para injeção no template HTML
type TemplateData struct {
	Title              string
	Repository         string
	Stats              GraphStats
	FormattedUpdatedAt string
	DataJSON           template.JS
}

// RenderHTML renderiza a página HTML standalone em memória com todos os dados do grafo embutidos
func RenderHTML(gv *GraphView) ([]byte, error) {
	dataBytes, err := json.Marshal(gv)
	if err != nil {
		return nil, fmt.Errorf("falha ao serializar GraphView em JSON: %w", err)
	}

	tmpl, err := template.New("graphview").Parse(htmlPageTemplate)
	if err != nil {
		return nil, fmt.Errorf("falha ao compilar template html do grafo: %w", err)
	}

	formattedUpdated := ""
	if gv != nil {
		formattedUpdated = gv.FormattedUpdatedAt()
	}

	td := TemplateData{
		Title:              gv.Title,
		Repository:         gv.Repository,
		Stats:              gv.Stats,
		FormattedUpdatedAt: formattedUpdated,
		DataJSON:           template.JS(dataBytes),
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, td); err != nil {
		return nil, fmt.Errorf("falha ao executar template html: %w", err)
	}

	return buf.Bytes(), nil
}

// ExportHTML gera o arquivo HTML standalone no caminho de destino
func ExportHTML(gv *GraphView, outputPath string) error {
	htmlBytes, err := RenderHTML(gv)
	if err != nil {
		return err
	}
	return os.WriteFile(outputPath, htmlBytes, 0644)
}
