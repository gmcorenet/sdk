(function(){
  const root = window.__crudRoot || (document.currentScript && document.currentScript.parentElement) || document;
  if (!root) return;
  if (root.getAttribute && root.getAttribute('data-crud-booted') === '1') return;
  const cfg = window.__crudConfig || {};
  const resource = String(cfg.resource || '').trim();
  const rootTheme = root.getAttribute && root.getAttribute('data-crud-theme');
  const themedNode = root.querySelector ? root.querySelector('[data-crud-theme]') : null;
  const activeTheme = rootTheme || (themedNode && themedNode.getAttribute('data-crud-theme')) || 'gmcore';
  function resolveRuntimeBasePath(resourceName, configuredBasePath){
    const configured = String(configuredBasePath || '').trim();
    const pathname = String((window.location && window.location.pathname) || '').trim();
    const suffix = '/' + String(resourceName || '').trim().replace(/^\/+|\/+$/g, '');
    if (pathname && suffix !== '/' && pathname === suffix) return pathname;
    if (pathname && suffix !== '/' && pathname.endsWith(suffix)) return pathname;
    if (configured) return configured;
    if (pathname) return pathname;
    return '/' + String(resourceName || '').trim().replace(/^\/+|\/+$/g, '');
  }
  const basePath = resolveRuntimeBasePath(resource, cfg.basePath);
  const primaryKey = String(cfg.primaryKey || 'id').trim();
  const formFields = Array.isArray(cfg.fields) ? cfg.fields : [];
  const formButtons = Array.isArray(cfg.buttons) ? cfg.buttons : [];
  const featuresRaw = cfg.features || {};
  const labels = cfg.labels || {};
  const features = {
    search: !!featuresRaw.search,
    filters: !!featuresRaw.filters,
    create: !!featuresRaw.create,
    view: !!featuresRaw.view,
    edit: !!featuresRaw.edit,
    clone: !!featuresRaw.clone,
    delete: !!featuresRaw.delete,
    bulk: !!featuresRaw.bulk,
    pagination: !!featuresRaw.pagination,
    perPage: !!(featuresRaw.perPage || featuresRaw.per_page),
    sort: !!featuresRaw.sort,
    designer: !!featuresRaw.designer
  };
  const headers = {'Content-Type': 'application/json'};
  if (cfg && cfg.csrfToken) headers['X-CSRF-Token'] = String(cfg.csrfToken);
  const crudIconRegistry = Object.assign({
    view: '/assets/crud/icons/eye.svg',
    edit: '/assets/crud/icons/edit.svg',
    clone: '/assets/crud/icons/copy.svg',
    delete: '/assets/crud/icons/trash.svg',
    create: '/assets/crud/icons/plus.svg',
    filter: '/assets/crud/icons/filter.svg',
    form: '/assets/crud/icons/form.svg',
    layout: '/assets/crud/icons/layout.svg',
    stack: '/assets/crud/icons/stack.svg',
    settings: '/assets/crud/icons/settings.svg',
    default: '/assets/crud/icons/form.svg'
  }, (cfg && cfg.iconRegistry && typeof cfg.iconRegistry === 'object' ? cfg.iconRegistry : {}));
  function crudIcon(name, fallback){
    const key = String(name || '').trim();
    const explicit = String(fallback || '').trim();
    if (explicit) return explicit;
    return crudIconRegistry[key] || crudIconRegistry.default;
  }
  const state = {
    instanceId: Math.random().toString(36).slice(2),
    page: 1,
    perPage: 10,
    totalPages: 1,
    q: '',
    sort: '',
    filters: {},
    filterFields: [],
    columns: [],
    selected: new Set(),
    pendingBulkAction: '',
    preferences: {
      perPage: 0
    },
    view: {
      columnWidths: {}
    },
    designer: {
      mode: 'create',
      metas: {},
      selectedField: ''
    }
  };
  window.GMCoreCrudIconRegistry = window.GMCoreCrudIconRegistry || {};
  window.GMCoreCrudIconRegistry[state.instanceId] = crudIconRegistry;
  function namespaceRuntimeIds(){
    if (!root.querySelectorAll) return;
    root.querySelectorAll('[id^="crud-"]').forEach(function(node){
      const original = String(node.getAttribute('data-crud-id') || node.id || '').trim();
      if (!original) return;
      node.setAttribute('data-crud-id', original);
      node.id = original + '--' + state.instanceId;
    });
    root.querySelectorAll('[data-crud-modal-open]').forEach(function(node){
      const target = String(node.getAttribute('data-crud-modal-open') || '').trim();
      if (!target || target.charAt(0) !== '#') return;
      const original = target.slice(1);
      node.setAttribute('data-crud-modal-open', '#' + original + '--' + state.instanceId);
    });
  }
  namespaceRuntimeIds();
  if (root.setAttribute) root.setAttribute('data-crud-booted', '1');
  if (root.setAttribute) root.setAttribute('data-crud-runtime-root', '1');
  if (root.setAttribute) root.setAttribute('data-crud-instance-id', state.instanceId);
  const storageKeyBase = 'gmcore-crud:' + (basePath || resource || 'default');
  const label = function(name, fallback){
    const value = labels && labels[name];
    return String(value == null || value === '' ? fallback : value);
  };
  function readStorage(name, fallback){
    try {
      const raw = window.localStorage.getItem(storageKeyBase + ':' + name);
      if (!raw) return fallback;
      const parsed = JSON.parse(raw);
      return parsed && typeof parsed === 'object' ? parsed : fallback;
    } catch (_err) {
      return fallback;
    }
  }
  function writeStorage(name, value){
    try {
      window.localStorage.setItem(storageKeyBase + ':' + name, JSON.stringify(value));
    } catch (_err) {}
  }
  state.view = readStorage('view', state.view);
  state.preferences = readStorage('preferences', state.preferences);
  if (!state.view || typeof state.view !== 'object') state.view = {columnWidths:{}};
  if (!state.view.columnWidths || typeof state.view.columnWidths !== 'object') state.view.columnWidths = {};
  if (state.preferences && Number(state.preferences.perPage || 0) > 0) {
    state.perPage = Number(state.preferences.perPage || 0);
  }
  function humanizeFieldName(name){
    return String(name || '')
      .replace(/[_-]+/g, ' ')
      .replace(/\s+/g, ' ')
      .trim()
      .replace(/\b\w/g, function(match){ return match.toUpperCase(); });
  }
  function displayFieldLabel(field){
    if (field && String(field.label || '').trim()) return String(field.label).trim();
    if (field && String(field.name || '').trim()) return humanizeFieldName(field.name);
    return '';
  }

  function byId(id){
    if (!root.querySelector) return null;
    const value = String(id || '');
    return root.querySelector('[data-crud-id="' + value.replace(/"/g, '\\"') + '"]') || root.querySelector('#' + value);
  }
  function all(selector){ return Array.prototype.slice.call(root.querySelectorAll ? root.querySelectorAll(selector) : []); }
  function portalID(name){ return String(name || '').trim() + '-' + state.instanceId; }
  function esc(value){
    return String(value == null ? '' : value)
      .replace(/&/g, '&amp;')
      .replace(/</g, '&lt;')
      .replace(/>/g, '&gt;')
      .replace(/"/g, '&quot;')
      .replace(/'/g, '&#39;');
  }
  function cleanErrorMessage(text, status){
    const raw = String(text == null ? '' : text).trim();
    if (raw) return raw;
    if (status === 403) return 'forbidden';
    return 'request failed';
  }
  async function fetchJSON(url, options){
    const response = await fetch(url, options || {});
    const text = await response.text();
    let data = {};
    if (text) {
      try { data = JSON.parse(text); } catch (_err) { data = text; }
    }
    if (!response.ok) {
      const error = new Error(cleanErrorMessage(text, response.status));
      error.status = response.status;
      error.payload = data;
      throw error;
    }
    return data;
  }
  function apiURL(path){
    return basePath.replace(/\/+$/, '') + path;
  }
  function dispatchMutation(kind, payload){
    try {
      window.dispatchEvent(new CustomEvent('gmcore:crud-mutated', {
        detail: {
          sourceBasePath: basePath,
          sourceResource: resource,
          sourceInstanceId: state.instanceId,
          kind: String(kind || '').trim(),
          payload: payload || {}
        }
      }));
    } catch (_err) {}
  }
  function floatingPortalHost(anchor){
    return document.body || document.documentElement || root;
  }
  function themeForNode(node){
    const themed = node && node.closest ? node.closest('[data-crud-theme]') : null;
    return (themed && themed.getAttribute('data-crud-theme')) || activeTheme;
  }
  function ensureFloatingPortal(name, themeName){
    const id = portalID(name);
    let portal = document.body ? document.body.querySelector('[data-crud-global-portal="' + id + '"]') : null;
    if (!portal && document.body) {
      portal = document.createElement('div');
      portal.setAttribute('data-crud-global-portal', id);
      portal.setAttribute('data-crud-portal-name', String(name || ''));
      portal.setAttribute('data-crud-instance-id', state.instanceId);
      portal.className = String(name || '').trim();
      document.body.appendChild(portal);
    }
    if (!portal) return null;
    if (themeName != null && String(themeName || '').trim()) {
      portal.setAttribute('data-crud-theme', String(themeName || activeTheme || 'gmcore'));
    } else if (!portal.getAttribute('data-crud-theme')) {
      portal.setAttribute('data-crud-theme', String(activeTheme || 'gmcore'));
    }
    return portal;
  }
  function isFloatingPortalOpen(name, anchorID){
    const portal = ensureFloatingPortal(name);
    if (!portal) return false;
    return portal.classList.contains('is-open') && (!anchorID || portal.getAttribute('data-anchor-id') === anchorID);
  }
  function closeFloatingPortal(name){
    const portal = ensureFloatingPortal(name);
    if (!portal) return;
    portal.classList.remove('is-open');
    portal.classList.remove('open');
    portal.innerHTML = '';
    portal.style.left = '';
    portal.style.top = '';
    portal.style.right = '';
    portal.style.bottom = '';
    portal.style.width = '';
    portal.style.visibility = '';
    portal.removeAttribute('data-open');
    portal.removeAttribute('data-anchor-id');
  }
  function closeAllFloatingPortals(){
    closeFloatingPortal('crud-floating-menu');
    closeFloatingPortal('crud-floating-relation');
    closeFloatingPortal('crud-floating-choice');
  }
  function positionFloatingPortal(name, anchor, options){
    const portal = ensureFloatingPortal(name, themeForNode(anchor));
    if (!portal || !anchor) return;
    const host = floatingPortalHost(anchor);
    if (host && portal.parentNode !== host) host.appendChild(portal);
    const rect = anchor.getBoundingClientRect();
    const hostRect = host.getBoundingClientRect ? host.getBoundingClientRect() : {left:0, top:0, width: window.innerWidth || 0, height: window.innerHeight || 0};
    const gap = Number(options && options.gap || 8) || 8;
    const minWidth = Number(options && options.minWidth || 220) || 220;
    const preferredWidth = Number(options && options.width || rect.width) || rect.width;
    portal.style.visibility = 'hidden';
    portal.classList.add(options && options.openClass || 'is-open');
    const menuRect = portal.getBoundingClientRect();
    const viewportWidth = window.innerWidth || hostRect.width || document.documentElement.clientWidth || 0;
    const viewportHeight = window.innerHeight || hostRect.height || document.documentElement.clientHeight || 0;
    const width = Math.max(minWidth, preferredWidth);
    let left = rect.left;
    if ((options && options.alignEnd) === true) {
      left = rect.right - Math.min(menuRect.width || width, width);
    }
    if (left + width > viewportWidth - gap) left = Math.max(gap, viewportWidth - width - gap);
    if (left < gap) left = gap;
    let top = rect.bottom + gap;
    if (!(options && options.noFlip) && top + menuRect.height > viewportHeight - gap) {
      top = Math.max(gap, rect.top - menuRect.height - gap);
    }
    portal.style.left = left + 'px';
    portal.style.top = top + 'px';
    portal.style.width = width + 'px';
    portal.style.visibility = '';
  }
  function openCrudPortal(options){
    const opts = options || {};
    const name = String(opts.name || '').trim();
    const anchor = opts.anchor || null;
    const portal = ensureFloatingPortal(name, opts.theme || themeForNode(anchor));
    if (!portal || !anchor) return null;
    if (opts.html != null) portal.innerHTML = String(opts.html);
    if (opts.anchorID != null) {
      portal.setAttribute('data-anchor-id', String(opts.anchorID || ''));
    } else if (opts.anchorAttr) {
      portal.setAttribute('data-anchor-id', anchor.getAttribute(opts.anchorAttr) || '');
    }
    if (opts.openData === true) portal.setAttribute('data-open', '1');
    portal.classList.add(opts.openClass || 'is-open');
    positionFloatingPortal(name, anchor, {
      alignEnd: opts.alignEnd,
      gap: opts.gap,
      minWidth: opts.minWidth,
      noFlip: opts.noFlip,
      openClass: opts.openClass || 'is-open',
      width: opts.width
    });
    if (typeof opts.restoreScrollTop === 'number' && opts.restoreScrollTop > 0) {
      window.requestAnimationFrame(function(){ portal.scrollTop = opts.restoreScrollTop; });
    }
    return portal;
  }
  function indexURL(){
    const url = new URL(apiURL('/api'), window.location.origin);
    if (state.page > 1) url.searchParams.set('page', String(state.page));
    if (state.perPage > 0) url.searchParams.set('per_page', String(state.perPage));
    if (state.q) url.searchParams.set('q', state.q);
    if (state.sort) url.searchParams.set('sort', state.sort);
    Object.keys(state.filters).forEach(function(name){
      const item = state.filters[name];
      if (!item || !item.value) return;
      url.searchParams.set('filter_' + name, item.value);
      if (item.op) url.searchParams.set('filter_' + name + '_op', item.op);
    });
    return url.toString();
  }
  function filterOperatorLabel(op){
    const value = String(op || 'eq').trim().toLowerCase();
    if (value === 'eq') return label('operator_eq', 'equals');
    if (value === 'ne' || value === 'neq' || value === 'not_eq') return label('operator_ne', 'is not');
    if (value === 'like') return label('operator_like', 'contains');
    if (value === 'not_like') return label('operator_not_like', 'does not contain');
    if (value === 'gt') return label('operator_gt', 'is greater than');
    if (value === 'gte') return label('operator_gte', 'is greater or equal to');
    if (value === 'lt') return label('operator_lt', 'is less than');
    if (value === 'lte') return label('operator_lte', 'is less or equal to');
    if (value === 'in') return label('operator_in', 'is in');
    if (value === 'null') return label('operator_null', 'is null');
    if (value === 'not_null') return label('operator_not_null', 'is not null');
    return value.replace(/_/g, ' ');
  }
  function findFilterField(name){
    return (state.filterFields || []).find(function(field){
      return String(field && field.name || '') === String(name || '');
    }) || null;
  }
  function activeFiltersSummary(){
    const items = [];
    if (String(state.q || '').trim()) {
      items.push(label('search_label', 'Search') + ' contains "' + String(state.q).trim() + '"');
    }
    return items.concat(Object.keys(state.filters).map(function(name){
      const item = state.filters[name];
      if (!item || !String(item.value || '').trim()) return '';
      const field = findFilterField(name);
      const fieldLabel = displayFieldLabel(field || {name:name}) || name;
      return fieldLabel + ' ' + filterOperatorLabel(item.op) + ' "' + String(item.value).trim() + '"';
    }).filter(Boolean));
  }
  function renderActiveFilterState(){
    const items = activeFiltersSummary();
    const hasFilters = items.length > 0;
    const filtersButton = byId('crud-open-filters');
    const resetFilters = root.querySelectorAll('[data-crud-reset-filters]');
    if (filtersButton) filtersButton.classList.toggle('is-active-filters', hasFilters);
    Array.prototype.forEach.call(resetFilters, function(node){
      node.classList.toggle('d-none', !hasFilters);
    });
    all('[data-crud-active-filters]').forEach(function(node){
      node.classList.toggle('d-none', !hasFilters);
      node.textContent = hasFilters ? items.join(' · ') : '';
    });
  }
  function renderResetViewState(){
    const resetView = byId('crud-reset-view');
    const hasCustomWidths = Object.keys(state.view.columnWidths || {}).some(function(key){
      return Number(state.view.columnWidths[key] || 0) > 0;
    });
    if (resetView) resetView.classList.toggle('d-none', !hasCustomWidths);
  }
  function columnKey(col){
    return String(col && (col.field || col.name || col.sort_expression) || '').trim();
  }
  function columnWidth(field){
    const value = Number(state.view.columnWidths[field] || 0);
    return value > 0 ? value : 0;
  }
  function renderColGroup(model){
    const root = byId('crud-colgroup');
    if (!root) return;
    const parts = [];
    if (features.bulk) parts.push('<col style="width:48px">');
    (model.columns || []).forEach(function(col){
      const key = columnKey(col);
      const width = columnWidth(key);
      parts.push('<col data-column-field="' + esc(key) + '"' + (width ? ' style="width:' + width + 'px"' : '') + '>');
    });
    parts.push('<col style="width:116px">');
    root.innerHTML = parts.join('');
  }
  function applyColumnWidthsToCells(){
    const head = byId('crud-head');
    const body = byId('crud-body');
    if (!head || !body) return;
    const headers = Array.prototype.slice.call(head.children || []);
    headers.forEach(function(th, index){
      const field = th.getAttribute('data-column-field') || '';
      const width = field ? columnWidth(field) : 0;
      th.style.width = width ? (width + 'px') : '';
      th.style.minWidth = width ? (width + 'px') : '';
      Array.prototype.slice.call(body.querySelectorAll('tr')).forEach(function(row){
        const cell = row.children[index];
        if (!cell) return;
        cell.style.width = width ? (width + 'px') : '';
        cell.style.minWidth = width ? (width + 'px') : '';
      });
    });
  }
  function setColumnWidth(field, width){
    const safeWidth = Math.max(96, Math.min(720, Number(width || 0) || 0));
    state.view.columnWidths[field] = safeWidth;
    writeStorage('view', state.view);
    renderColGroup({columns: state.columns || []});
    applyColumnWidthsToCells();
    renderResetViewState();
  }
  function resetView(){
    state.view.columnWidths = {};
    writeStorage('view', state.view);
    renderColGroup({columns: state.columns || []});
    applyColumnWidthsToCells();
    renderResetViewState();
  }
  function renderHead(model){
    const head = byId('crud-head');
    if (!head) return;
    state.columns = model.columns || [];
    renderColGroup(model);
    head.innerHTML =
      (features.bulk ? '<th class="crud-col-select"><input type="checkbox" id="crud-select-all--' + state.instanceId + '" data-crud-id="crud-select-all"></th>' : '') +
      (model.columns || []).map(function(col){
        const nextLabel = esc(displayFieldLabel(col) || col.field);
        const field = columnKey(col);
        const priority = Math.max(1, Number(col.priority || 0) || 1);
        const resize = field ? '<span class="crud-col-resize-handle" data-column-resize="' + esc(field) + '" title="' + esc(label('resize_column', 'Resize column')) + '"></span>' : '';
        if (col.sortable && features.sort) {
          const sortExpr = String(col.sort_expression || col.field || '');
          const asc = sortExpr;
          const desc = '-' + sortExpr;
          const current = String(state.sort || '');
          const isAsc = current === asc;
          const isDesc = current === desc;
          const nextSort = isAsc ? desc : asc;
        const sortLabel = '';
        const sortState = isAsc ? ' is-asc' : (isDesc ? ' is-desc' : '');
        const sortIcon = isAsc ? '↑' : (isDesc ? '↓' : '↕');
          return '<th class="crud-sort-head crud-col-head" data-column-field="' + esc(field) + '" data-col-priority="' + esc(priority) + '"><div class="crud-col-head-inner"><button type="button" class="crud-btn-link crud-btn-sm crud-sort' + sortState + '" data-sort="' + esc(nextSort) + '" data-sort-base="' + esc(sortExpr) + '"><span class="crud-sort-text">' + nextLabel + '</span><span class="crud-sort-meta"><span class="crud-sort-direction">' + sortIcon + '</span></span></button>' + resize + '</div></th>';
        }
        return '<th class="crud-col-head" data-column-field="' + esc(field) + '" data-col-priority="' + esc(priority) + '"><div class="crud-col-head-inner"><span class="crud-col-label">' + nextLabel + '</span>' + resize + '</div></th>';
      }).join('') +
      '<th class="crud-col-actions">Actions</th>';
  }
  function formatCellValue(value){
    const text = String(value == null ? '' : value);
    if (text.length <= 72) return '<span class="crud-cell-text">' + esc(text) + '</span>';
    return '<span class="crud-cell"><span class="crud-cell-text" title="' + esc(text) + '">' + esc(text.slice(0, 72)) + '…</span></span>';
  }
  function actionMenu(row){
    const actions = (row.actions || []).filter(function(action){
      if (action.name === 'view') return features.view;
      if (action.name === 'edit') return features.edit;
      if (action.name === 'clone') return features.clone;
      if (action.name === 'delete') return features.delete;
      return true;
    });
    if (!actions.length) return '';
    const menuItems = actions.map(function(action){
      return {
        name: String(action.name || ''),
        label: String(action.label || action.name || ''),
        kind: String(action.kind || ''),
        method: String(action.method || ''),
        url: String(action.url || ''),
        target: String(action.target || ''),
        group: String(action.group || ''),
        groupLabel: String(action.group_label || action.groupLabel || action.group || ''),
        separator: !!action.separator,
        separatorBefore: !!(action.separator_before || action.separatorBefore),
        separatorAfter: !!(action.separator_after || action.separatorAfter),
        confirmText: String(action.confirm_text || ''),
        key: String(row.key || ''),
        icon: crudIcon(action.name, action.icon)
      };
    });
    return '<div class="crud-row-actions" data-crud-menu>' +
      '<button type="button" class="crud-btn crud-btn-outline-secondary crud-btn-sm crud-row-menu-toggle" data-crud-menu-toggle data-crud-menu-items="' + esc(JSON.stringify(menuItems)) + '" aria-expanded="false" aria-haspopup="true">' +
        '<span>Actions</span><img class="crud-icon crud-icon-caret" src="/assets/crud/icons/caret-down.svg" alt="">' +
      '</button>' +
    '</div>';
  }
  function renderBody(model){
    const body = byId('crud-body');
    if (!body) return;
    const columns = model.columns || [];
    const colspan = Math.max(1, columns.length + (features.bulk ? 1 : 0) + 1);
    state.colspan = colspan;
    if (!(model.rows || []).length) {
      const emptyTemplate = byId('crud-empty-template');
      const emptyMarkup = emptyTemplate ? emptyTemplate.innerHTML : '<div class="crud-empty-state"><div class="crud-empty-title">' + esc(label('empty', 'No records available yet.')) + '</div></div>';
      body.innerHTML = '<tr><td colspan="' + colspan + '" class="p-0">' + emptyMarkup + '</td></tr>';
      return;
    }
    body.innerHTML = model.rows.map(function(row){
      const cells = (row.cells || []).map(function(cell, index){
        const col = columns[index] || {};
        const labelText = displayFieldLabel({label: col.label, name: col.field}) || col.field || '';
        const priority = Math.max(1, Number(col.priority || 0) || 1);
        return '<td data-col-label="' + esc(labelText) + '" data-col-priority="' + esc(priority) + '">' + formatCellValue(cell.value) + '</td>';
      }).join('');
      return '<tr>' +
        (features.bulk ? '<td class="crud-col-select"><input class="form-check-input crud-select" type="checkbox" value="' + esc(row.key) + '"></td>' : '') +
        cells + '<td class="crud-col-actions" data-col-label="Actions" data-col-priority="1">' + actionMenu(row) + '</td></tr>';
    }).join('');
    applyColumnWidthsToCells();
  }
  function renderFooter(model){
    const total = (model.result && model.result.total) || 0;
    const currentPage = (model.result && model.result.page) || 1;
    const perPage = (model.result && model.result.per_page) || state.perPage;
    state.totalPages = (model.result && model.result.total_pages) || 1;
    state.perPage = perPage;
    state.preferences.perPage = perPage;
    writeStorage('preferences', state.preferences);
    const rangeStart = total === 0 ? 0 : (((currentPage - 1) * perPage) + 1);
    const rangeEnd = total === 0 ? 0 : Math.min(total, rangeStart + perPage - 1);
    const summaryText = label('summary_entries', 'Entries') + ' ' + rangeStart + '–' + rangeEnd + ' ' + label('summary_of', 'of') + ' ' + total + ' · ' + label('summary_page', 'Page') + ' ' + currentPage + ' ' + label('summary_of', 'of') + ' ' + state.totalPages;

    all('[data-crud-total]').forEach(function(node){ node.textContent = total + ' ' + label('summary_total', 'total'); });
    all('[data-crud-summary]').forEach(function(node){ node.textContent = summaryText; });
    all('[data-crud-page-input]').forEach(function(node){
      node.min = '1';
      node.max = String(state.totalPages);
      node.value = String(currentPage);
    });
    all('[data-crud-page-jump]').forEach(function(node){
      node.classList.toggle('d-none', !features.pagination);
    });
    all('[data-crud-prev]').forEach(function(node){ node.disabled = currentPage <= 1; });
    all('[data-crud-next]').forEach(function(node){ node.disabled = currentPage >= state.totalPages; });
    all('[data-crud-first]').forEach(function(node){ node.disabled = currentPage <= 1; });
    all('[data-crud-last]').forEach(function(node){ node.disabled = currentPage >= state.totalPages; });

    const options = [10, 25, 50, 100];
    const perPageHTML = options.map(function(option){
        const selected = option === perPage ? ' selected' : '';
        return '<option value="' + option + '"' + selected + '>' + option + ' / page</option>';
      }).join('');
    all('[data-crud-per-page]').forEach(function(node){
      node.innerHTML = perPageHTML;
    });

    const bulkHTML = '<option value="">' + esc(label('bulk', 'Bulk actions')) + '</option>' + (model.bulk_actions || []).map(function(action){
        return '<option value="' + esc(action.name) + '">' + esc(action.label || action.name) + '</option>';
      }).join('');
    all('[data-crud-bulk-action]').forEach(function(node){
      const currentValue = node.value || '';
      node.innerHTML = bulkHTML;
      if (currentValue) node.value = currentValue;
    });

    const numbers = [];
    const start = Math.max(1, currentPage - 2);
    const end = Math.min(state.totalPages, currentPage + 2);
    if (start > 1) {
      numbers.push(1);
      if (start > 2) numbers.push('ellipsis');
    }
    for (let page = start; page <= end; page += 1) numbers.push(page);
    if (end < state.totalPages) {
      if (end < state.totalPages - 1) numbers.push('ellipsis');
      numbers.push(state.totalPages);
    }
    const numbersHTML = numbers.map(function(item){
        if (item === 'ellipsis') return '<span class="crud-page-ellipsis">…</span>';
        const klass = item === currentPage ? ' is-active' : '';
        return '<button type="button" class="crud-btn crud-btn-sm crud-page-btn' + klass + '" data-page-go="' + item + '">' + item + '</button>';
      }).join('');
    all('[data-crud-page-numbers]').forEach(function(node){
      node.innerHTML = numbersHTML;
    });
    renderActiveFilterState();
    renderResetViewState();
  }
  function renderFilters(model){
    const root = byId('crud-filters-list');
    if (!root) return;
    state.filterFields = model.filter_fields || [];
    root.innerHTML = (model.filter_fields || []).map(function(field){
      const current = state.filters[field.name] || {};
      const operators = (field.filter_operators || ['eq', 'like']).map(function(op){
        return '<option value="' + esc(op) + '"' + (String(current.op || '') === String(op) ? ' selected' : '') + '>' + esc(filterOperatorLabel(op)) + '</option>';
      }).join('');
      let input = '<input class="crud-input crud-input-sm" data-filter-value="' + esc(field.name) + '" placeholder="' + esc(field.placeholder || '') + '" value="' + esc(current.value || '') + '">';
      if (field.type === 'bool' || field.type === 'boolean') {
        input = '<select class="crud-select crud-select-sm" data-filter-value="' + esc(field.name) + '"><option value=""></option><option value="true"' + (String(current.value || '') === 'true' ? ' selected' : '') + '>true</option><option value="false"' + (String(current.value || '') === 'false' ? ' selected' : '') + '>false</option></select>';
      } else if (field.type === 'date') {
        input = '<input type="date" class="crud-input crud-input-sm" data-filter-value="' + esc(field.name) + '" value="' + esc(current.value || '') + '">';
      } else if (field.type === 'datetime') {
        input = '<input type="datetime-local" class="crud-input crud-input-sm" data-filter-value="' + esc(field.name) + '" value="' + esc(current.value || '') + '">';
      }
      return '<tr>' +
        '<th>' + esc(displayFieldLabel(field) || field.name) + '</th>' +
        '<td class="crud-filter-operator-cell"><select class="crud-select crud-select-sm" data-filter-op="' + esc(field.name) + '">' + operators + '</select></td>' +
        '<td class="crud-filter-value-cell">' + input + '</td>' +
      '</tr>';
    }).join('');
  }
  function fieldSpanClass(field){
    const span = Number(field.col_span || field.colSpan || 6) || 6;
    return 'span-' + Math.min(12, Math.max(1, span));
  }
  function asArray(value){
    if (Array.isArray(value)) return value.map(String);
    if (typeof value === 'string') {
      const trimmed = value.trim();
      if (!trimmed) return [];
      if (trimmed[0] === '[') {
        try {
          const parsed = JSON.parse(trimmed);
          if (Array.isArray(parsed)) return parsed.map(String);
        } catch (_err) {}
      }
      return trimmed.split(',').map(function(item){ return item.trim(); }).filter(Boolean);
    }
    return [];
  }
  function boolValue(value){
    if (typeof value === 'boolean') return value;
    const text = String(value == null ? '' : value).trim().toLowerCase();
    return text === '1' || text === 'true' || text === 'yes' || text === 'on';
  }
