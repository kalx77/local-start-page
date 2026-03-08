'use strict';

let state = { background: '#0f0f1a', groups: [] };

const COLS = 12;

// ── Position helpers ────────────────────────────────────────────────────────

function groupCells(g) {
  const w = g.w || 1;
  const cells = [];
  for (let dx = 0; dx < w; dx++) cells.push(`${g.x + dx},${g.y}`);
  return cells;
}

function occupiedCells(excludeIdx = -1) {
  const cells = new Set();
  state.groups.forEach((g, i) => {
    if (i !== excludeIdx) groupCells(g).forEach(c => cells.add(c));
  });
  return cells;
}

function assignMissingPositions() {
  const occupied = new Set();
  state.groups.forEach((g, i) => {
    if (!g.w) g.w = 1;
    const mine = groupCells(g);
    const conflict = g.x == null || g.y == null || mine.some(c => occupied.has(c));
    if (conflict) {
      g.w = 1;
      const pos = findFreeCell(i);
      g.x = pos.x; g.y = pos.y;
    }
    groupCells(g).forEach(c => occupied.add(c));
  });
}

function findFreeCell(excludeIdx = -1) {
  const occupied = occupiedCells(excludeIdx);
  for (let row = 0; row < 100; row++)
    for (let col = 0; col < COLS; col++)
      if (!occupied.has(`${col},${row}`)) return { x: col, y: row };
}

// ── Fetch helpers ──────────────────────────────────────────────────────────

async function loadConfig() {
  const res = await fetch('/api/config');
  state = await res.json();
  if (!state.groups) state.groups = [];

  // Migrate from 6-col to 12-col: all x < 6 and some w <= 1
  const needsMigration = state.groups.length > 0
    && state.groups.every(g => g.x < 6)
    && state.groups.some(g => (g.w || 1) <= 1);
  if (needsMigration) {
    state.groups.forEach(g => { g.x *= 2; g.w = Math.max((g.w || 1) * 2, 2); });
  }

  const before = JSON.stringify(state.groups.map(g => [g.x, g.y, g.w]));
  assignMissingPositions();
  const after = JSON.stringify(state.groups.map(g => [g.x, g.y, g.w]));
  applyBackground();
  render();
  if (before !== after || needsMigration) saveConfig();
}

async function saveConfig() {
  await fetch('/api/config', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(state),
  });
}

// ── Background ─────────────────────────────────────────────────────────────

function applyBackground() {
  const v = state.background || '#0f0f1a';
  if (v.startsWith('url(')) {
    document.body.style.background = '';
    document.body.style.backgroundImage = v;
    document.body.style.backgroundSize = 'cover';
    document.body.style.backgroundPosition = 'center';
    document.body.style.backgroundRepeat = 'no-repeat';
  } else {
    document.body.style.backgroundImage = '';
    document.body.style.backgroundSize = '';
    document.body.style.backgroundPosition = '';
    document.body.style.backgroundRepeat = '';
    document.body.style.background = v;
  }
}

// ── Icon rendering ─────────────────────────────────────────────────────────

function isEmoji(str) {
  if (!str) return false;
  const cp = str.codePointAt(0);
  return cp > 0x2000;
}

function iconElement(link) {
  const icon = link.icon || '';
  if (icon && isEmoji(icon)) {
    const span = document.createElement('span');
    span.className = 'icon';
    span.textContent = icon;
    return span;
  }
  const img = document.createElement('img');
  img.className = 'icon';
  if (icon && (icon.startsWith('http://') || icon.startsWith('https://'))) {
    img.src = icon;
  } else {
    try {
      const host = new URL(link.url).hostname;
      img.src = `https://www.google.com/s2/favicons?domain=${host}&sz=32`;
    } catch {
      img.src = '';
    }
  }
  img.alt = '';
  img.onerror = () => { img.style.display = 'none'; };
  return img;
}

// ── Render ─────────────────────────────────────────────────────────────────

