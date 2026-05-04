  function clearDesignerResizeState(root){
    if (state.designer.resizeMoveHandler) {
      document.removeEventListener('pointermove', state.designer.resizeMoveHandler, true);
      state.designer.resizeMoveHandler = null;
    }
    if (state.designer.resizeUpHandler) {
      document.removeEventListener('pointerup', state.designer.resizeUpHandler, true);
      document.removeEventListener('pointercancel', state.designer.resizeUpHandler, true);
      state.designer.resizeUpHandler = null;
    }
    state.designer.resizePointerId = null;
    state.designer.resizeFieldName = '';
    state.designer.resizeMode = '';
    state.designer.resizeSpan = 0;
    state.designer.resizeSignature = '';
    state.designer.resizeStartX = 0;
    state.designer.resizeStartY = 0;
    state.designer.resizeStartSpan = 0;
    state.designer.resizeStartHeight = 0;
    if (root) {
      root.classList.remove('is-resizing');
      root.querySelectorAll('.crud-designer-widget').forEach(function(node){
        node.classList.remove('is-resizing');
      });
      clearDesignerGuides(root);
    }
  }
  function ensureDesignerGuides(root){
    if (!root) return null;
    let vertical = root.querySelector('[data-crud-designer-guide="vertical"]');
    let horizontal = root.querySelector('[data-crud-designer-guide="horizontal"]');
    if (!vertical) {
      vertical = document.createElement('div');
      vertical.className = 'crud-designer-guide crud-designer-guide-vertical';
      vertical.setAttribute('data-crud-designer-guide', 'vertical');
      root.appendChild(vertical);
    }
    if (!horizontal) {
      horizontal = document.createElement('div');
      horizontal.className = 'crud-designer-guide crud-designer-guide-horizontal';
      horizontal.setAttribute('data-crud-designer-guide', 'horizontal');
      root.appendChild(horizontal);
    }
    return {vertical: vertical, horizontal: horizontal};
  }
  function clearDesignerGuides(root){
    if (!root) return;
    const guides = ensureDesignerGuides(root);
    if (!guides) return;
    guides.vertical.classList.remove('is-visible');
    guides.horizontal.classList.remove('is-visible');
  }
  function updateDesignerGuides(root, item, span, height){
    if (!root || !item) return;
    const guides = ensureDesignerGuides(root);
    if (!guides) return;
    const rootRect = root.getBoundingClientRect();
    let widthMatch = false;
    let heightMatch = false;
    root.querySelectorAll('[data-crud-designer-item]').forEach(function(node){
      if (node === item) return;
      const nodeSpan = Number(node.getAttribute('data-crud-current-span') || 0) || 0;
      const nodeHeight = Number(node.getAttribute('data-crud-current-height') || 0) || 0;
      if (nodeSpan === span) widthMatch = true;
      if (nodeHeight === height) heightMatch = true;
    });
    if (widthMatch) {
      const rect = item.getBoundingClientRect();
      guides.vertical.style.left = Math.round(rect.right - rootRect.left) + 'px';
      guides.vertical.style.top = '0px';
      guides.vertical.style.height = Math.round(rootRect.height) + 'px';
      guides.vertical.classList.add('is-visible');
    } else {
      guides.vertical.classList.remove('is-visible');
    }
    if (heightMatch) {
      const rect = item.getBoundingClientRect();
      guides.horizontal.style.top = Math.round(rect.bottom - rootRect.top) + 'px';
      guides.horizontal.style.left = '0px';
      guides.horizontal.style.width = Math.round(rootRect.width) + 'px';
      guides.horizontal.classList.add('is-visible');
    } else {
      guides.horizontal.classList.remove('is-visible');
    }
  }
  function updateDesignerGhostPosition(event){
    const ghost = state.designer.dragGhost;
    if (!ghost) return;
    ghost.style.left = Math.round(Number(event.clientX || 0) + 18) + 'px';
    ghost.style.top = Math.round(Number(event.clientY || 0) + 18) + 'px';
  }
  function gridMetrics(grid){
    if (!grid) return null;
    const rect = grid.getBoundingClientRect();
    const styles = window.getComputedStyle ? window.getComputedStyle(grid) : null;
    const gap = styles ? Number(parseFloat(styles.columnGap || styles.gap || '0')) || 0 : 0;
    const totalGap = gap * 11;
    const columnWidth = Math.max(1, (rect.width - totalGap) / 12);
    return {rect: rect, gap: gap, columnWidth: columnWidth};
  }
  function spanStepForItem(item){
    const grid = item && item.parentElement;
    const metrics = gridMetrics(grid);
    if (!metrics) return 1;
    return Math.max(1, metrics.columnWidth + metrics.gap);
  }
  function resizeHeightStep(){
    return 10;
  }
  function captureDesignerFlipState(){
    const root = byId('crud-designer-canvas');
    if (!root) return;
    const rects = {};
    root.querySelectorAll('[data-crud-designer-item]').forEach(function(node){
      const name = String(node.getAttribute('data-crud-designer-name') || '');
      if (!name) return;
      const rect = node.getBoundingClientRect();
      rects[name] = {
        left: rect.left,
        top: rect.top
      };
    });
    state.designer.flipRects = rects;
  }
  function playDesignerFlip(root){
    const rects = state.designer.flipRects || null;
    state.designer.flipRects = null;
    if (!root || !rects) return;
    root.querySelectorAll('[data-crud-designer-item]').forEach(function(node){
      const name = String(node.getAttribute('data-crud-designer-name') || '');
      if (!name || name === state.designer.dragFieldName) return;
      const previous = rects[name];
      if (!previous) return;
      const rect = node.getBoundingClientRect();
      const deltaX = previous.left - rect.left;
      const deltaY = previous.top - rect.top;
      if (Math.abs(deltaX) < 1 && Math.abs(deltaY) < 1) return;
      node.style.transition = 'none';
      node.style.transform = 'translate(' + Math.round(deltaX) + 'px,' + Math.round(deltaY) + 'px)';
      node.getBoundingClientRect();
      window.requestAnimationFrame(function(){
        node.style.transition = 'transform .24s cubic-bezier(.22, 1, .36, 1), box-shadow .18s ease, opacity .18s ease';
        node.style.transform = '';
        window.setTimeout(function(){
          if (!node.isConnected) return;
          node.style.transition = '';
        }, 260);
      });
    });
  }
  function findDesignerDropIntent(root, event){
    const targetNode = document.elementFromPoint(Number(event.clientX || 0), Number(event.clientY || 0));
    if (!targetNode) return null;
    const item = targetNode.closest ? targetNode.closest('[data-crud-designer-item]') : null;
    if (!item || !root.contains(item)) return null;
    const fieldName = String(item.getAttribute('data-crud-designer-name') || '');
    if (!fieldName || fieldName === state.designer.dragFieldName) return null;
    const targetIndex = Number(item.getAttribute('data-crud-designer-index') || '-1');
    const rect = item.getBoundingClientRect();
    const pointerX = Number(event.clientX || 0);
    const pointerY = Number(event.clientY || 0);
    const horizontal = rect.width > rect.height * 1.15;
    const before = horizontal ? (pointerX < (rect.left + rect.width / 2)) : (pointerY < (rect.top + rect.height / 2));
    let toIndex = before ? targetIndex : targetIndex + 1;
    const meta = currentDesignerMeta();
    const fromIndex = meta && meta.layout && Array.isArray(meta.layout.fields)
      ? meta.layout.fields.findIndex(function(entry){ return entry && entry.name === state.designer.dragFieldName; })
      : -1;
    if (fromIndex >= 0 && fromIndex < toIndex) {
      toIndex -= 1;
    }
    return {
      item: item,
      fieldName: fieldName,
      before: before,
      toIndex: Math.max(0, toIndex),
      hoverKey: fieldName + ':' + (before ? 'before' : 'after')
    };
  }
  function renderDesignerDropIntent(root, intent){
    root.querySelectorAll('.crud-designer-widget').forEach(function(node){
      node.classList.remove('is-drop-target');
      node.classList.remove('is-drop-before');
      node.classList.remove('is-drop-after');
      node.removeAttribute('data-crud-drop-side');
    });
    if (!intent || !intent.item) return;
    intent.item.classList.add('is-drop-target');
  }
  function createDesignerGhost(item){
    const ghost = item.cloneNode(true);
    ghost.removeAttribute('draggable');
    ghost.removeAttribute('data-crud-designer-item');
    ghost.removeAttribute('data-crud-designer-index');
    ghost.removeAttribute('data-crud-designer-name');
    ghost.classList.add('crud-designer-drag-ghost');
    ghost.classList.remove('is-drop-target');
    ghost.classList.remove('is-drop-before');
    ghost.classList.remove('is-drop-after');
    ghost.querySelectorAll('.crud-designer-slot-indicator').forEach(function(node){ node.remove(); });
    document.body.appendChild(ghost);
    return ghost;
  }
  function bindDesignerResize(root){
    if (!root) return;
    Array.prototype.slice.call(root.querySelectorAll('[data-crud-designer-resize]')).forEach(function(handle){
      handle.addEventListener('pointerdown', function(event){
        if (event.button !== 0) return;
        event.preventDefault();
        event.stopPropagation();
        const item = handle.closest('[data-crud-designer-item]');
        if (!item) return;
        clearDesignerDragState(root);
        clearDesignerResizeState(root);
        const fieldName = String(item.getAttribute('data-crud-designer-name') || '');
        if (!fieldName) return;
        const resizeMode = String(handle.getAttribute('data-crud-designer-resize') || 'both').trim() || 'both';
        const meta = currentDesignerMeta();
        const current = meta && meta.layout && Array.isArray(meta.layout.fields)
          ? meta.layout.fields.find(function(entry){ return entry && entry.name === fieldName; })
          : null;
        state.designer.resizeFieldName = fieldName;
        state.designer.resizePointerId = event.pointerId;
        state.designer.resizeSignature = '';
        state.designer.resizeMode = resizeMode;
        state.designer.resizeStartX = Number(event.clientX || 0);
        state.designer.resizeStartY = Number(event.clientY || 0);
        state.designer.resizeStartSpan = Math.max(1, Math.min(12, Number((current && current.col_span) || item.getAttribute('data-crud-current-span') || 6) || 6));
        state.designer.resizeStartHeight = Math.max(3, Math.min(18, Number((current && current.height) || 3) || 3));
        root.classList.add('is-resizing');
        item.classList.add('is-resizing');
        state.designer.resizeMoveHandler = function(moveEvent){
          if (state.designer.resizePointerId != null && moveEvent.pointerId !== state.designer.resizePointerId) return;
          const liveRoot = byId('crud-designer-canvas');
          const liveItem = liveRoot ? liveRoot.querySelector('[data-crud-designer-name="' + attrValueEscape(fieldName) + '"]') : null;
          if (!liveRoot || !liveItem) return;
          const dx = Number(moveEvent.clientX || 0) - Number(state.designer.resizeStartX || 0);
          const dy = Number(moveEvent.clientY || 0) - Number(state.designer.resizeStartY || 0);
          const mode = String(state.designer.resizeMode || 'both');
          const span = mode === 'height'
            ? Math.max(1, Math.min(12, Number(state.designer.resizeStartSpan || 6)))
            : Math.max(1, Math.min(12, Number(state.designer.resizeStartSpan || 6) + Math.round(dx / spanStepForItem(liveItem))));
          const height = mode === 'width'
            ? Math.max(3, Math.min(18, Number(state.designer.resizeStartHeight || 3)))
            : Math.max(3, Math.min(18, Number(state.designer.resizeStartHeight || 3) + Math.round(dy / resizeHeightStep())));
          const signature = String(span) + ':' + String(height);
          if (signature === state.designer.resizeSignature) return;
          state.designer.resizeSpan = span;
          state.designer.resizeSignature = signature;
          updateDesignerFieldDimensions(fieldName, span, height);
          window.requestAnimationFrame(function(){
            const nextRoot = byId('crud-designer-canvas');
            const nextItem = nextRoot ? nextRoot.querySelector('[data-crud-designer-name="' + attrValueEscape(fieldName) + '"]') : null;
            if (!nextRoot || !nextItem) return;
            nextRoot.classList.add('is-resizing');
            nextItem.classList.add('is-resizing');
            updateDesignerGuides(nextRoot, nextItem, span, height);
          });
        };
        state.designer.resizeUpHandler = function(upEvent){
          if (state.designer.resizePointerId != null && upEvent.pointerId !== state.designer.resizePointerId) return;
          clearDesignerResizeState(byId('crud-designer-canvas'));
        };
        document.addEventListener('pointermove', state.designer.resizeMoveHandler, true);
        document.addEventListener('pointerup', state.designer.resizeUpHandler, true);
        document.addEventListener('pointercancel', state.designer.resizeUpHandler, true);
      });
    });
  }
  function bindDesignerDrag(root){
    if (!root) return;
    const items = Array.prototype.slice.call(root.querySelectorAll('[data-crud-designer-item]'));
    items.forEach(function(item){
      item.ondragstart = function(){ return false; };
      item.addEventListener('pointerdown', function(event){
        if (event.button !== 0) return;
        if (state.designer.resizeFieldName) return;
        event.preventDefault();
        clearDesignerDragState(root);
        clearDesignerResizeState(root);
        const fieldName = String(item.getAttribute('data-crud-designer-name') || '');
        const index = Number(item.getAttribute('data-crud-designer-index') || '-1');
        if (!fieldName || index < 0) return;
        state.designer.dragFieldName = fieldName;
        state.designer.dragIndex = index;
        state.designer.dragPointerId = event.pointerId;
        state.designer.dragGhost = createDesignerGhost(item);
        updateDesignerGhostPosition(event);
        root.classList.add('is-sorting');
        item.classList.add('is-dragging');
        state.designer.dragMoveHandler = function(moveEvent){
          if (state.designer.dragPointerId != null && moveEvent.pointerId !== state.designer.dragPointerId) return;
          updateDesignerGhostPosition(moveEvent);
          const intent = findDesignerDropIntent(root, moveEvent);
          renderDesignerDropIntent(root, intent);
          if (!intent || !intent.hoverKey) return;
          if (intent.hoverKey === state.designer.dragHoverKey) return;
          state.designer.dragHoverKey = intent.hoverKey;
          reorderDesignerFieldByName(state.designer.dragFieldName, intent.toIndex);
          window.requestAnimationFrame(function(){
            const nextRoot = byId('crud-designer-canvas');
            if (!nextRoot) return;
            nextRoot.classList.add('is-sorting');
            const nextDragged = nextRoot.querySelector('[data-crud-designer-name="' + attrValueEscape(state.designer.dragFieldName) + '"]');
            if (nextDragged) nextDragged.classList.add('is-dragging');
            const refreshedIntent = findDesignerDropIntent(nextRoot, moveEvent);
            renderDesignerDropIntent(nextRoot, refreshedIntent);
          });
        };
        state.designer.dragUpHandler = function(upEvent){
          if (state.designer.dragPointerId != null && upEvent.pointerId !== state.designer.dragPointerId) return;
          clearDesignerDragState(byId('crud-designer-canvas'));
        };
        document.addEventListener('pointermove', state.designer.dragMoveHandler, true);
        document.addEventListener('pointerup', state.designer.dragUpHandler, true);
        document.addEventListener('pointercancel', state.designer.dragUpHandler, true);
      });
    });
  }
