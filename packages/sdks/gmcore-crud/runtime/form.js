
  function normalizeMode(mode){
    if (mode === 'show' || mode === 'view') return 'edit';
    if (mode === 'clone') return 'clone';
    return mode === 'create' ? 'create' : 'edit';
  }
  function fieldHeightStyle(field){
    const units = Number(field.height || field.rows || 3) || 3;
    let style = '--crud-field-height:' + (44 + ((units - 1) * 10)) + 'px;';
    if (field && String(field.background_color || field.backgroundColor || '').trim()) {
      style += '--crud-field-bg:' + String(field.background_color || field.backgroundColor || '').trim() + ';';
    }
    if (field && String(field.border_color || field.borderColor || '').trim()) {
      style += '--crud-field-border:' + String(field.border_color || field.borderColor || '').trim() + ';';
    }
    return style;
  }
  function fieldLabelText(field){
    if (field && String(field.label || '').trim()) return String(field.label).trim();
    return displayFieldLabel(field) || String(field && field.name || '');
  }
  function fieldHelpText(field){
    if (!field) return '';
    return String(field.help || field.help_text || field.helpText || field.help_key || '').trim();
  }
  function helpMarkup(field){
    const text = fieldHelpText(field);
    if (!text) return '';
    return '<div class="crud-field-caption">' + esc(text) + '</div>';
  }
  function requiredMark(field){
    if (!field || !field.required) return '';
    return '<span class="crud-field-required" aria-hidden="true">*</span>';
  }
  function fieldToneClass(field){
    return ' crud-field-tone-' + esc(String(field.color || 'default'));
  }
  let currentModalFields = [];
  function fieldValidationRules(field){
    if (!field) return [];
    const items = Array.isArray(field.validation) ? field.validation : [];
    return items.map(function(rule){ return String(rule || '').trim(); }).filter(Boolean);
  }
  function clearFieldErrors(form){
    if (!form || !form.querySelectorAll) return;
    form.querySelectorAll('.crud-field-shell').forEach(function(shell){
      shell.classList.remove('is-invalid');
    });
    form.querySelectorAll('.crud-field-error').forEach(function(node){
      if (node && node.parentNode) node.parentNode.removeChild(node);
    });
  }
  function renderFieldErrors(form, errors){
    if (!form || !errors || typeof errors !== 'object') return;
    Object.keys(errors).forEach(function(fieldName){
      const messages = Array.isArray(errors[fieldName]) ? errors[fieldName] : [errors[fieldName]];
      const input = form.querySelector('[name="' + CSS.escape(fieldName) + '"]') || form.querySelector('[name="' + CSS.escape(fieldName) + '[]"]');
      const shell = input ? input.closest('.crud-field-shell') : null;
      if (!shell) return;
      shell.classList.add('is-invalid');
      const error = document.createElement('div');
      error.className = 'crud-field-error';
      error.textContent = messages.filter(Boolean).join(' ');
      shell.appendChild(error);
    });
  }
  function clientValidationMessage(field, value, rule){
    const current = String(value == null ? '' : value).trim();
    const raw = String(rule || '').trim();
    if (!raw) return '';
    const parts = raw.split('=');
    const key = String(parts.shift() || '').trim().toLowerCase();
    const arg = parts.join('=').trim();
    if (key === 'required') {
      if (!current) return fieldLabelText(field) + ' is required';
      return '';
    }
    if (!current) return '';
    if (key === 'email') {
      return /^[^@\s]+@[^@\s]+\.[^@\s]+$/.test(current) ? '' : fieldLabelText(field) + ' must be a valid email';
    }
    if (key === 'minlength' || key === 'min_length') {
      const min = Number(arg || 0) || 0;
      return current.length >= min ? '' : fieldLabelText(field) + ' must have at least ' + min + ' characters';
    }
    if (key === 'maxlength' || key === 'max_length') {
      const max = Number(arg || 0) || 0;
      return current.length <= max ? '' : fieldLabelText(field) + ' must have at most ' + max + ' characters';
    }
    if (key === 'pattern' || key === 'regex') {
      try {
        return new RegExp(arg).test(current) ? '' : fieldLabelText(field) + ' has an invalid format';
      } catch (_err) {
        return '';
      }
    }
    return '';
  }
  function validateFormFields(form, fields){
    const errors = {};
    (fields || []).forEach(function(field){
      if (!field || field.hidden) return;
      const rules = fieldValidationRules(field);
      if (field.required && rules.indexOf('required') < 0) rules.unshift('required');
      if (!rules.length) return;
      const input = form.querySelector('[name="' + CSS.escape(String(field.name || '')) + '"]') || form.querySelector('[name="' + CSS.escape(String(field.name || '')) + '[]"]');
      let value = '';
      if (!input) {
        value = '';
      } else if (input.type === 'checkbox') {
        value = input.checked ? 'true' : '';
      } else {
        value = input.value;
      }
      rules.forEach(function(rule){
        const message = clientValidationMessage(field, value, rule);
        if (!message) return;
        if (!errors[field.name]) errors[field.name] = [];
        errors[field.name].push(message);
      });
    });
    return errors;
  }
  function bindWysiwygWidgets(root){
    Array.prototype.slice.call((root || document).querySelectorAll('[data-crud-wysiwyg]')).forEach(function(shell){
      if (shell.getAttribute('data-crud-wysiwyg-bound') === '1') return;
      shell.setAttribute('data-crud-wysiwyg-bound', '1');
      const editor = shell.querySelector('[data-crud-wysiwyg-editor]');
      const hidden = shell.querySelector('[data-crud-wysiwyg-hidden]');
      const toolbar = shell.querySelector('[data-crud-wysiwyg-toolbar]');
      if (!editor || !hidden || !toolbar) return;
      let savedRange = null;
      function sync(){
        hidden.value = editor.innerHTML;
      }
      function selection(){
        return window.getSelection ? window.getSelection() : null;
      }
      function selectionInEditor(){
        const sel = selection();
        if (!sel || !sel.rangeCount) return null;
        const range = sel.getRangeAt(0);
        if (!editor.contains(range.commonAncestorContainer)) return null;
        return range;
      }
      function saveRange(){
        const range = selectionInEditor();
        if (range) savedRange = range.cloneRange();
      }
      function restoreRange(){
        if (!savedRange) return null;
        const sel = selection();
        if (!sel) return null;
        try {
          sel.removeAllRanges();
          sel.addRange(savedRange);
          return savedRange;
        } catch (_err) {
          return null;
        }
      }
      function editable(){
        return editor.getAttribute('contenteditable') !== 'false';
      }
      function selectedFragmentText(range){
        const fragment = range.cloneContents();
        const wrap = document.createElement('div');
        wrap.appendChild(fragment);
        return String(wrap.textContent || '');
      }
      function nearestElement(node){
        if (!node) return null;
        return node.nodeType === 1 ? node : node.parentElement;
      }
      function nearestBlock(node){
        let current = nearestElement(node);
        while (current && current !== editor) {
          const tag = String(current.tagName || '').toLowerCase();
          if (['p', 'div', 'h1', 'h2', 'h3', 'h4', 'blockquote', 'li'].indexOf(tag) >= 0) return current;
          current = current.parentElement;
        }
        return editor;
      }
      function findAncestorTag(node, tagName){
        let current = nearestElement(node);
        const target = String(tagName || '').toLowerCase();
        while (current && current !== editor) {
          if (String(current.tagName || '').toLowerCase() === target) return current;
          current = current.parentElement;
        }
        return null;
      }
      function unwrapNode(node){
        if (!node || !node.parentNode) return;
        while (node.firstChild) node.parentNode.insertBefore(node.firstChild, node);
        node.parentNode.removeChild(node);
      }
      function wrapSelection(tagName, attrs){
        const range = restoreRange() || selectionInEditor();
        if (!range || range.collapsed) return;
        const wrapper = document.createElement(tagName);
        Object.keys(attrs || {}).forEach(function(key){
          if (attrs[key] == null || attrs[key] === '') return;
          wrapper.setAttribute(key, attrs[key]);
        });
        try {
          const content = range.extractContents();
          wrapper.appendChild(content);
          range.insertNode(wrapper);
          range.selectNodeContents(wrapper);
          const sel = selection();
          if (sel) {
            sel.removeAllRanges();
            sel.addRange(range);
          }
          savedRange = range.cloneRange();
        } catch (_err) {}
      }
      function toggleInline(tagName){
        const range = restoreRange() || selectionInEditor();
        if (!range || range.collapsed) return;
        const existing = findAncestorTag(range.commonAncestorContainer, tagName);
        if (existing) {
          unwrapNode(existing);
          sync();
          return;
        }
        wrapSelection(tagName);
        sync();
      }
      function setBlockTag(tagName){
        const range = restoreRange() || selectionInEditor();
        if (!range) return;
        const block = nearestBlock(range.startContainer);
        if (!block || block === editor) return;
        const next = document.createElement(tagName);
        Array.from(block.attributes || []).forEach(function(attr){
          next.setAttribute(attr.name, attr.value);
        });
        next.innerHTML = block.innerHTML;
        block.parentNode.replaceChild(next, block);
        sync();
      }
      function setTextAlign(value){
        const range = restoreRange() || selectionInEditor();
        if (!range) return;
        const block = nearestBlock(range.startContainer);
        if (!block) return;
        block.style.textAlign = String(value || '').trim();
        sync();
      }
      function setTextColor(value){
        const color = String(value || '').trim();
        if (!color) return;
        wrapSelection('span', {style: 'color:' + color + ';'});
        sync();
      }
      function setFontSize(value){
        const map = {'1':'12px','2':'13px','3':'16px','4':'18px','5':'22px','6':'28px'};
        const size = map[String(value || '').trim()] || '';
        if (!size) return;
        wrapSelection('span', {style: 'font-size:' + size + ';'});
        sync();
      }
      function toggleList(tagName){
        const range = restoreRange() || selectionInEditor();
        if (!range) return;
        const block = nearestBlock(range.startContainer);
        if (!block || block === editor) return;
        const existingList = findAncestorTag(block, 'ul') || findAncestorTag(block, 'ol');
        if (existingList) {
          const fragment = document.createDocumentFragment();
          Array.from(existingList.querySelectorAll('li')).forEach(function(li){
            const p = document.createElement('p');
            p.innerHTML = li.innerHTML;
            fragment.appendChild(p);
          });
          existingList.parentNode.replaceChild(fragment, existingList);
          sync();
          return;
        }
        const list = document.createElement(tagName);
        const li = document.createElement('li');
        li.innerHTML = block.innerHTML;
        list.appendChild(li);
        block.parentNode.replaceChild(list, block);
        sync();
      }
      function insertLink(url){
        const clean = String(url || '').trim();
        if (!clean) return;
        wrapSelection('a', {href: clean, target: '_blank', rel: 'noopener noreferrer'});
        sync();
      }
      function clearFormatting(){
        const range = restoreRange() || selectionInEditor();
        if (!range || range.collapsed) return;
        const fragment = range.extractContents();
        const wrap = document.createElement('div');
        wrap.appendChild(fragment);
        wrap.querySelectorAll('strong,b,em,i,u,s,strike,span,font,blockquote,h1,h2,h3,h4,a').forEach(function(node){
          if (String(node.tagName || '').toLowerCase() === 'a') {
            unwrapNode(node);
            return;
          }
          node.removeAttribute('style');
          if (['span', 'font'].indexOf(String(node.tagName || '').toLowerCase()) >= 0) {
            unwrapNode(node);
          }
        });
        const cleanFragment = document.createDocumentFragment();
        while (wrap.firstChild) cleanFragment.appendChild(wrap.firstChild);
        range.insertNode(cleanFragment);
        sync();
      }
      function runAction(action, value){
        if (!editable()) return;
        editor.focus();
        restoreRange();
        if (action === 'bold') return toggleInline('strong');
        if (action === 'italic') return toggleInline('em');
        if (action === 'underline') return toggleInline('u');
        if (action === 'strike') return toggleInline('s');
        if (action === 'paragraph') return setBlockTag('p');
        if (action === 'h2') return setBlockTag('h2');
        if (action === 'h3') return setBlockTag('h3');
        if (action === 'quote') return setBlockTag('blockquote');
        if (action === 'ul') return toggleList('ul');
        if (action === 'ol') return toggleList('ol');
        if (action === 'link') {
          const selected = selectedFragmentText(selectionInEditor() || restoreRange());
          const url = window.prompt('Link URL', selected && /^https?:\/\//i.test(selected) ? selected : 'https://');
          if (url) insertLink(url);
          return;
        }
        if (action === 'align') return setTextAlign(value);
        if (action === 'font_size') return setFontSize(value);
        if (action === 'text_color') return setTextColor(value);
        if (action === 'clear') return clearFormatting();
      }
      ['mouseup', 'keyup', 'focus', 'blur'].forEach(function(name){
        editor.addEventListener(name, saveRange);
      });
      editor.addEventListener('input', function(){
        saveRange();
        sync();
      });
      toolbar.querySelectorAll('[data-crud-wysiwyg-action]').forEach(function(button){
        button.addEventListener('mousedown', function(event){ event.preventDefault(); });
        button.addEventListener('click', function(event){
          event.preventDefault();
          runAction(String(button.getAttribute('data-crud-wysiwyg-action') || '').trim(), String(button.getAttribute('data-crud-wysiwyg-value') || '').trim());
        });
      });
      toolbar.querySelectorAll('[data-crud-wysiwyg-select]').forEach(function(select){
        select.addEventListener('change', function(){
          runAction(String(select.getAttribute('data-crud-wysiwyg-select') || '').trim(), String(select.value || ''));
        });
      });
      toolbar.querySelectorAll('[data-crud-wysiwyg-color]').forEach(function(input){
        input.addEventListener('input', function(){
          runAction('text_color', String(input.value || ''));
        });
      });
      sync();
    });
  }
  function relationDisplayValue(field, record, value){
    if (!field || !record) return String(value == null ? '' : value);
    const displayField = String(field.display_field || '').trim();
    if (displayField && record[displayField] != null && String(record[displayField]).trim()) {
      return String(record[displayField]).trim();
    }
    const option = (field.options || []).find(function(item){
      return String(item.value) === String(value == null ? '' : value);
    });
    if (option && String(option.label || '').trim()) {
      return String(option.label).trim();
    }
    return String(value == null ? '' : value);
  }
  async function fetchRelationPage(field, query, page, limitOverride){
    const relation = String(field.relation || '').trim();
    const limit = Math.max(1, Number(limitOverride || field.async_limit || 0) || (field.async_options ? 25 : 100));
    const safePage = Math.max(1, Number(page || 1) || 1);
    if (!relation) return {options:[], page:1, limit:limit, total:0, has_more:false};
    const url = new URL(apiURL('/api/create'), window.location.origin);
    url.searchParams.set('crud_relation', relation);
    const q = String(query || '').trim();
    if (q) url.searchParams.set('q', q);
    url.searchParams.set('page', String(safePage));
    url.searchParams.set('limit', String(limit));
    const response = await fetchJSON(url.toString(), {credentials:'same-origin'});
    return {
      options: Array.isArray(response.options) ? response.options : [],
      page: Number(response.page || safePage) || safePage,
      limit: Number(response.limit || limit) || limit,
      total: Number(response.total || 0) || 0,
      has_more: !!response.has_more
    };
  }
  function closeRelationPortal(){
    closeFloatingPortal('crud-floating-relation');
  }
  function closeChoicePortal(){
    closeFloatingPortal('crud-floating-choice');
  }
  function choiceSuppressionUntil(){
    window.__gmcoreCrudChoiceCloseSuppress = window.__gmcoreCrudChoiceCloseSuppress || {};
    return Number(window.__gmcoreCrudChoiceCloseSuppress[state.instanceId] || 0);
  }
  function suppressChoiceAutoClose(){
    window.__gmcoreCrudChoiceCloseSuppress = window.__gmcoreCrudChoiceCloseSuppress || {};
    window.__gmcoreCrudChoiceCloseSuppress[state.instanceId] = Date.now() + 250;
  }
  function relationSuppressionUntil(){
    window.__gmcoreCrudRelationCloseSuppress = window.__gmcoreCrudRelationCloseSuppress || {};
    return Number(window.__gmcoreCrudRelationCloseSuppress[state.instanceId] || 0);
  }
  function suppressRelationAutoClose(){
    window.__gmcoreCrudRelationCloseSuppress = window.__gmcoreCrudRelationCloseSuppress || {};
    window.__gmcoreCrudRelationCloseSuppress[state.instanceId] = Date.now() + 250;
  }
  function setActiveRelationController(controller){
    window.__gmcoreCrudRelationControllers = window.__gmcoreCrudRelationControllers || {};
    window.__gmcoreCrudRelationControllers[state.instanceId] = controller || null;
  }
  function activeRelationController(){
    window.__gmcoreCrudRelationControllers = window.__gmcoreCrudRelationControllers || {};
    return window.__gmcoreCrudRelationControllers[state.instanceId] || null;
  }
  function renderRelationLoading(shell){
    if (!shell) return;
    openCrudPortal({
      name: 'crud-floating-relation',
      anchor: shell,
      html: '<div class="crud-relation-meta is-loading"><span class="crud-relation-counter">' + esc(label('loading', 'Loading…')) + '</span></div>',
      anchorAttr: 'data-crud-relation-anchor',
      minWidth: Math.max(260, shell.getBoundingClientRect().width || 0),
      gap: 6
    });
  }
  function relationState(shell){
    let state = {};
    try {
      state = JSON.parse(shell.getAttribute('data-crud-relation-field') || '{}') || {};
    } catch (_err) {
      state = {};
    }
    if (!state.__queryCache || typeof state.__queryCache !== 'object') state.__queryCache = {};
    if (!Array.isArray(state.__loadedOptions)) state.__loadedOptions = [];
    if (!Array.isArray(state.__selectedValues)) state.__selectedValues = [];
    if (!Array.isArray(state.__selectedItems)) state.__selectedItems = [];
    return state;
  }
  function writeRelationState(shell, state){
    shell.setAttribute('data-crud-relation-field', JSON.stringify(state || {}));
  }
  function bindRelationWidgets(root){
    Array.prototype.slice.call((root || document).querySelectorAll('[data-crud-relation-shell]')).forEach(function(shell){
      if (shell.getAttribute('data-crud-relation-bound') === '1') return;
      shell.setAttribute('data-crud-relation-bound', '1');
      const input = shell.querySelector('[data-crud-relation-input]');
      const hidden = shell.querySelector('[data-crud-relation-hidden]');
      const hiddenList = shell.querySelector('[data-crud-relation-hidden-list]');
      const selectedBox = shell.querySelector('[data-crud-relation-selected]');
      const list = shell.querySelector('[data-crud-relation-options]');
      if (!input || !list) return;
      if (!shell.getAttribute('data-crud-relation-anchor')) {
        shell.setAttribute('data-crud-relation-anchor', 'crud-relation-' + Math.random().toString(36).slice(2));
      }
      let timer = null;
      let keepOpen = false;
      const isMultiple = shell.getAttribute('data-crud-relation-multiple') === '1';
      const picker = createCrudChoicePicker({
        shell: shell,
        input: input,
        list: list,
        portalName: 'crud-floating-relation',
        anchorAttr: 'data-crud-relation-anchor',
        valueAttr: 'data-value',
        labelAttr: 'data-label',
        valueSelector: '[data-value]'
      });
      const scrollHost = shell.closest('.crud-modal-body');
      if (scrollHost && !scrollHost.getAttribute('data-crud-relation-scroll-bound')) {
        scrollHost.setAttribute('data-crud-relation-scroll-bound', '1');
        scrollHost.addEventListener('scroll', function(){
          closeRelationPortal();
        });
      }
      function state(){
        const next = relationState(shell);
        const initialValues = isMultiple ? asArray(shell.getAttribute('data-crud-relation-initial') || '') : [];
        const initialLabels = isMultiple ? asArray(shell.getAttribute('data-crud-relation-initial-labels') || '') : [];
        if (isMultiple && !next.__initialHydrated && !next.__selectedValues.length && initialValues.length) {
          next.__selectedValues = initialValues.slice();
          next.__selectedItems = initialValues.map(function(value, index){
            return {value: String(value), label: String(initialLabels[index] || value)};
          });
          next.__initialHydrated = true;
        }
        if (!isMultiple && hidden && String(hidden.value || '').trim() && !next.__selectedItems.length) {
          const currentValue = String(hidden.value || '').trim();
          next.__selectedValues = [currentValue];
          next.__selectedItems = [{value: currentValue, label: String(input.value || currentValue)}];
          next.__initialHydrated = true;
        }
        return next;
      }
      function save(next){
        writeRelationState(shell, next);
      }
      function currentQuery(nextState){
        return String((nextState || state()).__query || '').trim();
      }
      function portal(){
        return ensureFloatingPortal('crud-floating-relation', themeForNode(shell));
      }
      function isOpen(){
        return isFloatingPortalOpen('crud-floating-relation', shell.getAttribute('data-crud-relation-anchor') || '');
      }
      function close(){
        if (isOpen()) closeRelationPortal();
        list.classList.remove('is-open');
        if (activeRelationController() === controller) {
          setActiveRelationController(null);
        }
        picker.closeAccessibility();
        picker.resetActive();
      }
      function ensureHiddenInputs(nextState){
        if (isMultiple) {
          if (!hiddenList) return;
          hiddenList.innerHTML = (nextState.__selectedValues || []).map(function(value){
            return '<input type="hidden" name="' + esc(input.getAttribute('data-crud-name') || '') + '" value="' + esc(value) + '">';
          }).join('');
          return;
        }
        if (hidden) {
          hidden.value = nextState.__selectedValues[0] || '';
        }
      }
      function renderSelected(nextState){
        if (isMultiple && selectedBox) {
          selectedBox.innerHTML = (nextState.__selectedItems || []).map(function(item){
            return picker.chipMarkup(item, 'data-remove-value');
          }).join('');
          Array.prototype.slice.call(selectedBox.querySelectorAll('[data-remove-value]')).forEach(function(button){
            button.onclick = function(){
              suppressRelationAutoClose();
              keepOpen = true;
              const current = state();
              const value = button.getAttribute('data-remove-value') || '';
              current.__selectedValues = (current.__selectedValues || []).filter(function(item){ return String(item) !== String(value); });
              current.__selectedItems = (current.__selectedItems || []).filter(function(item){ return String(item.value) !== String(value); });
              current.__initialHydrated = true;
              save(current);
              ensureHiddenInputs(current);
              renderSelected(current);
              rerenderCurrentOptions();
              if (input && input.focus) input.focus();
              window.setTimeout(function(){ keepOpen = false; }, 0);
            };
          });
          return;
        }
        if (!isMultiple) {
          const first = (nextState.__selectedItems || [])[0];
          input.value = first ? String(first.label || first.value || '') : '';
        }
      }
      function queryCacheKey(query){
        return String(query || '').trim().toLowerCase();
      }
      function cachedResult(nextState, query){
        return nextState.__queryCache[queryCacheKey(query)] || null;
      }
      function writeCache(nextState, query, result){
        nextState.__queryCache[queryCacheKey(query)] = {
          options: result.options || [],
          page: Number(result.page || 1) || 1,
          limit: Number(result.limit || 25) || 25,
          total: Number(result.total || 0) || 0,
          has_more: !!result.has_more
        };
      }
      function selectedValuesForState(nextState){
        return isMultiple ? (nextState.__selectedValues || []) : [hidden ? String(hidden.value || '') : ''];
      }
      function visibleOptions(nextState, options){
        return picker.visibleOptions(options, isMultiple ? selectedValuesForState(nextState) : [], '', !isMultiple);
      }
      function optionMarkup(option, selected){
        return picker.optionMarkup(option, {selected:selected, valueAttr:'data-value', labelAttr:'data-label'});
      }
      function rerenderCurrentOptions(restoreScrollTop){
        const current = state();
        const cached = cachedResult(current, currentQuery(current));
        if (cached) {
          renderOptions(cached, restoreScrollTop);
        }
      }
      function renderOptions(result, restoreScrollTop){
        const nextState = state();
        nextState.__loadedOptions = result.options || [];
        nextState.__hasMore = !!result.has_more;
        nextState.__page = Number(result.page || 1) || 1;
        nextState.__limit = Number(result.limit || nextState.async_limit || 25) || 25;
        nextState.__total = Number(result.total || 0) || 0;
        writeCache(nextState, currentQuery(nextState), result);
        save(nextState);
        const filteredOptions = visibleOptions(nextState, result.options || []);
        const loadedCount = filteredOptions.length;
        const totalCount = Number(result.total || loadedCount) || loadedCount;
        const loadAllLimit = Math.max(0, Number(nextState.load_all_limit || 0) || 250);
        const showLoadAll = loadAllLimit > 0 && totalCount > (result.options || []).length && totalCount <= loadAllLimit;
        const selectedValues = selectedValuesForState(nextState);
        const html = picker.metaMarkup(String(loadedCount) + ' / ' + String(totalCount) + ' ' + label('loaded', 'loaded')) +
          filteredOptions.map(function(option){
            return optionMarkup(option, selectedValues.indexOf(String(option.value || '')) >= 0);
          }).join('') +
          (showLoadAll ? '<button type="button" class="crud-relation-option crud-relation-option-more" data-load-all="1">' + esc(label('load_all', 'Load all matching')) + '</button>' : '') +
          (result.has_more ? '<button type="button" class="crud-relation-option crud-relation-option-more" data-load-more="1">' + esc(label('load_more', 'Load more')) + '</button>' : '');
        list.innerHTML = html;
        const dropdown = picker.openPortal(html, restoreScrollTop);
        if (dropdown) {
          dropdown.onmousedown = function(){ keepOpen = true; };
          dropdown.onmouseup = function(){ keepOpen = false; };
        }
        setActiveRelationController(controller);
        const target = dropdown || list;
        if (!(typeof restoreScrollTop === 'number' && restoreScrollTop > 0)) {
          picker.setActiveIndex(target, 0);
        }
        picker.bindOptions(target, '[data-value]', function(button){
            const value = String(button.getAttribute('data-value') || '');
            const labelText = String(button.getAttribute('data-label') || value);
            const current = state();
            if (isMultiple) {
              suppressRelationAutoClose();
              keepOpen = true;
              if ((current.__selectedValues || []).indexOf(value) < 0) {
                current.__selectedValues.push(value);
                current.__selectedItems.push({value:value, label:labelText});
              }
              current.__initialHydrated = true;
              save(current);
              ensureHiddenInputs(current);
              renderSelected(current);
              input.value = '';
              rerenderCurrentOptions(target && typeof target.scrollTop === 'number' ? target.scrollTop : 0);
              if (input && input.focus) input.focus();
              window.setTimeout(function(){ keepOpen = false; }, 0);
              return;
            }
            current.__selectedValues = [value];
            current.__selectedItems = [{value:value, label:labelText}];
            current.__initialHydrated = true;
            save(current);
            ensureHiddenInputs(current);
            renderSelected(current);
            keepOpen = false;
            close();
        }, function(value){ keepOpen = !!value; });
        Array.prototype.slice.call(target.querySelectorAll('[data-load-more]')).forEach(function(button){
          button.onmousedown = function(event){
            event.preventDefault();
            keepOpen = true;
          };
          button.onclick = async function(){
            suppressRelationAutoClose();
            const current = state();
            const target = portal() || list;
            const previousScrollTop = target ? target.scrollTop : 0;
            await loadOptions(currentQuery(current), true, previousScrollTop);
            window.setTimeout(function(){ keepOpen = false; }, 0);
          };
        });
        Array.prototype.slice.call(target.querySelectorAll('[data-load-all]')).forEach(function(button){
          button.onmousedown = function(event){
            event.preventDefault();
            keepOpen = true;
          };
          button.onclick = async function(){
            suppressRelationAutoClose();
            const current = state();
            const limit = Math.max(1, Number(current.load_all_limit || 0) || 250);
            current.__page = 1;
            save(current);
            renderRelationLoading(shell);
            try {
              const fullResult = await fetchRelationPage(current, currentQuery(current), 1, limit);
              renderOptions(fullResult);
            } finally {
              window.setTimeout(function(){ keepOpen = false; }, 0);
            }
          };
        });
      }
      async function loadOptions(query, append, restoreScrollTop){
        const current = state();
        const normalizedQuery = String(query || '').trim();
        current.__query = normalizedQuery;
        const page = append ? (Number(current.__page || 1) || 1) + 1 : 1;
        current.__page = page;
        save(current);
        if (!append) {
          const cached = cachedResult(current, normalizedQuery);
          if (cached) {
            renderOptions(cached);
            return;
          }
          renderRelationLoading(shell);
        }
        try {
          const fetched = await fetchRelationPage(current, normalizedQuery, page, current.async_limit);
          if (append) {
            const base = cachedResult(current, normalizedQuery) || {options:[], page:0, limit:fetched.limit, total:fetched.total, has_more:true};
            const merged = {
              options: (base.options || []).concat(fetched.options || []),
              page: fetched.page,
              limit: fetched.limit,
              total: fetched.total,
              has_more: fetched.has_more
            };
            renderOptions(merged, restoreScrollTop);
            return;
          }
          renderOptions(fetched, restoreScrollTop);
        } catch (_err) {
          close();
        }
      }
      function openFromCacheOrLoad(){
        const current = state();
        const query = currentQuery(current);
        const cached = cachedResult(current, query);
        if (cached) {
          renderOptions(cached);
          return;
        }
        loadOptions(query, false);
      }
      function commitActive(){
        const dropdown = portal();
        const target = dropdown && isOpen() ? dropdown : list;
        const button = picker.activeButton(target);
        if (button) button.click();
      }
      const controller = {
        open: openFromCacheOrLoad,
        close: close,
        move: function(delta){
          const dropdown = portal();
          const target = dropdown && isOpen() ? dropdown : list;
          if (!isOpen()) {
            openFromCacheOrLoad();
            return;
          }
          picker.setActiveIndex(target, picker.activeIndex() + delta);
        },
        commit: commitActive
      };
      window.__gmcoreCrudRelationKeyboardBound = window.__gmcoreCrudRelationKeyboardBound || {};
      if (!window.__gmcoreCrudRelationKeyboardBound[state.instanceId]) {
        window.__gmcoreCrudRelationKeyboardBound[state.instanceId] = true;
        document.addEventListener('keydown', function(event){
          const active = activeRelationController();
          if (!active) return;
          if (event.key === 'ArrowDown') {
            event.preventDefault();
            active.move(1);
            return;
          }
          if (event.key === 'ArrowUp') {
            event.preventDefault();
            active.move(-1);
            return;
          }
          if (event.key === 'Enter') {
            event.preventDefault();
            active.commit();
            return;
          }
          if (event.key === 'Escape') {
            event.preventDefault();
            active.close();
          }
        }, true);
      }
      function initSelection(){
        const current = state();
        ensureHiddenInputs(current);
        renderSelected(current);
      }
      initSelection();
      input.onfocus = function(){ openFromCacheOrLoad(); };
      input.onclick = function(){ openFromCacheOrLoad(); };
      input.oninput = function(){
        if (!isMultiple && hidden) hidden.value = '';
        const current = state();
        current.__selectedValues = isMultiple ? current.__selectedValues : [];
        current.__selectedItems = isMultiple ? current.__selectedItems : [];
        save(current);
        if (timer) window.clearTimeout(timer);
        timer = window.setTimeout(function(){
          loadOptions(input.value || '', false);
        }, Math.max(0, Number(current.async_debounce || 0) || 150));
      };
      input.onkeydown = function(event){
        if (isMultiple && event.key === 'Backspace' && !String(input.value || '').trim()) {
          const current = state();
          if ((current.__selectedValues || []).length) {
            current.__selectedValues.pop();
            current.__selectedItems.pop();
            current.__initialHydrated = true;
            save(current);
            ensureHiddenInputs(current);
            renderSelected(current);
            rerenderCurrentOptions();
          }
        }
      };
      input.onblur = function(){
        if (keepOpen) return;
      };
    });
  }
  function bindChoiceWidgets(root){
    Array.prototype.slice.call((root || document).querySelectorAll('[data-crud-choice-shell]')).forEach(function(shell){
      if (shell.getAttribute('data-crud-choice-bound') === '1') return;
      shell.setAttribute('data-crud-choice-bound', '1');
      const input = shell.querySelector('[data-crud-choice-input]');
      const hidden = shell.querySelector('[data-crud-choice-hidden]');
      const hiddenList = shell.querySelector('[data-crud-choice-hidden-list]');
      const selectedBox = shell.querySelector('[data-crud-choice-selected]');
      const list = shell.querySelector('[data-crud-choice-options]');
      if (!input || !list) return;
      const isMultiple = shell.getAttribute('data-crud-choice-multiple') === '1';
      const picker = createCrudChoicePicker({
        shell: shell,
        input: input,
        list: list,
        portalName: 'crud-floating-choice',
        anchorAttr: 'data-crud-choice-anchor',
        valueAttr: 'data-choice-value',
        labelAttr: 'data-choice-label',
        valueSelector: '[data-choice-value]'
      });
      let keepOpen = false;
      let timer = null;
      function portal(){
        return ensureFloatingPortal('crud-floating-choice', themeForNode(shell));
      }
      function isOpen(){
        return isFloatingPortalOpen('crud-floating-choice', shell.getAttribute('data-crud-choice-anchor') || '');
      }
      function close(){
        closeChoicePortal();
        list.classList.remove('is-open');
        picker.closeAccessibility();
        picker.resetActive();
      }
      function shellState(){
        let next = {};
        try {
          next = JSON.parse(shell.getAttribute('data-crud-choice-field') || '{}') || {};
        } catch (_err) {
          next = {};
        }
        if (!Array.isArray(next.__selectedValues)) next.__selectedValues = [];
        if (!Array.isArray(next.__selectedItems)) next.__selectedItems = [];
        return next;
      }
      function save(next){
        shell.setAttribute('data-crud-choice-field', JSON.stringify(next || {}));
      }
      function fieldOptions(){
        const current = shellState();
        return Array.isArray(current.options) ? current.options : [];
      }
      function visibleOptions(nextState, query){
        return picker.visibleOptions(fieldOptions(), nextState.__selectedValues || [], query, false);
      }
      function ensureHiddenInputs(nextState){
        if (isMultiple) {
          if (!hiddenList) return;
          hiddenList.innerHTML = (nextState.__selectedValues || []).map(function(value){
            return '<input type="hidden" name="' + esc(input.getAttribute('data-crud-name') || '') + '" value="' + esc(value) + '">';
          }).join('');
          return;
        }
        if (hidden) hidden.value = nextState.__selectedValues[0] || '';
      }
      function renderSelected(nextState){
        if (isMultiple && selectedBox) {
          selectedBox.innerHTML = (nextState.__selectedItems || []).map(function(item){
            return picker.chipMarkup(item, 'data-remove-choice-value');
          }).join('');
          Array.prototype.slice.call(selectedBox.querySelectorAll('[data-remove-choice-value]')).forEach(function(button){
          button.onclick = function(){
            const current = shellState();
            const value = String(button.getAttribute('data-remove-choice-value') || '');
            suppressChoiceAutoClose();
            keepOpen = true;
            current.__selectedValues = (current.__selectedValues || []).filter(function(item){ return String(item) !== value; });
            current.__selectedItems = (current.__selectedItems || []).filter(function(item){ return String(item.value) !== value; });
            current.__initialHydrated = true;
              save(current);
              ensureHiddenInputs(current);
              renderSelected(current);
              renderOptions(current, input.value || '');
            if (input.focus) input.focus();
            window.setTimeout(function(){ keepOpen = false; }, 0);
          };
        });
          return;
        }
        const first = (nextState.__selectedItems || [])[0];
        input.value = first ? String(first.label || first.value || '') : '';
      }
      function hydrateInitial(){
        const current = shellState();
        if (current.__initialHydrated) return current;
        const initialValues = isMultiple ? asArray(shell.getAttribute('data-crud-choice-initial') || '') : [];
        const initialLabels = isMultiple ? asArray(shell.getAttribute('data-crud-choice-initial-labels') || '') : [];
        if (isMultiple) {
          current.__selectedValues = initialValues.slice();
          current.__selectedItems = initialValues.map(function(value, index){
            return {value: String(value), label: String(initialLabels[index] || value)};
          });
          current.__initialHydrated = true;
        } else if (hidden && String(hidden.value || '').trim()) {
          const value = String(hidden.value || '').trim();
          current.__selectedValues = [value];
          current.__selectedItems = [{value: value, label: String(input.value || value)}];
          current.__initialHydrated = true;
        }
        save(current);
        return current;
      }
      function renderOptions(nextState, query){
        const options = visibleOptions(nextState, query);
        const showClear = !isMultiple && !nextState.required && (nextState.__selectedValues || []).length > 0;
        const html = (showClear ? '<button type="button" class="crud-relation-option crud-relation-option-clear" data-clear-choice="1">' + esc(label('clear_selection', 'Clear selection')) + '</button>' : '') + options.map(function(option){
          return picker.optionMarkup(option, {valueAttr:'data-choice-value', labelAttr:'data-choice-label'});
        }).join('') || picker.metaMarkup(label('no_matches', 'No matches'));
        const target = picker.openPortal(html);
        picker.setActiveIndex(target, 0);
        picker.bindOptions(target, '[data-choice-value]', function(button){
            const current = shellState();
            const value = String(button.getAttribute('data-choice-value') || '');
            const labelText = String(button.getAttribute('data-choice-label') || value);
            if (isMultiple) {
              suppressChoiceAutoClose();
              if ((current.__selectedValues || []).indexOf(value) < 0) {
                current.__selectedValues.push(value);
                current.__selectedItems.push({value: value, label: labelText});
              }
              current.__initialHydrated = true;
              save(current);
              ensureHiddenInputs(current);
              renderSelected(current);
              input.value = '';
              renderOptions(current, '');
              if (input.focus) input.focus();
              window.setTimeout(function(){ keepOpen = false; }, 0);
              return;
            }
            current.__selectedValues = [value];
            current.__selectedItems = [{value: value, label: labelText}];
            current.__initialHydrated = true;
            save(current);
            ensureHiddenInputs(current);
            renderSelected(current);
            keepOpen = false;
            close();
        }, function(value){ keepOpen = !!value; });
        Array.prototype.slice.call(target.querySelectorAll('[data-clear-choice]')).forEach(function(button){
          button.onmousedown = function(event){
            event.preventDefault();
            keepOpen = true;
          };
          button.onclick = function(){
            suppressChoiceAutoClose();
            const current = shellState();
            current.__selectedValues = [];
            current.__selectedItems = [];
            current.__initialHydrated = true;
            save(current);
            ensureHiddenInputs(current);
            renderSelected(current);
            input.value = '';
            renderOptions(current, '');
            if (input.focus) input.focus();
            window.setTimeout(function(){ keepOpen = false; }, 0);
          };
        });
      }
      function openChoices(){
        const current = hydrateInitial();
        renderOptions(current, isMultiple ? (input.value || '') : '');
      }
      if (!shell.getAttribute('data-crud-choice-anchor')) {
        shell.setAttribute('data-crud-choice-anchor', 'crud-choice-' + Math.random().toString(36).slice(2));
      }
      ensureHiddenInputs(hydrateInitial());
      renderSelected(hydrateInitial());
      input.onfocus = openChoices;
      input.onclick = openChoices;
      input.oninput = function(){
        if (!isMultiple && hidden) hidden.value = '';
        const current = shellState();
        if (!isMultiple) {
          current.__selectedValues = [];
          current.__selectedItems = [];
          save(current);
        }
        if (timer) window.clearTimeout(timer);
        timer = window.setTimeout(function(){
          renderOptions(shellState(), input.value || '');
        }, 100);
      };
      input.onkeydown = function(event){
        if (event.key === 'ArrowDown' || event.key === 'ArrowUp') {
          event.preventDefault();
          const target = portal() && isOpen() ? portal() : list;
          if (!isOpen()) {
            openChoices();
            return;
          }
          picker.setActiveIndex(target, picker.activeIndex() + (event.key === 'ArrowDown' ? 1 : -1));
          return;
        }
        if (event.key === 'Enter') {
          const target = portal() && isOpen() ? portal() : list;
          const button = picker.activeButton(target);
          if (button) {
            event.preventDefault();
            button.click();
            return;
          }
        }
        if (event.key === 'Escape') {
          event.preventDefault();
          close();
          return;
        }
        if (isMultiple && event.key === 'Backspace' && !String(input.value || '').trim()) {
          const current = shellState();
          if ((current.__selectedValues || []).length) {
            current.__selectedValues.pop();
            current.__selectedItems.pop();
            current.__initialHydrated = true;
            save(current);
            ensureHiddenInputs(current);
            renderSelected(current);
            renderOptions(current, '');
          }
        }
      };
      input.onblur = function(){
        if (keepOpen) return;
      };
    });
  }
  function formField(field, record, mode, previewOnly){
    if (field.hidden) return '';
    const value = record && record[field.name] != null ? record[field.name] : '';
    const disabledState = previewOnly || mode === 'show' || field.disabled || field.auto_managed;
    const readonly = !disabledState && (field.read_only || field.readOnly);
    const required = field.required && mode !== 'show' && !previewOnly ? ' required' : '';
    const name = esc(field.name);
    const labelText = esc(fieldLabelText(field));
    const helpText = helpMarkup(field);
    let widget = String(field.widget || '').trim();
    if (!widget && String(field.relation || '').trim()) {
      widget = field.multiple && !field.async_options ? 'multiselect' : 'relation';
    }
    if (!widget) widget = String(field.type || 'text');
    const style = fieldHeightStyle(field);
    const shellClass = 'crud-field-shell' + fieldToneClass(field);
    if (widget === 'hidden') {
      return '<input type="hidden" name="' + name + '" value="' + esc(value) + '">';
    }
    if (widget === 'checkbox' || widget === 'bool' || widget === 'switch' || widget === 'toggle' || field.type === 'bool') {
      const checked = boolValue(value) ? ' checked' : '';
      const disabledAttr = disabledState ? ' disabled' : '';
      return '<div class="' + shellClass + '" style="' + esc(style) + '"><label class="crud-checkbox-field"><input class="crud-check-input" type="checkbox" name="' + name + '"' + checked + disabledAttr + '><span class="crud-checkbox-copy"><span class="crud-field-label">' + labelText + requiredMark(field) + '</span><span class="crud-field-caption">' + esc(fieldHelpText(field)) + '</span></span></label></div>';
    }
    if (widget === 'select') {
      const displayValue = relationDisplayValue(field, record, value);
      return '<div class="' + shellClass + '" style="' + esc(style) + '">' +
        '<label class="crud-field-label">' + labelText + requiredMark(field) + '</label>' +
        '<div class="crud-choice-shell" data-crud-choice-shell data-crud-choice-field="' + esc(JSON.stringify(field)) + '">' +
          '<div class="crud-relation-input-shell"><input class="crud-input crud-relation-input" type="text" value="' + esc(displayValue) + '" placeholder="' + esc(field.placeholder || 'Select…') + '" data-crud-name="' + name + '"' + (disabledState ? ' disabled' : '') + (readonly ? ' readonly' : '') + ' data-crud-choice-input><img class="crud-icon crud-relation-caret" src="/assets/crud/icons/caret-down.svg" alt=""></div>' +
          '<input type="hidden" name="' + name + '" value="' + esc(value) + '" data-crud-choice-hidden>' +
          '<div class="crud-choice-options" data-crud-choice-options></div>' +
        '</div>' + helpText +
      '</div>';
    }
    if (widget === 'relation') {
      const displayValue = relationDisplayValue(field, record, value);
      if (field.async_options) {
        if (field.multiple) {
          const selectedValues = asArray(value);
          const selectedLabels = asArray(record && record[field.display_field] != null ? record[field.display_field] : '');
          return '<div class="' + shellClass + '" style="' + esc(style) + '">' +
            '<label class="crud-field-label">' + labelText + requiredMark(field) + '</label>' +
            '<div class="crud-relation-shell crud-relation-shell-multiple" data-crud-relation-shell data-crud-relation-multiple="1" data-crud-relation-field="' + esc(JSON.stringify(field)) + '" data-crud-relation-initial="' + esc(JSON.stringify(selectedValues)) + '" data-crud-relation-initial-labels="' + esc(JSON.stringify(selectedLabels)) + '">' +
              '<div class="crud-relation-selected" data-crud-relation-selected></div>' +
              '<div class="crud-relation-input-shell"><input class="crud-input crud-relation-input" type="text" value="" placeholder="' + esc(field.placeholder || 'Search…') + '" data-crud-name="' + name + '"' + (disabledState ? ' disabled' : '') + (readonly ? ' readonly' : '') + ' data-crud-relation-input><img class="crud-icon crud-relation-caret" src="/assets/crud/icons/caret-down.svg" alt=""></div>' +
              '<div class="crud-relation-hidden-list" data-crud-relation-hidden-list></div>' +
              '<div class="crud-relation-options" data-crud-relation-options></div>' +
            '</div>' +
          helpText + '</div>';
      }
      return '<div class="' + shellClass + '" style="' + esc(style) + '">' +
          '<label class="crud-field-label">' + labelText + requiredMark(field) + '</label>' +
          '<div class="crud-relation-shell" data-crud-relation-shell data-crud-relation-field="' + esc(JSON.stringify(field)) + '">' +
            '<div class="crud-relation-input-shell"><input class="crud-input crud-relation-input" type="text" value="' + esc(displayValue) + '" placeholder="' + esc(field.placeholder || 'Search…') + '" data-crud-name="' + name + '"' + (disabledState ? ' disabled' : '') + (readonly ? ' readonly' : '') + ' data-crud-relation-input><img class="crud-icon crud-relation-caret" src="/assets/crud/icons/caret-down.svg" alt=""></div>' +
            '<input type="hidden" name="' + name + '" value="' + esc(value) + '" data-crud-relation-hidden>' +
            '<div class="crud-relation-options" data-crud-relation-options></div>' +
          '</div>' + helpText +
        '</div>';
      }
      if (field.multiple) {
        const selected = asArray(value);
        const options = (field.options || []).map(function(option){
          const isSelected = selected.indexOf(String(option.value)) >= 0 ? ' selected' : '';
          return '<option value="' + esc(option.value) + '"' + isSelected + '>' + esc(option.label) + '</option>';
        }).join('');
        return '<div class="' + shellClass + '" style="' + esc(style) + '"><label class="crud-field-label">' + labelText + requiredMark(field) + '</label><select class="crud-select" name="' + name + '" multiple' + (disabledState ? ' disabled' : '') + required + '>' + options + '</select>' + helpText + '</div>';
      }
      const placeholderOption = field.required ? '' : '<option value="">' + esc(field.placeholder || 'Select an option') + '</option>';
      const options = (field.options || []).map(function(option){
        const selected = String(option.value) === String(value) ? ' selected' : '';
        return '<option value="' + esc(option.value) + '"' + selected + '>' + esc(option.label) + '</option>';
      }).join('');
      return '<div class="' + shellClass + '" style="' + esc(style) + '"><label class="crud-field-label">' + labelText + requiredMark(field) + '</label><select class="crud-select" name="' + name + '"' + (disabledState ? ' disabled' : '') + required + '>' + placeholderOption + options + '</select>' + helpText + '</div>';
    }
    if (widget === 'multiselect') {
      const selected = asArray(value);
      const selectedLabels = asArray(record && record[field.display_field] != null ? record[field.display_field] : '');
      return '<div class="' + shellClass + '" style="' + esc(style) + '">' +
        '<label class="crud-field-label">' + labelText + requiredMark(field) + '</label>' +
        '<div class="crud-choice-shell crud-choice-shell-multiple" data-crud-choice-shell data-crud-choice-multiple="1" data-crud-choice-field="' + esc(JSON.stringify(field)) + '" data-crud-choice-initial="' + esc(JSON.stringify(selected)) + '" data-crud-choice-initial-labels="' + esc(JSON.stringify(selectedLabels)) + '">' +
          '<div class="crud-relation-selected" data-crud-choice-selected></div>' +
          '<div class="crud-relation-input-shell"><input class="crud-input crud-relation-input" type="text" value="" placeholder="' + esc(field.placeholder || 'Search…') + '" data-crud-name="' + name + '"' + (disabledState ? ' disabled' : '') + (readonly ? ' readonly' : '') + ' data-crud-choice-input><img class="crud-icon crud-relation-caret" src="/assets/crud/icons/caret-down.svg" alt=""></div>' +
          '<div class="crud-choice-hidden-list" data-crud-choice-hidden-list></div>' +
          '<div class="crud-choice-options" data-crud-choice-options></div>' +
        '</div>' + helpText +
      '</div>';
    }
    if (widget === 'wysiwyg' || widget === 'wysiwyg_min' || widget === 'wysiwyg_full') {
      const wysiwygMode = widget === 'wysiwyg_full' ? 'full' : 'min';
      const disabledClass = (readonly || disabledState) ? ' is-readonly' : '';
      const disabledAttr = disabledState ? ' disabled' : '';
      const toolbar = wysiwygMode === 'full'
        ? '<div class="crud-wysiwyg-toolbar-group">' +
            '<button type="button" class="crud-wysiwyg-btn" data-crud-wysiwyg-action="paragraph"' + disabledAttr + '>P</button>' +
            '<button type="button" class="crud-wysiwyg-btn" data-crud-wysiwyg-action="h2"' + disabledAttr + '>H2</button>' +
            '<button type="button" class="crud-wysiwyg-btn" data-crud-wysiwyg-action="h3"' + disabledAttr + '>H3</button>' +
            '<button type="button" class="crud-wysiwyg-btn" data-crud-wysiwyg-action="quote"' + disabledAttr + '>❝</button>' +
          '</div>' +
          '<div class="crud-wysiwyg-toolbar-group">' +
            '<button type="button" class="crud-wysiwyg-btn" data-crud-wysiwyg-action="bold"' + disabledAttr + '><strong>B</strong></button>' +
            '<button type="button" class="crud-wysiwyg-btn" data-crud-wysiwyg-action="italic"' + disabledAttr + '><em>I</em></button>' +
            '<button type="button" class="crud-wysiwyg-btn" data-crud-wysiwyg-action="underline"' + disabledAttr + '><u>U</u></button>' +
            '<button type="button" class="crud-wysiwyg-btn" data-crud-wysiwyg-action="strike"' + disabledAttr + '><span style="text-decoration:line-through;">S</span></button>' +
          '</div>' +
          '<div class="crud-wysiwyg-toolbar-group">' +
            '<button type="button" class="crud-wysiwyg-btn" data-crud-wysiwyg-action="ul"' + disabledAttr + '>•</button>' +
            '<button type="button" class="crud-wysiwyg-btn" data-crud-wysiwyg-action="ol"' + disabledAttr + '>1.</button>' +
            '<button type="button" class="crud-wysiwyg-btn" data-crud-wysiwyg-action="link"' + disabledAttr + '>🔗</button>' +
          '</div>' +
          '<div class="crud-wysiwyg-toolbar-group">' +
            '<select class="crud-select crud-select-sm crud-wysiwyg-select" data-crud-wysiwyg-select="font_size"' + disabledAttr + '>' +
              '<option value="">Size</option>' +
              '<option value="2">S</option>' +
              '<option value="3">M</option>' +
              '<option value="4">L</option>' +
              '<option value="5">XL</option>' +
            '</select>' +
            '<button type="button" class="crud-wysiwyg-btn" data-crud-wysiwyg-action="align" data-crud-wysiwyg-value="left"' + disabledAttr + '>⟸</button>' +
            '<button type="button" class="crud-wysiwyg-btn" data-crud-wysiwyg-action="align" data-crud-wysiwyg-value="center"' + disabledAttr + '>≡</button>' +
            '<button type="button" class="crud-wysiwyg-btn" data-crud-wysiwyg-action="align" data-crud-wysiwyg-value="right"' + disabledAttr + '>⟹</button>' +
          '</div>' +
          '<div class="crud-wysiwyg-toolbar-group">' +
            '<input class="crud-wysiwyg-color" type="color" value="#111827" data-crud-wysiwyg-color' + disabledAttr + '>' +
            '<button type="button" class="crud-wysiwyg-btn" data-crud-wysiwyg-action="clear"' + disabledAttr + '>⌫</button>' +
          '</div>'
        : '<div class="crud-wysiwyg-toolbar-group">' +
            '<button type="button" class="crud-wysiwyg-btn" data-crud-wysiwyg-action="bold"' + disabledAttr + '><strong>B</strong></button>' +
            '<button type="button" class="crud-wysiwyg-btn" data-crud-wysiwyg-action="italic"' + disabledAttr + '><em>I</em></button>' +
            '<button type="button" class="crud-wysiwyg-btn" data-crud-wysiwyg-action="underline"' + disabledAttr + '><u>U</u></button>' +
          '</div>' +
          '<div class="crud-wysiwyg-toolbar-group">' +
            '<button type="button" class="crud-wysiwyg-btn" data-crud-wysiwyg-action="ul"' + disabledAttr + '>•</button>' +
            '<button type="button" class="crud-wysiwyg-btn" data-crud-wysiwyg-action="ol"' + disabledAttr + '>1.</button>' +
            '<button type="button" class="crud-wysiwyg-btn" data-crud-wysiwyg-action="link"' + disabledAttr + '>🔗</button>' +
          '</div>';
      return '<div class="' + shellClass + '" style="' + esc(style) + '">' +
        '<label class="crud-field-label">' + labelText + requiredMark(field) + '</label>' +
        '<div class="crud-wysiwyg-shell' + disabledClass + '" data-crud-wysiwyg data-crud-wysiwyg-mode="' + esc(wysiwygMode) + '">' +
          '<div class="crud-wysiwyg-toolbar" data-crud-wysiwyg-toolbar>' +
            toolbar +
          '</div>' +
          '<div class="crud-wysiwyg-editor crud-input" contenteditable="' + ((readonly || disabledState) ? 'false' : 'true') + '" data-placeholder="' + esc(field.placeholder || 'Start writing…') + '" data-crud-wysiwyg-editor>' + String(value || '') + '</div>' +
          '<input type="hidden" name="' + name + '" value="' + esc(value) + '" data-crud-wysiwyg-hidden>' +
        '</div>' + helpText +
      '</div>';
    }
    if (widget === 'textarea' || field.type === 'textarea' || field.type === 'json') {
      const rows = Math.max(3, Number(field.rows || field.height || 4) || 4);
      const codeClass = (widget === 'code' || field.type === 'json') ? ' crud-textarea-code' : '';
      return '<div class="' + shellClass + '" style="' + esc(style) + '"><label class="crud-field-label">' + labelText + requiredMark(field) + '</label><textarea class="crud-input crud-textarea' + codeClass + '" name="' + name + '" rows="' + rows + '" placeholder="' + esc(field.placeholder || '') + '"' + (disabledState ? ' disabled' : '') + (readonly ? ' readonly' : '') + required + '>' + esc(value) + '</textarea>' + helpText + '</div>';
    }
    if (widget === 'range') {
      const currentValue = String(value == null || value === '' ? '0' : value);
      return '<div class="' + shellClass + '" style="' + esc(style) + '"><label class="crud-field-label">' + labelText + requiredMark(field) + '</label><div class="crud-range-shell"><input class="form-range" type="range" name="' + name + '" min="0" max="10" step="' + esc(field.step || '1') + '" value="' + esc(currentValue) + '"' + (disabledState ? ' disabled' : '') + required + ' data-crud-range-input><div class="crud-range-value" data-crud-range-value>' + esc(currentValue) + '</div></div>' + helpText + '</div>';
    }
    const inputType = widget === 'password' ? 'password' : widget === 'email' ? 'email' : widget === 'url' ? 'url' : widget === 'tel' ? 'tel' : widget === 'color' ? 'color' : widget === 'time' ? 'time' : widget === 'date' ? 'date' : (widget === 'datetime-local' || widget === 'datetime' ? 'datetime-local' : (widget === 'number' ? 'number' : 'text'));
    const extraClass = widget === 'color' ? ' crud-input-color' : '';
    return '<div class="' + shellClass + '" style="' + esc(style) + '"><label class="crud-field-label">' + labelText + requiredMark(field) + '</label><input class="crud-input' + extraClass + '" type="' + inputType + '" name="' + name + '" value="' + esc(value) + '" placeholder="' + esc(field.placeholder || '') + '"' + (disabledState ? ' disabled' : '') + (readonly ? ' readonly' : '') + required + '>' + helpText + '</div>';
  }
  function resolvePrimaryButtonLabel(mode){
    if (mode === 'clone') return label('clone', 'Clone');
    if (mode === 'create') return label('create', 'Create');
    return label('save', 'Save');
  }
  function renderForm(fields, record, mode, key){
    const body = byId('crud-entry-body');
    const title = byId('crud-entry-title');
    const actions = byId('crud-entry-actions');
    const note = byId('crud-entry-note');
    if (!body || !title || !actions || !note) return;
    title.textContent = mode === 'create' ? label('create', 'Create') : mode === 'edit' ? label('edit', 'Edit') : mode === 'clone' ? label('clone', 'Clone') : label('view', 'View');
    note.textContent = key ? ('Key: ' + key) : designerModeLabel(normalizeMode(mode));
    const formId = 'crud-entry-form';
    currentModalFields = Array.isArray(fields) ? fields.slice() : [];
    body.innerHTML = '<form id="' + formId + '"><input type="hidden" name="_key" value="' + esc(key || '') + '"><div class="crud-form-grid">' +
      fields.filter(function(field){ return !field.hidden; }).map(function(field){
        return '<div class="crud-form-col ' + fieldSpanClass(field) + '">' + formField(field, record, mode, false) + '</div>';
      }).join('') + '</div></form>';
    const buttons = [];
    if (mode !== 'show') {
      buttons.push('<button type="button" class="crud-btn crud-btn-outline-secondary" data-crud-close>' + esc(label('close', 'Close')) + '</button>');
      buttons.push('<button type="button" class="crud-btn crud-btn-primary" data-crud-submit>' + esc(resolvePrimaryButtonLabel(mode)) + '</button>');
    } else {
      buttons.push('<button type="button" class="crud-btn crud-btn-outline-secondary" data-crud-close>' + esc(label('close', 'Close')) + '</button>');
    }
    actions.innerHTML = buttons.join('');
    actions.querySelectorAll('[data-crud-close]').forEach(function(btn){
      btn.onclick = hideEntryModal;
    });
    actions.querySelectorAll('[data-crud-submit]').forEach(function(btn){
      btn.onclick = function(){ submitCurrentForm(mode); };
    });
    body.querySelectorAll('[data-crud-range-input]').forEach(function(input){
      const shell = input.closest('.crud-range-shell');
      const valueNode = shell ? shell.querySelector('[data-crud-range-value]') : null;
      const sync = function(){ if (valueNode) valueNode.textContent = String(input.value || '0'); };
      input.addEventListener('input', sync);
      input.addEventListener('change', sync);
      sync();
    });
    bindRelationWidgets(body);
    bindChoiceWidgets(body);
    bindWysiwygWidgets(body);
  }
  function serializeForm(form){
    const payload = {};
    Array.from(form.elements || []).forEach(function(element){
      if (!element.name || element.disabled) return;
      const key = String(element.name || '').replace(/\[\]$/, '');
      if (element.type === 'checkbox') {
        payload[key] = element.checked ? 'true' : 'false';
        return;
      }
      if (element.multiple) {
        payload[key] = Array.from(element.selectedOptions || []).map(function(option){ return option.value; });
        return;
      }
      if (payload[key] !== undefined) {
        if (!Array.isArray(payload[key])) payload[key] = [payload[key]];
        payload[key].push(element.value);
        return;
      }
      payload[key] = element.value;
    });
    return payload;
  }
  async function openCreate(){
    const meta = await fetchFormMeta('create');
    renderForm((meta.form && meta.form.fields) || formFields, {}, 'create', '');
    showEntryModal();
  }
  async function openExisting(mode, key, url){
    const normalizedMode = normalizeMode(mode);
    const meta = await fetchFormMeta(normalizedMode);
    const requestURL = url || (apiURL('/api/' + encodeURIComponent(key)) + '?mode=' + encodeURIComponent(mode || normalizedMode));
    const record = await fetchJSON(requestURL, {credentials:'same-origin'});
    const nextMode = mode === 'view' ? 'show' : mode;
    const nextRecord = mode === 'clone' ? cloneRecord(record) : record;
    const nextKey = mode === 'clone' ? '' : key;
    renderForm((meta.form && meta.form.fields) || formFields, nextRecord, nextMode, nextKey);
    showEntryModal();
  }
  function cloneRecord(record){
    const next = Object.assign({}, record || {});
    delete next[primaryKey];
    delete next.id;
    delete next.key;
    return next;
  }
  async function deleteRecord(key){
    if (!window.confirm(label('confirm_delete', 'Delete record?'))) return;
    await fetchJSON(apiURL('/api/delete'), {method:'POST', credentials:'same-origin', headers:headers, body:JSON.stringify({_key:key})});
    await load();
    dispatchMutation('delete', {recordKey:key});
  }
  async function submitCurrentForm(mode){
    const form = byId('crud-entry-form');
    if (!form) return;
    clearFieldErrors(form);
    const validationErrors = validateFormFields(form, currentModalFields || []);
    if (Object.keys(validationErrors).length) {
      renderFieldErrors(form, validationErrors);
      return;
    }
    const payload = serializeForm(form);
    const path = mode === 'create' || mode === 'clone' ? '/api/create' : '/api/update';
    try {
      const response = await fetchJSON(apiURL(path), {method:'POST', credentials:'same-origin', headers:headers, body:JSON.stringify(payload)});
      hideEntryModal();
      await load();
      dispatchMutation(mode === 'create' || mode === 'clone' ? 'create' : 'update', {
        path: path,
        recordKey: payload._key || '',
        response: response
      });
    } catch (error) {
      const payloadErrors = error && error.payload && error.payload.errors;
      if (payloadErrors && typeof payloadErrors === 'object') {
        renderFieldErrors(form, payloadErrors);
        return;
      }
      throw error;
    }
  }
  function showEntryModal(){
    const modalEl = byId('crud-entry-modal');
    const instance = window.gmcoreCrudModal && window.gmcoreCrudModal.getOrCreateInstance ? window.gmcoreCrudModal.getOrCreateInstance(modalEl) : null;
    if (instance) instance.show();
  }
  function hideEntryModal(){
    const modalEl = byId('crud-entry-modal');
    const instance = window.gmcoreCrudModal && window.gmcoreCrudModal.getOrCreateInstance ? window.gmcoreCrudModal.getOrCreateInstance(modalEl) : null;
    closeAllFloatingPortals();
    if (instance) instance.hide();
  }
