  function createCrudChoicePicker(config){
    const cfg = config || {};
    const shell = cfg.shell || null;
    const input = cfg.input || null;
    const list = cfg.list || null;
    let activeIndex = -1;
    function anchorID(){
      return shell ? String(shell.getAttribute(cfg.anchorAttr || '') || '') : '';
    }
    function portalElementID(){
      return 'crud-choice-picker-' + String(cfg.portalName || 'choice').replace(/[^a-z0-9_-]+/gi, '-') + '-' + anchorID();
    }
    function setupInputAccessibility(){
      if (!input) return;
      input.setAttribute('role', 'combobox');
      input.setAttribute('aria-haspopup', 'listbox');
      input.setAttribute('aria-expanded', 'false');
      if (anchorID()) input.setAttribute('aria-controls', portalElementID());
    }
    function optionMarkup(option, options){
      options = options || {};
      const value = option && option.value != null ? option.value : '';
      const labelText = option && option.label != null ? option.label : value;
      const className = String(options.className || cfg.optionClass || 'crud-relation-option');
      const valueAttr = String(options.valueAttr || cfg.valueAttr || 'data-value');
      const labelAttr = String(options.labelAttr || cfg.labelAttr || 'data-label');
      const selected = !!options.selected;
      return '<button type="button" role="option" aria-selected="' + (selected ? 'true' : 'false') + '" class="' + esc(className) + (selected ? ' is-selected' : '') + '" ' + esc(valueAttr) + '="' + esc(value) + '" ' + esc(labelAttr) + '="' + esc(labelText) + '">' + esc(labelText) + '</button>';
    }
    function metaMarkup(text, extraClass){
      return '<div class="crud-relation-meta' + (extraClass ? ' ' + esc(extraClass) : '') + '"><span class="crud-relation-counter">' + esc(text) + '</span></div>';
    }
    function chipMarkup(item, removeAttr){
      return '<button type="button" class="crud-relation-chip" ' + esc(removeAttr) + '="' + esc(item.value) + '"><span class="crud-relation-chip-label">' + esc(item.label || item.value) + '</span><span class="crud-relation-chip-remove">&times;</span></button>';
    }
    function openPortal(html, restoreScrollTop){
      if (!shell) return null;
      const portal = openCrudPortal({
        name: cfg.portalName,
        anchor: shell,
        html: html,
        anchorAttr: cfg.anchorAttr,
        minWidth: Math.max(260, shell.getBoundingClientRect().width || 0),
        gap: 6,
        restoreScrollTop: restoreScrollTop
      });
      if (portal) {
        portal.id = portalElementID();
        portal.setAttribute('role', 'listbox');
        if (input) {
          input.setAttribute('aria-expanded', 'true');
          input.setAttribute('aria-controls', portal.id);
        }
        return portal;
      }
      if (!list) return null;
      list.innerHTML = html;
      list.id = portalElementID();
      list.setAttribute('role', 'listbox');
      list.classList.add('is-open');
      if (input) {
        input.setAttribute('aria-expanded', 'true');
        input.setAttribute('aria-controls', list.id);
      }
      if (typeof restoreScrollTop === 'number' && restoreScrollTop > 0) {
        window.requestAnimationFrame(function(){ list.scrollTop = restoreScrollTop; });
      }
      return list;
    }
    function closeAccessibility(){
      if (input) input.setAttribute('aria-expanded', 'false');
    }
    function setActiveIndex(target, nextIndex){
      const buttons = Array.prototype.slice.call((target || list || document).querySelectorAll(cfg.valueSelector || '[data-value]'));
      if (!buttons.length) {
        activeIndex = -1;
        return;
      }
      activeIndex = Math.max(0, Math.min(buttons.length - 1, nextIndex));
      buttons.forEach(function(button, index){
        const active = index === activeIndex;
        button.classList.toggle('is-active', active);
        if (button.hasAttribute('aria-selected')) button.setAttribute('aria-selected', active ? 'true' : 'false');
      });
      if (buttons[activeIndex] && buttons[activeIndex].scrollIntoView) {
        buttons[activeIndex].scrollIntoView({block:'nearest'});
      }
    }
    function bindOptions(target, selector, callback, keepOpenCallback){
      Array.prototype.slice.call((target || list || document).querySelectorAll(selector)).forEach(function(button){
        button.onmousedown = function(event){
          event.preventDefault();
          if (keepOpenCallback) keepOpenCallback(true);
        };
        button.onclick = function(event){
          event.preventDefault();
          callback(button, target || list);
        };
      });
    }
    function visibleOptions(options, selectedValues, query, allowSelected){
      const q = String(query || '').trim().toLowerCase();
      const selected = (selectedValues || []).map(function(value){ return String(value || ''); });
      return (Array.isArray(options) ? options : []).filter(function(option){
        if (!option) return false;
        const value = String(option.value == null ? '' : option.value);
        const labelText = String(option.label == null ? value : option.label);
        if (!allowSelected && selected.indexOf(value) >= 0) return false;
        if (!q) return true;
        return labelText.toLowerCase().indexOf(q) >= 0 || value.toLowerCase().indexOf(q) >= 0;
      });
    }
    function activeButton(target){
      const buttons = Array.prototype.slice.call((target || list || document).querySelectorAll(cfg.valueSelector || '[data-value]'));
      return activeIndex >= 0 ? buttons[activeIndex] : null;
    }
    setupInputAccessibility();
    return {
      optionMarkup: optionMarkup,
      metaMarkup: metaMarkup,
      chipMarkup: chipMarkup,
      openPortal: openPortal,
      closeAccessibility: closeAccessibility,
      setActiveIndex: setActiveIndex,
      bindOptions: bindOptions,
      visibleOptions: visibleOptions,
      activeButton: activeButton,
      activeIndex: function(){ return activeIndex; },
      resetActive: function(){ activeIndex = -1; }
    };
  }
