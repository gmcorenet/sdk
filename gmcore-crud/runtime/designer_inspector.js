  function renderDesignerModeHeader(){
    const bar = byId('crud-designer-controlbar');
    if (!bar) return;
    bar.innerHTML = '' +
      '<div class="crud-designer-controlbar-panel">' +
        '<div class="crud-designer-controlbar-section">' +
          '<div class="crud-designer-controlbar-kicker">' + esc(label('designer_layout_mode', 'Layout mode')) + '</div>' +
          '<select id="crud-designer-modes" class="crud-select crud-designer-mode-select">' +
            designerModes.map(function(mode){
              const selected = mode === state.designer.mode ? ' selected' : '';
              return '<option value="' + esc(mode) + '"' + selected + '>' + esc(designerModeLabel(mode)) + '</option>';
            }).join('') +
          '</select>' +
        '</div>' +
        '<div class="crud-designer-controlbar-section">' +
          '<div class="crud-designer-controlbar-kicker">' + esc(label('copy_from', 'Copy from…')) + '</div>' +
          '<div class="crud-designer-copy-tools">' +
            '<select id="crud-designer-copy-source" class="crud-select crud-select-sm">' +
              '<option value="">' + esc(label('copy_from', 'Copy from…')) + '</option>' +
              designerModes.filter(function(mode){
                return mode !== state.designer.mode;
              }).map(function(mode){
                return '<option value="' + esc(mode) + '">' + esc(designerCopyLabel(mode)) + '</option>';
              }).join('') +
            '</select>' +
            '<button class="crud-btn crud-btn-outline-secondary crud-btn-sm" id="crud-designer-copy-apply" type="button">Copy layout</button>' +
          '</div>' +
        '</div>' +
      '</div>';
    const modeSelect = byId('crud-designer-modes');
    if (modeSelect) {
      modeSelect.onchange = async function(){
        state.designer.mode = modeSelect.value || 'create';
        await ensureDesignerMode(state.designer.mode);
        renderDesigner();
      };
    }
    const copySelect = byId('crud-designer-copy-source');
    const copyButton = byId('crud-designer-copy-apply');
    if (copyButton) {
      copyButton.onclick = async function(){
        const sourceMode = copySelect ? String(copySelect.value || '') : '';
        if (!sourceMode) return;
        await copyDesignerMode(sourceMode, state.designer.mode);
      };
    }
  }
  function renderDesignerToolbar(){
    const panel = byId('crud-designer-toolbar');
    if (!panel) return;
    const fieldName = selectedDesignerFieldName();
    const baseField = baseDesignerField(fieldName);
    const layoutField = currentDesignerFieldMeta(fieldName);
    if (!fieldName || !baseField || !layoutField) {
      panel.classList.add('is-visible');
      panel.innerHTML = '' +
        '<div class="crud-designer-toolbar-panel crud-designer-toolbar-panel-empty">' +
          '<div class="crud-designer-toolbar-kicker">' + esc(label('designer_inspector', 'Field settings')) + '</div>' +
          '<div class="crud-designer-toolbar-title">' + esc(label('designer_select_field', 'Select a field')) + '</div>' +
          '<div class="crud-designer-toolbar-meta">' + esc(label('designer_select_field_copy', 'Click the cog on a widget to edit its widget type and visibility.')) + '</div>' +
        '</div>';
      return;
    }
    panel.classList.add('is-visible');
    const widgetOptions = allowedDesignerWidgets(baseField);
    const modeValue = designerModeLabel(state.designer.mode);
    const widgetValue = String(layoutField.widget || baseField.widget || baseField.type || 'text');
    const backgroundColor = String(layoutField.background_color || '');
    const borderColor = String(layoutField.border_color || '');
    panel.innerHTML = '' +
      '<div class="crud-designer-toolbar-panel">' +
        '<div class="crud-designer-toolbar-head">' +
          '<div>' +
            '<div class="crud-designer-toolbar-kicker">' + esc(label('designer_inspector', 'Field settings')) + '</div>' +
            '<div class="crud-designer-toolbar-title-row">' +
              '<span class="crud-designer-toolbar-title-icon"><img class="crud-icon" src="' + esc(widgetIconPath(widgetValue)) + '" alt=""></span>' +
              '<span class="crud-designer-toolbar-title">' + esc(displayFieldLabel(baseField) || baseField.name) + '</span>' +
            '</div>' +
            '<div class="crud-designer-toolbar-meta">' + esc(baseField.name) + ' · ' + esc(String(baseField.type || 'text')) + ' · ' + esc(modeValue) + '</div>' +
          '</div>' +
          '<div class="crud-designer-toolbar-stats">' +
            '<span class="crud-designer-toolbar-stat">' + esc(label('designer_width', 'Width')) + ': <strong>' + esc(String(layoutField.col_span || 6) + '/12') + '</strong></span>' +
            '<span class="crud-designer-toolbar-stat">' + esc(label('designer_height', 'Height')) + ': <strong>' + esc(String(layoutField.height || 3)) + '</strong></span>' +
            '<span class="crud-designer-toolbar-stat crud-designer-toolbar-stat-widget">' + esc(widgetLabelName(widgetValue)) + '</span>' +
          '</div>' +
        '</div>' +
        '<div class="crud-designer-toolbar-body">' +
          '<div class="crud-designer-toolbar-group">' +
            '<label class="crud-field-label" for="crud-designer-field-widget">' + esc(label('designer_widget', 'Widget')) + '</label>' +
            '<select class="crud-select" id="crud-designer-field-widget">' +
              widgetOptions.map(function(widget){
                const selected = String(layoutField.widget || '') === widget ? ' selected' : '';
                return '<option value="' + esc(widget) + '"' + selected + '>' + esc(widgetLabelName(widget)) + '</option>';
              }).join('') +
            '</select>' +
          '</div>' +
          '<div class="crud-designer-toolbar-group crud-designer-toolbar-group-color">' +
            '<label class="crud-field-label" for="crud-designer-field-background">' + esc(label('designer_background', 'Background')) + '</label>' +
            '<input class="crud-designer-color-input" type="color" id="crud-designer-field-background" value="' + esc(backgroundColor || '#ffffff') + '">' +
          '</div>' +
          '<div class="crud-designer-toolbar-group crud-designer-toolbar-group-color">' +
            '<label class="crud-field-label" for="crud-designer-field-border">' + esc(label('designer_border', 'Border')) + '</label>' +
            '<input class="crud-designer-color-input" type="color" id="crud-designer-field-border" value="' + esc(borderColor || '#d8dee8') + '">' +
          '</div>' +
          '<div class="crud-designer-toolbar-group crud-designer-toolbar-group-color-action">' +
            '<label class="crud-field-label">&nbsp;</label>' +
            '<button class="crud-btn crud-btn-outline-secondary crud-btn-sm" type="button" id="crud-designer-reset-colors">' + esc(label('designer_reset_colors', 'Reset colors')) + '</button>' +
          '</div>' +
        '</div>' +
      '</div>';
    const widgetSelect = byId('crud-designer-field-widget');
    if (widgetSelect) {
      widgetSelect.onchange = function(){
        updateDesignerFieldWidget(fieldName, widgetSelect.value || '');
      };
    }
    const backgroundInput = byId('crud-designer-field-background');
    const borderInput = byId('crud-designer-field-border');
    function commitColorUpdate(){
      updateDesignerFieldColors(
        fieldName,
        backgroundInput ? String(backgroundInput.value || '') : '',
        borderInput ? String(borderInput.value || '') : ''
      );
    }
    if (backgroundInput) {
      backgroundInput.oninput = commitColorUpdate;
      backgroundInput.onchange = commitColorUpdate;
    }
    if (borderInput) {
      borderInput.oninput = commitColorUpdate;
      borderInput.onchange = commitColorUpdate;
    }
    const resetColors = byId('crud-designer-reset-colors');
    if (resetColors) {
      resetColors.onclick = function(event){
        event.preventDefault();
        if (backgroundInput) backgroundInput.value = '#ffffff';
        if (borderInput) borderInput.value = '#d8dee8';
        updateDesignerFieldColors(fieldName, '', '');
      };
    }
  }
  function renderDesignerHiddenTray(){
    const tray = byId('crud-designer-hidden-tray');
    const meta = currentDesignerMeta();
    if (!tray || !meta || !meta.layout || !Array.isArray(meta.layout.fields)) return;
    const fieldName = selectedDesignerFieldName();
    const baseField = baseDesignerField(fieldName);
    const layoutField = currentDesignerFieldMeta(fieldName);
    const hiddenFields = meta.layout.fields.filter(function(field){ return field && field.hidden; }).map(function(item){
      return {
        layout: item,
        base: baseDesignerField(item.name)
      };
    }).filter(function(item){ return !!item.base; });
    const selectedField = selectedDesignerFieldName();
    const validatorOptions = allowedDesignerValidators(baseField);
    const selectedValidators = layoutField && Array.isArray(layoutField.validation) ? layoutField.validation.slice() : [];
    const stateBar = (!fieldName || !baseField || !layoutField)
      ? '<div class="crud-designer-field-state-placeholder">' + esc(label('designer_field_state_placeholder', 'Select a field to configure visibility and validation.')) + '</div>'
      : '<div class="crud-designer-field-state-row">' +
          '<div class="crud-designer-field-state-group">' +
            '<div class="crud-designer-hidden-kicker">' + esc(label('designer_states', 'States')) + '</div>' +
            '<div class="crud-designer-field-state-toggles">' +
              '<label class="crud-designer-toolbar-stat crud-designer-toolbar-stat-toggle' + (layoutField.hidden ? ' is-active' : '') + '">' +
                '<input class="crud-check-input" type="checkbox" id="crud-designer-bottom-hidden"' + (layoutField.hidden ? ' checked' : '') + '>' +
                '<span>' + esc(label('designer_hidden', 'Hidden')) + '</span>' +
              '</label>' +
              '<label class="crud-designer-toolbar-stat crud-designer-toolbar-stat-toggle' + (layoutField.required ? ' is-active' : '') + '">' +
                '<input class="crud-check-input" type="checkbox" id="crud-designer-bottom-required"' + (layoutField.required ? ' checked' : '') + '>' +
                '<span>' + esc(label('designer_required', 'Required')) + '</span>' +
              '</label>' +
              '<label class="crud-designer-toolbar-stat crud-designer-toolbar-stat-toggle' + (layoutField.read_only ? ' is-active' : '') + '">' +
                '<input class="crud-check-input" type="checkbox" id="crud-designer-bottom-readonly"' + (layoutField.read_only ? ' checked' : '') + '>' +
                '<span>' + esc(label('designer_read_only', 'Read only')) + '</span>' +
              '</label>' +
              '<label class="crud-designer-toolbar-stat crud-designer-toolbar-stat-toggle' + (layoutField.disabled ? ' is-active' : '') + '">' +
                '<input class="crud-check-input" type="checkbox" id="crud-designer-bottom-disabled"' + (layoutField.disabled ? ' checked' : '') + '>' +
                '<span>' + esc(label('designer_disabled', 'Disabled')) + '</span>' +
              '</label>' +
            '</div>' +
          '</div>' +
          '<div class="crud-designer-field-state-group crud-designer-field-state-validation">' +
            '<div class="crud-designer-hidden-kicker">' + esc(label('designer_validation', 'Validation')) + '</div>' +
            '<div class="crud-designer-validation-list">' +
              (validatorOptions.length ? validatorOptions.map(function(option){
                const checked = selectedValidators.indexOf(option.value) >= 0 ? ' checked' : '';
                return '<label class="crud-designer-toolbar-stat crud-designer-toolbar-stat-toggle' + (checked ? ' is-active' : '') + '">' +
                  '<input class="crud-check-input" type="checkbox" data-crud-designer-bottom-validation value="' + esc(option.value) + '"' + checked + '>' +
                  '<span>' + esc(option.label) + '</span>' +
                '</label>';
              }).join('') : '<span class="crud-designer-toolbar-meta">' + esc(label('designer_validation_none', 'No presets for this field')) + '</span>') +
            '</div>' +
          '</div>' +
        '</div>';
    tray.innerHTML = '' +
      '<div class="crud-designer-hidden-panel">' +
        stateBar +
        '<div class="crud-designer-hidden-kicker">' + esc(label('designer_hidden_fields', 'Hidden fields')) + '</div>' +
        '<div class="crud-designer-hidden-list-shell">' +
          (hiddenFields.length
            ? '<div class="crud-designer-hidden-list">' +
                hiddenFields.map(function(item){
                  return '' +
                    '<button type="button" class="crud-designer-hidden-chip' + (selectedField === item.base.name ? ' is-selected' : '') + '" data-crud-designer-hidden-select="' + esc(item.base.name) + '">' +
                      '<span class="crud-designer-hidden-chip-icon"><img class="crud-icon" src="/assets/crud/icons/settings.svg" alt=""></span>' +
                      '<span class="crud-designer-hidden-chip-label">' + esc(displayFieldLabel(item.base) || item.base.name) + '</span>' +
                      '<span class="crud-designer-hidden-chip-action">' + esc(label('designer_edit_hidden', 'Settings')) + '</span>' +
                    '</button>';
                }).join('') +
              '</div>'
            : '<div class="crud-designer-hidden-empty">' + esc(label('designer_hidden_empty', 'No hidden fields in this mode.')) + '</div>') +
        '</div>' +
      '</div>';
    tray.classList.add('is-visible');
    const hiddenToggle = byId('crud-designer-bottom-hidden');
    if (hiddenToggle) {
      hiddenToggle.onchange = function(){
        updateDesignerFieldHidden(fieldName, !!hiddenToggle.checked);
      };
    }
    const requiredToggle = byId('crud-designer-bottom-required');
    const readOnlyToggle = byId('crud-designer-bottom-readonly');
    const disabledToggle = byId('crud-designer-bottom-disabled');
    function commitBottomBooleanProps(){
      updateDesignerFieldProps(fieldName, {
        required: requiredToggle ? !!requiredToggle.checked : false,
        read_only: readOnlyToggle ? !!readOnlyToggle.checked : false,
        disabled: disabledToggle ? !!disabledToggle.checked : false
      });
    }
    [requiredToggle, readOnlyToggle, disabledToggle].forEach(function(toggle){
      if (!toggle) return;
      toggle.onchange = commitBottomBooleanProps;
    });
    Array.prototype.slice.call(tray.querySelectorAll('[data-crud-designer-bottom-validation]')).forEach(function(toggle){
      toggle.onchange = function(){
        const values = Array.prototype.slice.call(tray.querySelectorAll('[data-crud-designer-bottom-validation]:checked')).map(function(node){
          return String(node.value || '').trim();
        }).filter(Boolean);
        updateDesignerFieldProps(fieldName, {validation: values});
      };
    });
    Array.prototype.slice.call(tray.querySelectorAll('[data-crud-designer-hidden-select]')).forEach(function(button){
      button.onclick = function(event){
        event.preventDefault();
        const fieldName = String(button.getAttribute('data-crud-designer-hidden-select') || '').trim();
        if (!fieldName) return;
        setSelectedDesignerField(fieldName);
      };
    });
  }
