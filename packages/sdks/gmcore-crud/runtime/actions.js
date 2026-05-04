
  async function openAction(name, key, url, item){
    item = item || {};
    try {
      if (name === 'view' || name === 'edit' || name === 'clone') {
        await openExisting(name, key, url ? (url + (url.indexOf('?') >= 0 ? '&' : '?') + 'mode=' + encodeURIComponent(name)) : '');
        return;
      }
      if (name === 'delete') {
        await deleteRecord(key);
        return;
      }
      const confirmText = String(item.confirmText || item.confirm_text || '').trim();
      if (confirmText && !window.confirm(confirmText)) return;
      if (isServerAction(item)) {
        const result = await runServerAction(name, key);
        handleServerActionResult(name, key, result);
        return;
      }
      if (url && url !== '#') {
        const target = String(item.target || '').trim();
        if (target === '_self' || target === 'self') {
          window.location.href = url;
          return;
        }
        window.open(url, target || '_blank');
      }
    } catch (err) {
      window.alert(String(err && err.message || err));
    }
  }
  function isServerAction(item){
    const kind = String(item && item.kind || '').trim().toLowerCase();
    const method = String(item && item.method || '').trim().toLowerCase();
    return kind === 'serveraction' || kind === 'server_action' || kind === 'server-action' || method === 'serveraction' || method === 'server_action' || method === 'server-action';
  }
  async function runServerAction(name, key){
    return fetchJSON(apiURL('/api/action'), {
      method: 'POST',
      credentials: 'same-origin',
      headers: headers,
      body: JSON.stringify({action:name, key:key})
    });
  }
  function handleServerActionResult(name, key, result){
    result = result && typeof result === 'object' ? result : {};
    if (result.message) window.alert(String(result.message));
    if (result.redirect) {
      window.location.href = String(result.redirect);
      return;
    }
    dispatchMutation('server_action', {action:name, recordKey:key, result:result});
    if (result.reload !== false) {
      loadPage();
    }
  }
  function closeFloatingMenu(){
    closeFloatingPortal('crud-floating-menu');
    root.querySelectorAll('[data-crud-menu-toggle][aria-expanded="true"]').forEach(function(node){
      node.setAttribute('aria-expanded', 'false');
    });
  }
  function openFloatingMenu(toggle){
    if (!toggle) return;
    let items = [];
    try {
      items = JSON.parse(toggle.getAttribute('data-crud-menu-items') || '[]');
    } catch (_err) {
      items = [];
    }
    if (!Array.isArray(items) || !items.length) {
      closeFloatingMenu();
      return;
    }
    if (!toggle.getAttribute('data-crud-menu-anchor')) {
      toggle.setAttribute('data-crud-menu-anchor', 'crud-anchor-' + Math.random().toString(36).slice(2));
    }
    const anchorID = toggle.getAttribute('data-crud-menu-anchor') || '';
    const menu = ensureFloatingPortal('crud-floating-menu', themeForNode(toggle));
    if (!menu) return;
    const isSameAnchorOpen = menu.getAttribute('data-open') === '1' && menu.getAttribute('data-anchor-id') === anchorID;
    if (isSameAnchorOpen) {
      closeFloatingMenu();
      return;
    }
    let lastGroup = '';
    const html = items.map(function(item){
      const group = String(item.group || '').trim();
      const groupLabel = String(item.groupLabel || item.group_label || group).trim();
      let itemHTML = '';
      if (group && group !== lastGroup) {
        if (lastGroup) itemHTML += '<div class="crud-action-separator" role="separator"></div>';
        itemHTML += '<div class="crud-action-group" role="presentation">' + esc(groupLabel || group) + '</div>';
        lastGroup = group;
      }
      if (item.separator || item.separatorBefore) itemHTML += '<div class="crud-action-separator" role="separator"></div>';
      if (item.separator && !item.name) return itemHTML;
      itemHTML += '<button type="button" class="crud-action-item" data-row-action="' + esc(item.name) + '" data-row-key="' + esc(item.key) + '" data-row-url="' + esc(item.url) + '" data-row-action-config="' + esc(JSON.stringify(item)) + '">' +
        '<span class="crud-action-label"><img class="crud-icon" src="' + esc(item.icon) + '" alt=""><span>' + esc(item.label || item.name) + '</span></span>' +
      '</button>';
      if (item.separatorAfter) itemHTML += '<div class="crud-action-separator" role="separator"></div>';
      return itemHTML;
    }).join('');
    const opened = openCrudPortal({
      name: 'crud-floating-menu',
      anchor: toggle,
      html: html,
      anchorID: anchorID,
      openData: true,
      minWidth: 196,
      width: 196,
      openClass: 'open',
      noFlip: true
    });
    if (!opened) return;
    root.querySelectorAll('[data-crud-menu-toggle][aria-expanded="true"]').forEach(function(node){
      if (node !== toggle) node.setAttribute('aria-expanded', 'false');
    });
    toggle.setAttribute('aria-expanded', 'true');
    Array.prototype.slice.call(opened.querySelectorAll('[data-row-action]')).forEach(function(btn){
      btn.onclick = function(){
        let item = {};
        try {
          item = JSON.parse(btn.getAttribute('data-row-action-config') || '{}') || {};
        } catch (_err) {
          item = {};
        }
        closeFloatingMenu();
        openAction(btn.getAttribute('data-row-action') || '', btn.getAttribute('data-row-key') || '', btn.getAttribute('data-row-url') || '', item);
      };
    });
  }
  function showBulkConfirm(action, count){
    state.pendingBulkAction = action || '';
    const meta = byId('crud-bulk-confirm-meta');
    if (meta) meta.textContent = count + ' selected · ' + action;
    const modalEl = byId('crud-bulk-confirm-modal');
    const instance = window.gmcoreCrudModal && window.gmcoreCrudModal.getOrCreateInstance ? window.gmcoreCrudModal.getOrCreateInstance(modalEl) : null;
    if (instance) instance.show();
  }
  function hideBulkConfirm(){
    state.pendingBulkAction = '';
    const meta = byId('crud-bulk-confirm-meta');
    if (meta) meta.textContent = '';
    const modalEl = byId('crud-bulk-confirm-modal');
    const instance = window.gmcoreCrudModal && window.gmcoreCrudModal.getOrCreateInstance ? window.gmcoreCrudModal.getOrCreateInstance(modalEl) : null;
    if (instance) instance.hide();
  }
  async function submitBulkAction(){
    const action = state.pendingBulkAction;
    const keys = Array.from(state.selected);
    if (!action || !keys.length) return;
    const response = await fetch(apiURL('/api/bulk'), {method:'POST', credentials:'same-origin', headers:headers, body:JSON.stringify({action:action, keys:keys})});
    if (!response.ok) {
      const text = await response.text();
      window.alert(cleanErrorMessage(text, response.status));
      return;
    }
    hideBulkConfirm();
    const contentType = response.headers.get('Content-Type') || '';
    if (contentType.indexOf('application/json') === -1) {
      const blob = await response.blob();
      const href = URL.createObjectURL(blob);
      const link = document.createElement('a');
      link.href = href;
      link.download = resource + '-' + action;
      link.click();
      URL.revokeObjectURL(href);
      return;
    }
    state.selected.clear();
    const selectAll = byId('crud-select-all');
    if (selectAll) selectAll.checked = false;
    await load();
    dispatchMutation('bulk', {action: action, keys: keys});
  }
  function bind(model){
    root.querySelectorAll('[data-column-resize]').forEach(function(handle){
      handle.onmousedown = function(event){
        event.preventDefault();
        event.stopPropagation();
        const field = handle.getAttribute('data-column-resize') || '';
        if (!field) return;
        const th = handle.closest('[data-column-field]');
        if (!th) return;
        const startX = event.clientX;
        const startWidth = Math.round(th.getBoundingClientRect().width);
        function onMove(moveEvent){
          setColumnWidth(field, startWidth + (moveEvent.clientX - startX));
        }
        function onUp(){
          document.removeEventListener('mousemove', onMove);
          document.removeEventListener('mouseup', onUp);
        }
        document.addEventListener('mousemove', onMove);
        document.addEventListener('mouseup', onUp);
      };
    });
    root.querySelectorAll('[data-crud-menu-toggle]').forEach(function(btn){
      btn.onclick = function(event){
        event.stopPropagation();
        openFloatingMenu(btn);
      };
    });
    const selectAll = byId('crud-select-all');
    if (selectAll) {
      selectAll.onchange = function(){
        root.querySelectorAll('.crud-select').forEach(function(cb){
          cb.checked = selectAll.checked;
          if (cb.checked) state.selected.add(cb.value); else state.selected.delete(cb.value);
        });
      };
    }
    root.querySelectorAll('.crud-select').forEach(function(cb){
      cb.onchange = function(){
        if (cb.checked) state.selected.add(cb.value); else state.selected.delete(cb.value);
      };
    });
    root.querySelectorAll('.crud-sort').forEach(function(btn){
      btn.onclick = function(){
        state.sort = btn.getAttribute('data-sort') || '';
        state.page = 1;
        load();
      };
    });
    root.querySelectorAll('[data-page-go]').forEach(function(btn){
      btn.onclick = function(){ state.page = parseInt(btn.getAttribute('data-page-go') || '1', 10) || 1; load(); };
    });
    root.querySelectorAll('[data-row-action]').forEach(function(btn){
      btn.onclick = function(){
        let item = {};
        try {
          item = JSON.parse(btn.getAttribute('data-row-action-config') || '{}') || {};
        } catch (_err) {
          item = {};
        }
        closeFloatingMenu();
        openAction(btn.getAttribute('data-row-action') || '', btn.getAttribute('data-row-key') || '', btn.getAttribute('data-row-url') || '', item);
      };
    });
    root.querySelectorAll('[data-crud-bulk-apply]').forEach(function(bulkButton){
      bulkButton.onclick = function(){
        const select = Array.prototype.slice.call(root.querySelectorAll('[data-crud-bulk-action]')).find(function(node){ return node.value; }) || root.querySelector('[data-crud-bulk-action]');
        const action = select ? select.value : '';
        const keys = Array.from(state.selected);
        if (!action || !keys.length) return;
        showBulkConfirm(action, keys.length);
      };
    });
    root.querySelectorAll('[data-crud-bulk-action]').forEach(function(select){
      select.onchange = function(){
        root.querySelectorAll('[data-crud-bulk-action]').forEach(function(other){
          if (other !== select) other.value = select.value;
        });
      };
    });
    root.querySelectorAll('[data-crud-page-apply]').forEach(function(button){
      button.onclick = function(){
        const shell = button.closest('[data-crud-page-jump]');
        const input = shell ? shell.querySelector('[data-crud-page-input]') : null;
        const nextPage = Math.max(1, Math.min(state.totalPages, parseInt(input && input.value || '1', 10) || 1));
        state.page = nextPage;
        load();
      };
    });
    root.querySelectorAll('[data-crud-page-input]').forEach(function(input){
      input.onkeydown = function(event){
        if (event.key === 'Enter') {
          event.preventDefault();
          const nextPage = Math.max(1, Math.min(state.totalPages, parseInt(input.value || '1', 10) || 1));
          state.page = nextPage;
          load();
        }
      };
    });
    const bulkConfirm = byId('crud-bulk-confirm-submit');
    if (bulkConfirm) {
      bulkConfirm.onclick = async function(){
        try {
          await submitBulkAction();
        } catch (err) {
          window.alert(String(err && err.message || err));
        }
      };
    }
    root.querySelectorAll('[data-crud-per-page]').forEach(function(select){
      select.onchange = function(){
        state.perPage = parseInt(select.value || '10', 10) || 10;
        state.preferences.perPage = state.perPage;
        writeStorage('preferences', state.preferences);
        state.page = 1;
        root.querySelectorAll('[data-crud-per-page]').forEach(function(other){
          if (other !== select) other.value = select.value;
        });
        load();
      };
    });
    root.querySelectorAll('[data-crud-first]').forEach(function(btn){
      btn.onclick = function(){ if (state.page > 1) { state.page = 1; load(); } };
    });
    root.querySelectorAll('[data-crud-prev]').forEach(function(btn){
      btn.onclick = function(){ if (state.page > 1) { state.page -= 1; load(); } };
    });
    root.querySelectorAll('[data-crud-next]').forEach(function(btn){
      btn.onclick = function(){ if (state.page < state.totalPages) { state.page += 1; load(); } };
    });
    root.querySelectorAll('[data-crud-last]').forEach(function(btn){
      btn.onclick = function(){ if (state.page < state.totalPages) { state.page = state.totalPages; load(); } };
    });
    root.querySelectorAll('[data-crud-reset-filters]').forEach(function(button){
      button.onclick = function(){
        state.filters = {};
        state.q = '';
        const search = byId('crud-search');
        if (search) search.value = '';
        state.page = 1;
        load();
      };
    });
    const resetViewButton = byId('crud-reset-view');
    if (resetViewButton) {
      resetViewButton.onclick = function(){
        resetView();
      };
    }
  }
  function applyFeatureVisibility(){
    const mapping = [
      ['crud-search-shell', features.search],
      ['crud-open-filters', features.filters],
      ['[data-crud-reset-filters]', features.filters],
      ['crud-open-create', features.create],
      ['crud-open-designer', features.designer],
      ['[data-crud-bulk-action]', features.bulk],
      ['[data-crud-bulk-apply]', features.bulk],
      ['[data-crud-per-page]', features.perPage],
      ['[data-crud-pagination-group]', features.pagination]
    ];
    mapping.forEach(function(item){
      if (item[0][0] === '[') {
        root.querySelectorAll(item[0]).forEach(function(node){
          node.classList.toggle('d-none', !item[1]);
        });
        return;
      }
      const node = byId(item[0]);
      if (node) node.classList.toggle('d-none', !item[1]);
    });
    renderActiveFilterState();
    renderResetViewState();
  }
  async function load(){
    try {
      closeFloatingMenu();
      const model = await fetchJSON(indexURL(), {credentials:'same-origin'});
      renderFilters(model);
      renderHead(model);
      renderBody(model);
      renderFooter(model);
      bind(model);
    } catch (err) {
      const body = byId('crud-body');
      const colspan = Math.max(1, Number(state.colspan || 0) || (features.bulk ? 3 : 2));
      if (body) body.innerHTML = '<tr><td colspan="' + colspan + '" class="p-4 text-danger">' + esc(String(err && err.message || err)) + '</td></tr>';
    }
  }
