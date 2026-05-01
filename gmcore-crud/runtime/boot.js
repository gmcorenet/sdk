
  document.addEventListener('pointerdown', function(event){
    if (!event.target.closest('[data-crud-menu]') && !event.target.closest('[data-crud-global-portal="' + portalID('crud-floating-menu') + '"]')) {
      closeFloatingMenu();
    }
    if (relationSuppressionUntil() > Date.now()) {
      return;
    }
    if (!event.target.closest('[data-crud-relation-shell]') && !event.target.closest('[data-crud-global-portal="' + portalID('crud-floating-relation') + '"]')) {
      closeRelationPortal();
    }
    if (typeof choiceSuppressionUntil === 'function' && choiceSuppressionUntil() > Date.now()) {
      return;
    }
    if (!event.target.closest('[data-crud-choice-shell]') && !event.target.closest('[data-crud-global-portal="' + portalID('crud-floating-choice') + '"]')) {
      closeChoicePortal();
    }
  }, true);
  document.addEventListener('click', function(event){
    if (!event.target.closest('[data-crud-menu]') && !event.target.closest('[data-crud-global-portal="' + portalID('crud-floating-menu') + '"]')) {
      closeFloatingMenu();
    }
    if (relationSuppressionUntil() > Date.now()) {
      return;
    }
    if (!event.target.closest('[data-crud-relation-shell]') && !event.target.closest('[data-crud-global-portal="' + portalID('crud-floating-relation') + '"]')) {
      closeRelationPortal();
    }
    if (typeof choiceSuppressionUntil === 'function' && choiceSuppressionUntil() > Date.now()) {
      return;
    }
    if (!event.target.closest('[data-crud-choice-shell]') && !event.target.closest('[data-crud-global-portal="' + portalID('crud-floating-choice') + '"]')) {
      closeChoicePortal();
    }
  });
  window.addEventListener('gmcore:crud-mutated', function(event){
    const detail = event && event.detail ? event.detail : {};
    if (String(detail.sourceInstanceId || '') === String(state.instanceId || '')) return;
    if (String(detail.sourceBasePath || '') === String(basePath || '')) return;
    load();
  });
  window.addEventListener('resize', function(){ closeAllFloatingPortals(); });
  root.querySelectorAll('.crud-modal-body, .crud-table-card, .crud-table-wrap').forEach(function(node){
    if (node.getAttribute('data-crud-scroll-bound') === '1') return;
    node.setAttribute('data-crud-scroll-bound', '1');
    node.addEventListener('scroll', function(){
      closeFloatingMenu();
      closeRelationPortal();
      closeChoicePortal();
    }, {passive:true});
  });
  if (byId('crud-search')) byId('crud-search').oninput = function(event){ state.q = (event.target.value || '').trim(); state.page = 1; load(); };
  if (byId('crud-first')) byId('crud-first').onclick = function(){ state.page = 1; load(); };
  if (byId('crud-prev')) byId('crud-prev').onclick = function(){ state.page = Math.max(1, state.page - 1); load(); };
  if (byId('crud-next')) byId('crud-next').onclick = function(){ state.page = Math.min(state.totalPages, state.page + 1); load(); };
  if (byId('crud-last')) byId('crud-last').onclick = function(){ state.page = state.totalPages; load(); };
  if (byId('crud-per-page')) byId('crud-per-page').onchange = function(event){ state.perPage = parseInt(event.target.value || '10', 10) || 10; state.page = 1; load(); };
  if (byId('crud-open-filters')) byId('crud-open-filters').onclick = function(){
    const modalEl = byId('crud-filters-modal');
    const instance = window.gmcoreCrudModal && window.gmcoreCrudModal.getOrCreateInstance ? window.gmcoreCrudModal.getOrCreateInstance(modalEl) : null;
    if (instance) instance.show();
  };
  if (byId('crud-open-create')) byId('crud-open-create').onclick = function(){ openCreate(); };
  if (byId('crud-open-designer')) byId('crud-open-designer').onclick = function(){ openDesigner('create'); };
  all('[data-crud-modal-close]').forEach(function(button){
    button.addEventListener('click', function(){
      closeAllFloatingPortals();
    });
  });
  document.addEventListener('gmcore:crud-modal-hidden', function(event){
    const modal = event && event.target ? event.target : null;
    if (!modal) return;
    if (!root.contains(modal)) return;
    closeAllFloatingPortals();
  });
  if (byId('crud-apply-filters')) byId('crud-apply-filters').onclick = function(){
    const next = {};
    root.querySelectorAll('[data-filter-value]').forEach(function(input){
      const field = input.getAttribute('data-filter-value') || '';
      const value = input.value || '';
      if (!field || !String(value).trim()) return;
      const op = root.querySelector('[data-filter-op="' + CSS.escape(field) + '"]');
      next[field] = {value:String(value).trim(), op: op ? op.value : 'like'};
    });
    state.filters = next;
    state.page = 1;
    const modalEl = byId('crud-filters-modal');
    const instance = window.gmcoreCrudModal && window.gmcoreCrudModal.getOrCreateInstance ? window.gmcoreCrudModal.getOrCreateInstance(modalEl) : null;
    if (instance) instance.hide();
    load();
  };
  if (byId('crud-designer-save')) byId('crud-designer-save').onclick = function(){ saveDesignerMode(); };
  if (byId('crud-designer-reset')) byId('crud-designer-reset').onclick = function(){
    delete state.designer.metas[state.designer.mode];
    openDesigner(state.designer.mode);
  };

  applyFeatureVisibility();
  load();
})();
