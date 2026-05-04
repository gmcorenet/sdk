  function renderDesignerCanvas(){
    const root = byId('crud-designer-canvas');
    const meta = currentDesignerMeta();
    if (!root || !meta) return;
    const fields = applyDesignerLayout(meta.originalFields, meta.layout);
    const visibleFields = [];
    fields.forEach(function(field, index){
      if (field.hidden) return;
      visibleFields.push({field: field, layoutIndex: index});
    });
    state.designer.selectedField = normalizeDesignerSelection(visibleFields);
    root.innerHTML = '<form class="crud-designer-form-surface" onsubmit="return false;"><div class="crud-form-grid">' +
      visibleFields.map(function(item){
        const field = item.field;
        const mode = state.designer.mode === 'clone' ? 'clone' : (state.designer.mode === 'edit' ? 'edit' : 'create');
        const selectedClass = state.designer.selectedField === field.name ? ' is-selected' : '';
        const styleParts = [];
        if (field.background_color) styleParts.push('--crud-designer-widget-bg:' + String(field.background_color) + ';');
        if (field.border_color) styleParts.push('--crud-designer-widget-border:' + String(field.border_color) + ';');
        return '<div class="crud-form-col ' + fieldSpanClass(field) + ' crud-designer-widget' + selectedClass + '" draggable="true" data-crud-designer-item data-crud-designer-index="' + item.layoutIndex + '" data-crud-designer-name="' + esc(field.name) + '" data-crud-current-span="' + esc(field.col_span || field.colSpan || 6) + '" data-crud-current-height="' + esc(field.height || field.rows || 3) + '"' + (styleParts.length ? ' style="' + esc(styleParts.join('')) + '"' : '') + '>' +
          '<button type="button" class="crud-designer-select-handle" data-crud-designer-select aria-label="' + esc(label('designer_select', 'Field settings')) + '"><img class="crud-icon" src="/assets/crud/icons/settings.svg" alt=""></button>' +
          '<button type="button" class="crud-designer-resize-handle crud-designer-resize-handle-side" data-crud-designer-resize="width" aria-label="' + esc(label('resize_width', 'Resize width')) + '"><span class="crud-designer-resize-grip crud-designer-resize-grip-side"></span></button>' +
          '<button type="button" class="crud-designer-resize-handle crud-designer-resize-handle-bottom" data-crud-designer-resize="height" aria-label="' + esc(label('resize_height', 'Resize height')) + '"><span class="crud-designer-resize-grip crud-designer-resize-grip-bottom"></span></button>' +
          '<button type="button" class="crud-designer-resize-handle crud-designer-resize-handle-corner" data-crud-designer-resize="both" aria-label="' + esc(label('resize_field', 'Resize field')) + '"><span class="crud-designer-resize-grip crud-designer-resize-grip-corner"></span></button>' +
          formField(field, {}, mode, true) +
        '</div>';
      }).join('') +
      '</div></form>';
    const hasVisibleSelection = !!state.designer.selectedField && visibleFields.some(function(item){
      return item && item.field && item.field.name === state.designer.selectedField;
    });
    root.classList.toggle('is-has-selection', hasVisibleSelection);
    Array.prototype.slice.call(root.querySelectorAll('[data-crud-designer-select]')).forEach(function(button){
      button.onpointerdown = function(event){
        event.preventDefault();
        event.stopPropagation();
      };
      button.onclick = function(event){
        event.preventDefault();
        event.stopPropagation();
        const item = button.closest('[data-crud-designer-item]');
        if (!item) return;
        setSelectedDesignerField(item.getAttribute('data-crud-designer-name') || '');
      };
    });
    root.querySelectorAll('[data-crud-designer-item]').forEach(function(node){
      const selected = String(node.getAttribute('data-crud-designer-name') || '') === state.designer.selectedField;
      node.classList.toggle('is-selected', selected);
      node.classList.toggle('is-dimmed', hasVisibleSelection && !selected);
      const shell = node.querySelector('.crud-field-shell');
      if (shell) {
        const bg = String(node.style.getPropertyValue('--crud-designer-widget-bg') || '').trim();
        const border = String(node.style.getPropertyValue('--crud-designer-widget-border') || '').trim();
        shell.style.background = bg || '';
        shell.style.borderColor = border || '';
      }
    });
    bindDesignerDrag(root);
    bindDesignerResize(root);
    playDesignerFlip(root);
    if (state.designer.dragFieldName) {
      root.classList.add('is-sorting');
      const dragged = root.querySelector('[data-crud-designer-name="' + attrValueEscape(state.designer.dragFieldName) + '"]');
      if (dragged) dragged.classList.add('is-dragging');
    }
    if (state.designer.resizeFieldName) {
      root.classList.add('is-resizing');
      const resized = root.querySelector('[data-crud-designer-name="' + attrValueEscape(state.designer.resizeFieldName) + '"]');
      if (resized) resized.classList.add('is-resizing');
    }
  }
  function renderDesignerSaveState(message, saving){
    const saveButton = byId('crud-designer-save');
    if (saveButton) {
      saveButton.disabled = !!saving;
      const text = saving ? label('saving', 'Saving…') : (message || label('save_layout', 'Save layout'));
      saveButton.textContent = text;
    }
  }
  function renderDesigner(){
    renderDesignerModeHeader();
    renderDesignerCanvas();
    renderDesignerToolbar();
    renderDesignerHiddenTray();
    renderDesignerSaveState(label('save_layout', 'Save layout'), false);
  }
  function bindDesignerDismiss(){
    const modal = byId('crud-designer-modal');
    if (!modal || modal.getAttribute('data-crud-designer-dismiss-bound') === '1') return;
    modal.setAttribute('data-crud-designer-dismiss-bound', '1');
    modal.addEventListener('pointerdown', function(event){
      if (state.designer.dragFieldName || state.designer.resizeFieldName) return;
      if (!event.target.closest('.crud-designer-body')) return;
      if (event.target.closest('[data-crud-designer-item]')) return;
      if (event.target.closest('#crud-designer-toolbar')) return;
      if (event.target.closest('#crud-designer-hidden-tray')) return;
      if (event.target.closest('#crud-designer-controlbar')) return;
      if (event.target.closest('.crud-designer-workspace')) return;
      clearSelectedDesignerField();
    });
  }
  async function openDesigner(initialMode){
    const mode = String(initialMode || state.designer.mode || 'create');
    state.designer.mode = designerModes.indexOf(mode) >= 0 ? mode : 'create';
    await ensureDesignerMode(state.designer.mode);
    bindDesignerDismiss();
    renderDesigner();
    const modalEl = byId('crud-designer-modal');
    const instance = window.gmcoreCrudModal && window.gmcoreCrudModal.getOrCreateInstance ? window.gmcoreCrudModal.getOrCreateInstance(modalEl) : null;
    if (instance) instance.show();
  }
  async function saveDesignerMode(){
    const meta = currentDesignerMeta();
    if (!meta || meta.saving) return;
    meta.saving = true;
    renderDesignerSaveState(label('saving', 'Saving…'), true);
    try {
      const payload = designerLayoutPayload(meta.layout);
      await fetchJSON(apiURL('/api/layout/' + encodeURIComponent(state.designer.mode)), {
        method: 'POST',
        credentials: 'same-origin',
        headers: headers,
        body: JSON.stringify(payload)
      });
      meta.layout = payload;
      renderDesignerSaveState(label('saved', 'Saved'), false);
      window.setTimeout(function(){
        if (currentDesignerMeta() === meta) {
          renderDesignerSaveState(label('save_layout', 'Save layout'), false);
        }
      }, 900);
    } catch (_err) {
      renderDesignerSaveState(label('save_layout', 'Save layout'), false);
    } finally {
      meta.saving = false;
    }
  }