function render() {
  const container = document.getElementById('groups');
  container.innerHTML = '';

  state.groups.forEach((group, gi) => {
    const el = document.createElement('div');
    el.className = 'group' + (group.collapsed ? ' group--collapsed' : '');
    el.dataset.gi = gi;
    el.style.gridColumn = `${group.x + 1} / span ${group.w || 1}`;
    el.style.gridRow = group.y + 1;
    if (group.color) el.style.background = group.color;

    // Header
    const header = document.createElement('div');
    header.className = 'group-header';
    header.addEventListener('mousedown', (e) => {
      if (e.target.closest('.edit-btns') || e.target.closest('.btn-collapse')) return;
      if (document.body.classList.contains('editing')) startDrag(gi, e);
    });
    header.addEventListener('click', (e) => {
      if (e.target.closest('.edit-btns') || e.target.closest('.btn-collapse')) return;
      toggleCollapse(gi);
    });

    const btnCollapse = document.createElement('button');
    btnCollapse.className = 'btn-collapse';
    btnCollapse.title = group.collapsed ? 'Expand' : 'Collapse';
    btnCollapse.textContent = group.collapsed ? '▸' : '▾';
    btnCollapse.addEventListener('click', (e) => { e.stopPropagation(); toggleCollapse(gi); });
    header.appendChild(btnCollapse);

    const title = document.createElement('span');
    title.className = 'group-title';
    title.textContent = group.name;
    header.appendChild(title);

    const editBtns = document.createElement('div');
    editBtns.className = 'edit-btns';

    const btnRename = document.createElement('button');
    btnRename.className = 'btn-icon';
    btnRename.textContent = 'rename';
    btnRename.onclick = (e) => { e.stopPropagation(); openGroupModal(gi); };
    editBtns.appendChild(btnRename);

    const btnDelGroup = document.createElement('button');
    btnDelGroup.className = 'btn-icon danger';
    btnDelGroup.textContent = 'del';
    btnDelGroup.onclick = (e) => { e.stopPropagation(); deleteGroup(gi); };
    editBtns.appendChild(btnDelGroup);

    header.appendChild(editBtns);
    el.appendChild(header);

    // Links
    const links = document.createElement('div');
    links.className = 'links';

    (group.links || []).forEach((link, li) => {
      const row = document.createElement('div');
      row.className = 'link-row';

      const a = document.createElement('a');
      a.className = 'link-anchor';
      a.href = link.url;
      a.target = '_blank';
      a.rel = 'noopener noreferrer';
      a.appendChild(iconElement(link));
      const nameSpan = document.createElement('span');
      nameSpan.textContent = link.name;
      a.appendChild(nameSpan);
      row.appendChild(a);

      const rowBtns = document.createElement('div');
      rowBtns.className = 'edit-btns';

      const btnEdit = document.createElement('button');
      btnEdit.className = 'btn-icon';
      btnEdit.textContent = 'edit';
      btnEdit.onclick = () => openLinkModal(gi, li);
      rowBtns.appendChild(btnEdit);

      const btnDel = document.createElement('button');
      btnDel.className = 'btn-icon danger';
      btnDel.textContent = 'del';
      btnDel.onclick = () => deleteLink(gi, li);
      rowBtns.appendChild(btnDel);

      row.appendChild(rowBtns);
      links.appendChild(row);
    });

    el.appendChild(links);

    // Add link button
    const btnAddLink = document.createElement('button');
    btnAddLink.className = 'btn-icon btn-add-link';
    btnAddLink.textContent = '+ Link';
    btnAddLink.onclick = () => openLinkModal(gi, null);
    el.appendChild(btnAddLink);

    // Resize handle
    const resizeHandle = document.createElement('div');
    resizeHandle.className = 'resize-handle';
    resizeHandle.addEventListener('mousedown', (e) => {
      e.stopPropagation();
      if (document.body.classList.contains('editing')) startResize(gi, e);
    });
    el.appendChild(resizeHandle);

    container.appendChild(el);
  });

  // Add group row visibility
  document.getElementById('add-group-row').classList.toggle(
    'hidden',
    !document.body.classList.contains('editing')
  );
}

// ── Drag logic ─────────────────────────────────────────────────────────────

let drag = null;

