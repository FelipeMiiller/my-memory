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
      margin-top: auto;
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
    </div>
    <div class="links-section">
      <div class="links-title">Conexões de Entrada</div>
      <div class="links-list" id="sb-in-links"></div>
    </div>
    <div class="links-section">
      <div class="links-title">Conexões de Saída</div>
      <div class="links-list" id="sb-out-links"></div>
    </div>
    <a href="#" id="sb-obsidian-link" class="obsidian-btn" target="_blank">Abrir no Obsidian</a>
  </div>

  <script>
    const DATA = {{ .DataJSON }};
    const nodes = DATA.nodes;
    const edges = DATA.edges;

    const width = window.innerWidth;
    const height = window.innerHeight;

    // Inicialização aleatória com dispersão inicial
    const nodeMap = new Map();
    nodes.forEach((n, i) => {
      const angle = (2 * Math.PI * i) / nodes.length;
      const radius = 180 + Math.random() * 200;
      n.x = (width / 2) + radius * Math.cos(angle);
      n.y = (height / 2) + radius * Math.sin(angle);
      n.vx = 0;
      n.vy = 0;
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

      const circle = document.createElementNS('http://www.w3.org/2000/svg', 'circle');
      circle.setAttribute('class', 'node-circle');
      circle.setAttribute('r', n.radius || 10);
      circle.setAttribute('fill', n.color);

      const text = document.createElementNS('http://www.w3.org/2000/svg', 'text');
      text.setAttribute('class', 'node-label');
      text.setAttribute('dy', (n.radius || 10) + 14);
      text.textContent = n.title;

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

    // Simulação Force-Directed Graph em JS puro
    let alpha = 1.0;
    const alphaDecay = 0.008;
    const alphaMin = 0.001;

    function tickSimulation() {
      if (alpha < alphaMin) return;

      const k = 140; // distância ideal de mola
      const repStrength = 1800; // força de repulsão

      // 1. Repulsão Many-Body (Coulomb)
      for (let i = 0; i < nodes.length; i++) {
        for (let j = i + 1; j < nodes.length; j++) {
          const a = nodes[i];
          const b = nodes[j];
          let dx = b.x - a.x;
          let dy = b.y - a.y;
          let dist = Math.sqrt(dx * dx + dy * dy) || 1;
          if (dist > 450) continue;

          let f = (repStrength / (dist * dist)) * alpha;
          let fx = (dx / dist) * f;
          let fy = (dy / dist) * f;

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

        let force = (dist - k) * 0.05 * alpha;
        let fx = (dx / dist) * force;
        let fy = (dy / dist) * force;

        if (!l.source.pinned) { l.source.vx += fx; l.source.vy += fy; }
        if (!l.target.pinned) { l.target.vx -= fx; l.target.vy -= fy; }
      }

      // 3. Centralização e Amortecimento
      const centerX = width / 2;
      const centerY = height / 2;
      for (let i = 0; i < nodes.length; i++) {
        const n = nodes[i];
        if (n.pinned) continue;

        n.vx += (centerX - n.x) * 0.008 * alpha;
        n.vy += (centerY - n.y) * 0.008 * alpha;

        n.vx *= 0.85;
        n.vy *= 0.85;

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
        } else {
          n.element.style.opacity = '0.2';
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
  </script>
</body>
</html>`

// TemplateData empacota os dados para injeção no template HTML
type TemplateData struct {
	Title      string
	Repository string
	Stats      GraphStats
	DataJSON   template.JS
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

	td := TemplateData{
		Title:      gv.Title,
		Repository: gv.Repository,
		Stats:      gv.Stats,
		DataJSON:   template.JS(dataBytes),
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
