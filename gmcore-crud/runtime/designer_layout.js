  const designerModes = ['create', 'edit', 'clone'];

  function cloneJSON(value){
    return JSON.parse(JSON.stringify(value == null ? null : value));
  }
  function attrValueEscape(value){
    return String(value == null ? '' : value).replace(/\\/g, '\\\\').replace(/"/g, '\\"');
  }
  function designerFieldFromField(field){
    return {
      name: field.name,
      hidden: !!field.hidden,
      col_span: Number(field.col_span || field.colSpan || 6) || 6,
      widget: String(field.widget || field.type || 'text'),
      height: Number(field.height || field.rows || 3) || 3,
      label: String(field.label || ''),
      placeholder: String(field.placeholder || ''),
      help: String(field.help || field.help_text || field.helpText || field.help_key || ''),
      required: !!field.required,
      read_only: !!(field.read_only || field.readOnly),
      disabled: !!(field.disabled || field.auto_managed),
      color: String(field.color || 'default'),
      background_color: String(field.background_color || field.backgroundColor || ''),
      border_color: String(field.border_color || field.borderColor || ''),
      validation: Array.isArray(field.validation) ? field.validation.slice() : []
    };
  }
  function normalizeDesignerLayout(fields, layout){
    const baseFields = Array.isArray(fields) ? fields : [];
    const baseMap = {};
    baseFields.forEach(function(field){ baseMap[field.name] = field; });
    const ordered = [];
    const seen = {};
    const items = (layout && Array.isArray(layout.fields)) ? layout.fields : [];
    items.forEach(function(item){
      if (!item || !item.name || !baseMap[item.name]) return;
      ordered.push({
        name: item.name,
        hidden: !!item.hidden,
        col_span: Number(item.col_span || 6) || 6,
        widget: String(item.widget || baseMap[item.name].widget || baseMap[item.name].type || 'text'),
        height: Number(item.height || baseMap[item.name].height || baseMap[item.name].rows || 3) || 3,
        label: String(item.label || baseMap[item.name].label || ''),
        placeholder: String(item.placeholder || baseMap[item.name].placeholder || ''),
        help: String(item.help || baseMap[item.name].help || baseMap[item.name].help_text || baseMap[item.name].helpText || baseMap[item.name].help_key || ''),
        required: item.required != null ? !!item.required : !!baseMap[item.name].required,
        read_only: item.read_only != null ? !!item.read_only : !!(baseMap[item.name].read_only || baseMap[item.name].readOnly),
        disabled: item.disabled != null ? !!item.disabled : !!(baseMap[item.name].disabled || baseMap[item.name].auto_managed),
        color: String(item.color || baseMap[item.name].color || 'default'),
        background_color: String(item.background_color || baseMap[item.name].background_color || baseMap[item.name].backgroundColor || ''),
        border_color: String(item.border_color || baseMap[item.name].border_color || baseMap[item.name].borderColor || ''),
        validation: Array.isArray(item.validation) ? item.validation.slice() : (Array.isArray(baseMap[item.name].validation) ? baseMap[item.name].validation.slice() : [])
      });
      seen[item.name] = true;
    });
    baseFields.forEach(function(field){
      if (seen[field.name]) return;
      ordered.push(designerFieldFromField(field));
    });
    return {fields: ordered};
  }
  function applyDesignerLayout(fields, layout){
    const order = normalizeDesignerLayout(fields, layout);
    const baseMap = {};
    (fields || []).forEach(function(field){
      baseMap[field.name] = cloneJSON(field);
    });
    return order.fields.map(function(item){
      const field = baseMap[item.name] || {name:item.name};
      field.hidden = !!item.hidden;
      field.col_span = Number(item.col_span || 6) || 6;
      field.widget = String(item.widget || field.widget || field.type || 'text');
      field.height = Number(item.height || field.height || field.rows || 3) || 3;
      field.label = String(item.label || field.label || '');
      field.placeholder = String(item.placeholder || field.placeholder || '');
      field.help = String(item.help || field.help || field.help_text || field.helpText || field.help_key || '');
      field.required = !!item.required;
      field.read_only = !!item.read_only;
      field.disabled = !!item.disabled;
      field.color = String(item.color || field.color || 'default');
      field.background_color = String(item.background_color || field.background_color || field.backgroundColor || '');
      field.border_color = String(item.border_color || field.border_color || field.borderColor || '');
      field.validation = Array.isArray(item.validation) ? item.validation.slice() : (Array.isArray(field.validation) ? field.validation.slice() : []);
      if ((field.widget === 'textarea' || field.widget === 'code') && field.height > 0) {
        field.rows = Math.max(3, field.height);
      }
      return field;
    });
  }
  function designerModeLabel(mode){
    if (mode === 'create') return label('mode_create', 'Create');
    if (mode === 'edit') return label('mode_edit', 'Edit');
    if (mode === 'clone') return label('mode_clone', 'Clone');
    return mode;
  }
  function designerCopyLabel(mode){
    return 'Copy from ' + designerModeLabel(mode);
  }
  function baseDesignerField(name){
    const meta = currentDesignerMeta();
    if (!meta || !Array.isArray(meta.originalFields)) return null;
    return meta.originalFields.find(function(field){ return field && field.name === name; }) || null;
  }
  function currentDesignerFieldMeta(name){
    const meta = currentDesignerMeta();
    if (!meta || !meta.layout || !Array.isArray(meta.layout.fields)) return null;
    return meta.layout.fields.find(function(field){ return field && field.name === name; }) || null;
  }
  function clearSelectedDesignerField(){
    state.designer.selectedField = '';
    renderDesignerToolbar();
    renderDesignerHiddenTray();
    const root = byId('crud-designer-canvas');
    if (!root) return;
    root.classList.remove('is-has-selection');
    root.querySelectorAll('[data-crud-designer-item]').forEach(function(node){
      node.classList.remove('is-selected');
      node.classList.remove('is-dimmed');
    });
  }
  function selectedDesignerFieldName(){
    return String(state.designer.selectedField || '').trim();
  }
  function setSelectedDesignerField(name){
    state.designer.selectedField = String(name || '').trim();
    renderDesignerToolbar();
    renderDesignerHiddenTray();
    const root = byId('crud-designer-canvas');
    if (!root) return;
    const hasSelection = !!state.designer.selectedField;
    root.classList.toggle('is-has-selection', hasSelection);
    root.querySelectorAll('[data-crud-designer-item]').forEach(function(node){
      const selected = String(node.getAttribute('data-crud-designer-name') || '') === state.designer.selectedField;
      node.classList.toggle('is-selected', selected);
      node.classList.toggle('is-dimmed', hasSelection && !selected);
    });
  }
  function normalizeDesignerSelection(visibleFields){
    const items = Array.isArray(visibleFields) ? visibleFields : [];
    const current = selectedDesignerFieldName();
    if (current && items.some(function(item){ return item && item.field && item.field.name === current; })) {
      return current;
    }
    if (current && currentDesignerFieldMeta(current) && baseDesignerField(current)) {
      return current;
    }
    return '';
  }
  function widgetLabelName(widget){
    const value = String(widget || '').trim();
    if (value === 'text') return 'Text';
    if (value === 'email') return 'Email';
    if (value === 'password') return 'Password';
    if (value === 'tel') return 'Phone';
    if (value === 'url') return 'URL';
    if (value === 'number') return 'Number';
    if (value === 'textarea') return 'Textarea';
    if (value === 'wysiwyg' || value === 'wysiwyg_min') return 'WYSIWYG Min';
    if (value === 'wysiwyg_full') return 'WYSIWYG Full';
    if (value === 'code') return 'Code';
    if (value === 'select') return 'Select';
    if (value === 'multiselect') return 'Multi select';
    if (value === 'relation') return 'Relation';
    if (value === 'checkbox') return 'Checkbox';
    if (value === 'toggle') return 'Toggle';
    if (value === 'range') return 'Range';
    if (value === 'date') return 'Date';
    if (value === 'datetime-local' || value === 'datetime') return 'Date time';
    if (value === 'time') return 'Time';
    if (value === 'color') return 'Color';
    return humanizeFieldName(value);
  }
  function widgetIconPath(widget){
    const value = String(widget || '').trim();
    if (value === 'relation' || value === 'multiselect') return '/assets/crud/icons/stack.svg';
    if (value === 'wysiwyg' || value === 'wysiwyg_min' || value === 'wysiwyg_full') return '/assets/crud/icons/form.svg';
    if (value === 'select') return '/assets/crud/icons/filter.svg';
    if (value === 'checkbox' || value === 'toggle' || value === 'range' || value === 'color') return '/assets/crud/icons/settings.svg';
    return '/assets/crud/icons/form.svg';
  }
  function hasFieldOptions(field){
    return !!(field && Array.isArray(field.options) && field.options.length);
  }
  function allowedDesignerValidators(field){
    if (!field) return [];
    const type = String(field.type || '').trim().toLowerCase();
    const out = [];
    function push(value, label){
      if (!value) return;
      if (out.some(function(item){ return item.value === value; })) return;
      out.push({value:value, label:label});
    }
    if (type === 'int' || type === 'integer' || type === 'float' || type === 'number') {
      push('pattern=^\\d+$', 'Digits only');
      push('min=0', 'Min 0');
    } else if (type === 'email') {
      push('email', 'Email');
    } else if (type === 'url') {
      push('pattern=^https?://.+$', 'URL');
    } else if (type !== 'bool' && type !== 'array' && type !== 'json') {
      push('pattern=^\\d+$', 'Digits only');
      push('email', 'Email');
      push('pattern=^https?://.+$', 'URL');
      push('pattern=^[a-z0-9]+(?:-[a-z0-9]+)*$', 'Slug');
      push('minLength=3', 'Min 3');
      push('maxLength=255', 'Max 255');
    }
    return out;
  }
  function allowedDesignerWidgets(field){
    if (!field) return [];
    const type = String(field.type || '').trim();
    const widgets = [];
    function push(widget){
      if (!widget) return;
      if (widgets.indexOf(widget) >= 0) return;
      widgets.push(widget);
    }
    if (String(field.relation || '').trim()) {
      push('relation');
      if (field.multiple) push('multiselect');
    }
    if (type === 'bool') {
      push('checkbox');
      push('toggle');
    } else if (type === 'array') {
      push('multiselect');
      if (String(field.relation || '').trim()) push('relation');
    } else if (type === 'json') {
      push('textarea');
      push('code');
    } else if (type === 'datetime') {
      push('datetime-local');
      push('text');
    } else if (type === 'date') {
      push('date');
      push('text');
    } else if (type === 'time') {
      push('time');
      push('text');
    } else if (type === 'int' || type === 'integer' || type === 'float' || type === 'number') {
      push('number');
      push('range');
      push('text');
    } else if (type === 'email') {
      push('email');
      push('text');
      push('textarea');
    } else {
      push('text');
      push('wysiwyg_min');
      push('wysiwyg_full');
      push('textarea');
      push('email');
      push('url');
      push('tel');
      push('password');
    }
    if (hasFieldOptions(field) && !String(field.relation || '').trim()) {
      push(field.multiple ? 'multiselect' : 'select');
    }
    return widgets;
  }
  function updateDesignerFieldWidget(fieldName, widget){
    const meta = currentDesignerMeta();
    const baseField = baseDesignerField(fieldName);
    if (!meta || !meta.layout || !Array.isArray(meta.layout.fields) || !baseField) return;
    const safeWidget = String(widget || '').trim();
    if (!safeWidget) return;
    if (allowedDesignerWidgets(baseField).indexOf(safeWidget) < 0) return;
    captureDesignerFlipState();
    let changed = false;
    meta.layout.fields = meta.layout.fields.map(function(item){
      if (!item || item.name !== fieldName) return item;
      if (String(item.widget || '') === safeWidget) return item;
      changed = true;
      const next = cloneJSON(item);
      next.widget = safeWidget;
      return next;
    });
    if (!changed) return;
    meta.layout = designerLayoutPayload(meta.layout);
    renderDesignerCanvas();
    renderDesignerToolbar();
  }
  function updateDesignerFieldHidden(fieldName, hidden){
    const meta = currentDesignerMeta();
    if (!meta || !meta.layout || !Array.isArray(meta.layout.fields)) return;
    let changed = false;
    meta.layout.fields = meta.layout.fields.map(function(item){
      if (!item || item.name !== fieldName) return item;
      if (!!item.hidden === !!hidden) return item;
      changed = true;
      const next = cloneJSON(item);
      next.hidden = !!hidden;
      return next;
    });
    if (!changed) return;
    meta.layout = designerLayoutPayload(meta.layout);
    renderDesigner();
  }
  function updateDesignerFieldColors(fieldName, backgroundColor, borderColor){
    const meta = currentDesignerMeta();
    if (!meta || !meta.layout || !Array.isArray(meta.layout.fields)) return;
    const nextBackground = String(backgroundColor || '').trim();
    const nextBorder = String(borderColor || '').trim();
    let changed = false;
    meta.layout.fields = meta.layout.fields.map(function(item){
      if (!item || item.name !== fieldName) return item;
      const currentBackground = String(item.background_color || '').trim();
      const currentBorder = String(item.border_color || '').trim();
      if (currentBackground === nextBackground && currentBorder === nextBorder) return item;
      changed = true;
      const next = cloneJSON(item);
      next.background_color = nextBackground;
      next.border_color = nextBorder;
      return next;
    });
    if (!changed) return;
    meta.layout = designerLayoutPayload(meta.layout);
    renderDesignerCanvas();
    renderDesignerToolbar();
  }
  function updateDesignerFieldProps(fieldName, patch){
    const meta = currentDesignerMeta();
    if (!meta || !meta.layout || !Array.isArray(meta.layout.fields) || !patch) return;
    let changed = false;
    meta.layout.fields = meta.layout.fields.map(function(item){
      if (!item || item.name !== fieldName) return item;
      const next = cloneJSON(item);
      Object.keys(patch).forEach(function(key){
        const incoming = patch[key];
        const current = next[key];
        if (String(current) === String(incoming) && typeof current === typeof incoming) return;
        next[key] = incoming;
        changed = true;
      });
      return next;
    });
    if (!changed) return;
    meta.layout = designerLayoutPayload(meta.layout);
    renderDesignerCanvas();
    renderDesignerToolbar();
  }
  function currentDesignerMeta(){
    return state.designer.metas[state.designer.mode] || null;
  }
  function currentDesignerLayout(){
    const meta = currentDesignerMeta();
    return meta && meta.layout ? meta.layout : {fields: []};
  }
  function designerLayoutPayload(layout){
    const safeLayout = layout && Array.isArray(layout.fields) ? layout.fields : [];
    return {
      fields: safeLayout.map(function(item){
        return {
          name: String(item.name || '').trim(),
          hidden: !!item.hidden,
          col_span: Number(item.col_span || 6) || 6,
          widget: String(item.widget || 'text'),
          height: Number(item.height || 3) || 3,
          label: String(item.label || ''),
          placeholder: String(item.placeholder || ''),
          help: String(item.help || ''),
          required: !!item.required,
          read_only: !!item.read_only,
          disabled: !!item.disabled,
          color: String(item.color || 'default'),
          background_color: String(item.background_color || ''),
          border_color: String(item.border_color || ''),
          validation: Array.isArray(item.validation) ? item.validation.slice() : []
        };
      }).filter(function(item){ return !!item.name; })
    };
  }
  async function fetchFormMeta(mode){
    return fetchJSON(apiURL('/api/layout/' + encodeURIComponent(mode)), {credentials:'same-origin'});
  }
  async function ensureDesignerMode(mode){
    mode = String(mode || 'create');
    if (state.designer.metas[mode]) return state.designer.metas[mode];
    const meta = await fetchFormMeta(mode);
    const sourceFields = ((meta.form && meta.form.fields) || formFields).map(function(field){ return cloneJSON(field); });
    state.designer.metas[mode] = {
      mode: mode,
      originalFields: sourceFields,
      layout: normalizeDesignerLayout(sourceFields, meta.layout || {fields: []}),
      saving: false
    };
    return state.designer.metas[mode];
  }
  async function copyDesignerMode(fromMode, toMode){
    if (!fromMode || !toMode || fromMode === toMode) return;
    const source = await ensureDesignerMode(fromMode);
    const target = await ensureDesignerMode(toMode);
    target.layout = normalizeDesignerLayout(target.originalFields, cloneJSON(source.layout));
    state.designer.mode = toMode;
    renderDesigner();
  }
  function reorderDesignerField(fromIndex, toIndex){
    const meta = currentDesignerMeta();
    if (!meta || !meta.layout || !Array.isArray(meta.layout.fields)) return;
    captureDesignerFlipState();
    const fields = meta.layout.fields.slice();
    if (fromIndex < 0 || fromIndex >= fields.length || toIndex < 0 || toIndex >= fields.length || fromIndex === toIndex) return;
    const moved = fields.splice(fromIndex, 1)[0];
    fields.splice(toIndex, 0, moved);
    meta.layout = designerLayoutPayload({fields: fields});
    renderDesignerCanvas();
  }
  function reorderDesignerFieldByName(fieldName, toIndex){
    const meta = currentDesignerMeta();
    if (!meta || !meta.layout || !Array.isArray(meta.layout.fields)) return;
    const fromIndex = meta.layout.fields.findIndex(function(item){ return item && item.name === fieldName; });
    if (fromIndex < 0) return;
    reorderDesignerField(fromIndex, toIndex);
  }
  function updateDesignerFieldSpan(fieldName, nextSpan){
    const meta = currentDesignerMeta();
    if (!meta || !meta.layout || !Array.isArray(meta.layout.fields)) return;
    const current = meta.layout.fields.find(function(item){ return item && item.name === fieldName; });
    const currentHeight = current ? Number(current.height || 3) || 3 : 3;
    updateDesignerFieldDimensions(fieldName, nextSpan, currentHeight);
  }
  function updateDesignerFieldDimensions(fieldName, nextSpan, nextHeight){
    const meta = currentDesignerMeta();
    if (!meta || !meta.layout || !Array.isArray(meta.layout.fields)) return;
    const safeSpan = Math.max(1, Math.min(12, Number(nextSpan || 6) || 6));
    const safeHeight = Math.max(3, Math.min(18, Number(nextHeight || 3) || 3));
    captureDesignerFlipState();
    let changed = false;
    meta.layout.fields = meta.layout.fields.map(function(item){
      if (!item || item.name !== fieldName) return item;
      const currentSpan = Math.max(1, Math.min(12, Number(item.col_span || 6) || 6));
      const currentHeight = Math.max(3, Math.min(18, Number(item.height || 3) || 3));
      if (currentSpan === safeSpan && currentHeight === safeHeight) return item;
      changed = true;
      const next = cloneJSON(item);
      next.col_span = safeSpan;
      next.height = safeHeight;
      return next;
    });
    if (!changed) return;
    meta.layout = designerLayoutPayload(meta.layout);
    renderDesignerCanvas();
  }
  function clearDesignerDragState(root){
    if (state.designer.dragMoveHandler) {
      document.removeEventListener('pointermove', state.designer.dragMoveHandler, true);
      state.designer.dragMoveHandler = null;
    }
    if (state.designer.dragUpHandler) {
      document.removeEventListener('pointerup', state.designer.dragUpHandler, true);
      document.removeEventListener('pointercancel', state.designer.dragUpHandler, true);
      state.designer.dragUpHandler = null;
    }
    if (state.designer.dragPointerId != null) {
      state.designer.dragPointerId = null;
    }
    if (state.designer.dragGhost && state.designer.dragGhost.parentNode) {
      state.designer.dragGhost.parentNode.removeChild(state.designer.dragGhost);
    }
    state.designer.dragGhost = null;
    state.designer.dragIndex = -1;
    state.designer.dragFieldName = '';
    state.designer.dragHoverKey = '';
    if (root) {
      root.classList.remove('is-sorting');
      root.querySelectorAll('.crud-designer-widget').forEach(function(node){
        node.classList.remove('is-dragging');
        node.classList.remove('is-drop-target');
        node.classList.remove('is-drop-before');
        node.classList.remove('is-drop-after');
        node.removeAttribute('data-crud-drop-side');
      });
    }
  }