function startDrag(gi, e) {
  e.preventDefault();
  const group = state.groups[gi];
  const sourceEl = document.querySelector(`.group[data-gi="${gi}"]`);
  const rect = sourceEl.getBoundingClientRect();
  const offsetX = e.clientX - rect.left;
  const offsetY = e.clientY - rect.top;

  const ghost = sourceEl.cloneNode(true);
  ghost.id = 'drag-ghost';
  ghost.style.width = rect.width + 'px';
  ghost.style.left = (e.clientX - offsetX) + 'px';
  ghost.style.top = (e.clientY - offsetY) + 'px';
  document.body.appendChild(ghost);

  sourceEl.classList.add('dragging');
  document.body.style.userSelect = 'none';

  drag = { gi, offsetX, offsetY, targetX: group.x, targetY: group.y };
}

function getCellFromMouse(clientX, clientY) {
  const gridEl = document.getElementById('groups');
  const gridRect = gridEl.getBoundingClientRect();
  const col = Math.max(0, Math.min(COLS - 1, Math.floor((clientX - gridRect.left) / (gridRect.width / COLS))));

  // Build row map from group elements
  const rows = {};
  document.querySelectorAll('.group').forEach(el => {
    const r = el.getBoundingClientRect();
    const rowIdx = parseInt(el.style.gridRow) - 1;
    if (!rows[rowIdx]) rows[rowIdx] = { top: r.top, bottom: r.bottom };
    else {
      rows[rowIdx].top = Math.min(rows[rowIdx].top, r.top);
      rows[rowIdx].bottom = Math.max(rows[rowIdx].bottom, r.bottom);
    }
  });

  const sortedRows = Object.keys(rows).map(Number).sort((a, b) => a - b);
  if (sortedRows.length === 0) return { x: col, y: 0 };

  // Above all rows
  if (clientY < rows[sortedRows[0]].top) return { x: col, y: sortedRows[0] };
  // Below all rows
  if (clientY >= rows[sortedRows[sortedRows.length - 1]].bottom) {
    return { x: col, y: sortedRows[sortedRows.length - 1] + 1 };
  }
  // Within a row
  for (const rowIdx of sortedRows) {
    const r = rows[rowIdx];
    if (clientY >= r.top && clientY < r.bottom) return { x: col, y: rowIdx };
  }
  // Between rows — pick closer one
  for (let i = 0; i < sortedRows.length - 1; i++) {
    const r1 = rows[sortedRows[i]];
    const r2 = rows[sortedRows[i + 1]];
    if (clientY >= r1.bottom && clientY < r2.top) {
      const mid = (r1.bottom + r2.top) / 2;
      return { x: col, y: clientY < mid ? sortedRows[i] : sortedRows[i + 1] };
    }
  }
  return { x: col, y: sortedRows[0] };
}

document.addEventListener('mousemove', (e) => {
  if (!drag) return;
  const ghost = document.getElementById('drag-ghost');
  if (!ghost) return;
  ghost.style.left = (e.clientX - drag.offsetX) + 'px';
  ghost.style.top = (e.clientY - drag.offsetY) + 'px';

  const cell = getCellFromMouse(e.clientX, e.clientY);
  drag.targetX = cell.x;
  drag.targetY = cell.y;

  const occupied = occupiedCells(drag.gi);
  const dragW = state.groups[drag.gi].w || 1;
  let blocked = false;
  for (let dx = 0; dx < dragW; dx++) {
    if (occupied.has(`${cell.x + dx},${cell.y}`)) { blocked = true; break; }
  }
  ghost.classList.toggle('blocked', blocked);
});

document.addEventListener('mouseup', async (e) => {
  if (!drag) return;
  const ghost = document.getElementById('drag-ghost');
  if (ghost) ghost.remove();

  const sourceEl = document.querySelector(`.group[data-gi="${drag.gi}"]`);
  if (sourceEl) sourceEl.classList.remove('dragging');
  document.body.style.userSelect = '';

  const group = state.groups[drag.gi];
  const occupied = occupiedCells(drag.gi);
  const dragW = group.w || 1;
  let blocked = false;
  for (let dx = 0; dx < dragW; dx++) {
    if (occupied.has(`${drag.targetX + dx},${drag.targetY}`)) { blocked = true; break; }
  }
  if (!blocked && (drag.targetX !== group.x || drag.targetY !== group.y)) {
    group.x = drag.targetX;
    group.y = drag.targetY;
    await saveConfig();
    render();
  }

  drag = null;
});

// ── Resize logic ───────────────────────────────────────────────────────────

let resize = null;

function startResize(gi, e) {
  e.preventDefault();
  document.body.style.userSelect = 'none';
  const handle = e.currentTarget;
  handle.classList.add('active');
  resize = { gi, handle, newW: state.groups[gi].w || 1 };
}

document.addEventListener('mousemove', (e) => {
  if (!resize) return;
  const group = state.groups[resize.gi];
  const gridEl = document.getElementById('groups');
  const gridRect = gridEl.getBoundingClientRect();
  const colWidth = gridRect.width / COLS;
  const targetCol = Math.floor((e.clientX - gridRect.left) / colWidth);
  let newW = Math.max(1, targetCol - group.x + 1);

  // Clamp to grid boundary
  newW = Math.min(newW, COLS - group.x);

  // Clamp to nearest group to the right at same row
  state.groups.forEach((g, i) => {
    if (i === resize.gi) return;
    if (g.y === group.y && g.x >= group.x + 1) {
      newW = Math.min(newW, g.x - group.x);
    }
    // Also check if spanning group occupies cells in our row
    for (let dx = 0; dx < (g.w || 1); dx++) {
      if (g.y === group.y && g.x + dx >= group.x + 1) {
        newW = Math.min(newW, g.x + dx - group.x);
      }
    }
  });
  newW = Math.max(1, newW);

  resize.newW = newW;
  const el = document.querySelector(`.group[data-gi="${resize.gi}"]`);
  if (el) el.style.gridColumn = `${group.x + 1} / span ${newW}`;
});

document.addEventListener('mouseup', async () => {
  if (!resize) return;
  resize.handle.classList.remove('active');
  document.body.style.userSelect = '';
  const group = state.groups[resize.gi];
  if (resize.newW !== (group.w || 1)) {
    group.w = resize.newW;
    await saveConfig();
    render();
  }
  resize = null;
});

// ── Edit mode ──────────────────────────────────────────────────────────────

document.getElementById('btn-edit').addEventListener('click', () => {
  document.body.classList.toggle('editing');
  const editing = document.body.classList.contains('editing');
  document.getElementById('btn-bg').classList.toggle('hidden', !editing);
  document.getElementById('add-group-row').classList.toggle('hidden', !editing);
});

// ── Delete helpers ─────────────────────────────────────────────────────────

async function deleteGroup(gi) {
  state.groups.splice(gi, 1);
  await saveConfig();
  render();
}

async function deleteLink(gi, li) {
  state.groups[gi].links.splice(li, 1);
  await saveConfig();
  render();
}

// ── Collapse ────────────────────────────────────────────────────────────────

async function toggleCollapse(gi) {
  state.groups[gi].collapsed = !state.groups[gi].collapsed;
  await saveConfig();
  render();
}

// ── Add group ──────────────────────────────────────────────────────────────

document.getElementById('btn-add-group').addEventListener('click', () => {
  const pos = findFreeCell();
  state.groups.push({ name: 'New Group', links: [], x: pos.x, y: pos.y, w: 2 });
  saveConfig().then(render);
});

// ── Link modal ─────────────────────────────────────────────────────────────

let linkModalCtx = null; // { gi, li }

function openLinkModal(gi, li) {
  linkModalCtx = { gi, li };
  const isNew = li === null;
  document.getElementById('modal-link-title').textContent = isNew ? 'Add Link' : 'Edit Link';
  if (!isNew) {
    const link = state.groups[gi].links[li];
    document.getElementById('link-name').value = link.name;
    document.getElementById('link-url').value = link.url;
    document.getElementById('link-icon').value = link.icon;
  } else {
    document.getElementById('link-name').value = '';
    document.getElementById('link-url').value = '';
    document.getElementById('link-icon').value = '';
  }
  document.getElementById('modal-link').classList.remove('hidden');
  document.getElementById('link-name').focus();
}

document.getElementById('modal-link-cancel').addEventListener('click', () => {
  document.getElementById('modal-link').classList.add('hidden');
});

document.getElementById('modal-link-save').addEventListener('click', async () => {
  const name = document.getElementById('link-name').value.trim();
  const url = document.getElementById('link-url').value.trim();
  const icon = document.getElementById('link-icon').value.trim();
  if (!name || !url) return;

  const { gi, li } = linkModalCtx;
  const link = { name, url, icon };
  if (li === null) {
    if (!state.groups[gi].links) state.groups[gi].links = [];
    state.groups[gi].links.push(link);
  } else {
    state.groups[gi].links[li] = link;
  }

  document.getElementById('modal-link').classList.add('hidden');
  await saveConfig();
  render();
});

// ── Group modal ────────────────────────────────────────────────────────────

let groupModalCtx = null;

function openGroupModal(gi) {
  groupModalCtx = gi;
  const group = state.groups[gi];
  document.getElementById('group-name').value = group.name;
  const color = group.color || '';
  document.getElementById('group-color-value').value = color;
  if (/^#[0-9a-fA-F]{6}$/.test(color)) {
    document.getElementById('group-color-picker').value = color;
  }
  document.getElementById('modal-group').classList.remove('hidden');
  document.getElementById('group-name').focus();
}

document.getElementById('modal-group-cancel').addEventListener('click', () => {
  document.getElementById('modal-group').classList.add('hidden');
});

document.getElementById('modal-group-save').addEventListener('click', async () => {
  const name = document.getElementById('group-name').value.trim();
  if (!name) return;
  state.groups[groupModalCtx].name = name;
  state.groups[groupModalCtx].color = document.getElementById('group-color-value').value.trim();
  document.getElementById('modal-group').classList.add('hidden');
  await saveConfig();
  render();
});

document.getElementById('group-color-picker').addEventListener('input', (e) => {
  document.getElementById('group-color-value').value = e.target.value;
});

document.getElementById('group-color-clear').addEventListener('click', () => {
  document.getElementById('group-color-value').value = '';
});

// ── Background modal ───────────────────────────────────────────────────────

document.getElementById('btn-bg').addEventListener('click', () => {
  const v = state.background || '#0f0f1a';
  document.getElementById('bg-value').value = v;
  if (/^#[0-9a-fA-F]{6}$/.test(v)) {
    document.getElementById('bg-color-picker').value = v;
  }
  document.getElementById('modal-bg').classList.remove('hidden');
});

document.getElementById('bg-color-picker').addEventListener('input', (e) => {
  document.getElementById('bg-value').value = e.target.value;
});

document.getElementById('bg-file').addEventListener('change', async (e) => {
  const file = e.target.files[0];
  if (!file) return;
  const fd = new FormData();
  fd.append('image', file);
  const res = await fetch('/api/upload/background', { method: 'POST', body: fd });
  const { url } = await res.json();
  document.getElementById('bg-value').value = `url(${url})`;
  document.body.style.backgroundImage = `url(${url})`;
  document.body.style.backgroundSize = 'cover';
  document.body.style.backgroundPosition = 'center';
});

document.getElementById('modal-bg-cancel').addEventListener('click', () => {
  document.getElementById('modal-bg').classList.add('hidden');
});

document.getElementById('modal-bg-save').addEventListener('click', async () => {
  const v = document.getElementById('bg-value').value.trim();
  if (!v) return;
  state.background = v;
  document.getElementById('modal-bg').classList.add('hidden');
  applyBackground();
  await saveConfig();
});

// ── Modal keyboard shortcuts ───────────────────────────────────────────────

document.addEventListener('keydown', (e) => {
  if (e.key === 'Escape') {
    document.querySelectorAll('.modal').forEach(m => m.classList.add('hidden'));
  }
  if (e.key === 'Enter') {
    const active = document.querySelector('.modal:not(.hidden)');
    if (!active) return;
    if (active.id === 'modal-link') document.getElementById('modal-link-save').click();
    if (active.id === 'modal-group') document.getElementById('modal-group-save').click();
    if (active.id === 'modal-bg') document.getElementById('modal-bg-save').click();
  }
});

// ── Init ───────────────────────────────────────────────────────────────────

loadConfig();
