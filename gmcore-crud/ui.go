package gmcore_crud

import (
	"encoding/json"
	"fmt"
	"html/template"

	gmcore_form "github.com/gmcorenet/sdk/gmcore-form"
)

type UIIndexOptions struct {
	Locale            string
	ResourceName      string
	BasePath          string
	CurrentUser       interface{}
	Features          interface{}
	Fields            []gmcore_form.Field
	Buttons           []gmcore_form.Button
	PrimaryKey        string
	Partials          map[string]string
	IconRegistry      map[string]string
	Title             string
	Subtitle          string
	DashboardURL      string
	DashboardLabel    string
	BackURL           string
	BackLabel         string
	SearchPlaceholder string
	FiltersLabel      string
	CreateLabel       string
	LoadingLabel      string
	ApplyLabel        string
	BulkLabel         string
	PrevLabel         string
	NextLabel         string
	AdvancedFilters   string
	CloseLabel        string
	ResetFiltersLabel string
	ResetViewLabel    string
	DesignerLabel     string
	DesignerTitle     string
	DesignerSubtitle  string
	PreviewLabel      string
	SaveLayoutLabel   string
	CancelLabel       string
	CreateModeLabel   string
	EditModeLabel     string
	CloneModeLabel    string
	WidthLabel        string
	HeightLabel       string
	WidgetLabel       string
	ColorLabel        string
	HiddenLabel       string
	ResetLabel        string
	SaveModeLabel     string
	CSRFToken         string
}

type UIIndexFragments struct {
	ToolbarHTML         template.HTML
	TableHTML           template.HTML
	EmptyHTML           template.HTML
	FiltersModalHTML    template.HTML
	EntryModalHTML      template.HTML
	DesignerModalHTML   template.HTML
	ClientBootstrapHTML template.HTML
}

func RenderClientBootstrap(opts UIIndexOptions) template.HTML {
	currentUserJSON, _ := json.Marshal(opts.CurrentUser)
	fieldsJSON, _ := json.Marshal(opts.Fields)
	buttonsJSON, _ := json.Marshal(opts.Buttons)
	primaryKeyJSON, _ := json.Marshal(opts.PrimaryKey)
	partialsJSON, _ := json.Marshal(opts.Partials)
	iconRegistryJSON, _ := json.Marshal(opts.IconRegistry)
	featuresJSON, _ := json.Marshal(opts.Features)
	labelsJSON, _ := json.Marshal(map[string]string{
		"loading":             opts.LoadingLabel,
		"empty":               "No records available yet.",
		"operator":            "Operator",
		"value":               "Value",
		"confirm_delete":      "Delete record?",
		"confirm_bulk_delete": "Delete selected records?",
		"create":              opts.CreateLabel,
		"edit":                "Edit",
		"clone":               "Clone",
		"view":                "View",
		"save":                "Save",
		"bulk":                opts.BulkLabel,
		"close":               opts.CloseLabel,
		"designer":            opts.DesignerLabel,
		"mode_create":         opts.CreateModeLabel,
		"mode_edit":           opts.EditModeLabel,
		"mode_clone":          opts.CloneModeLabel,
		"width":               opts.WidthLabel,
		"height":              opts.HeightLabel,
		"widget":              opts.WidgetLabel,
		"color":               opts.ColorLabel,
		"hidden":              opts.HiddenLabel,
		"reset":               opts.ResetLabel,
		"save_mode":           opts.SaveModeLabel,
		"reset_filters":       "Reset filters",
		"reset_view":          "Reset view",
		"load_more":           "Load more",
		"load_all":            "Load all matching",
		"loaded":              "loaded",
		"operator_eq":         "equals",
		"operator_ne":         "is not",
		"operator_like":       "contains",
		"operator_not_like":   "does not contain",
		"operator_gt":         "is greater than",
		"operator_gte":        "is greater or equal to",
		"operator_lt":         "is less than",
		"operator_lte":        "is less or equal to",
		"operator_in":         "is in",
		"operator_null":       "is null",
		"operator_not_null":   "is not null",
		"summary_entries":     "Entries",
		"summary_of":          "of",
		"summary_page":        "Page",
		"summary_total":       "total",
		"resize_column":       "Resize column",
		"search_label":        opts.SearchPlaceholder,
	})
	return template.HTML(fmt.Sprintf(`
<script>
window.__crudConfig = {
  resource: %q,
  basePath: %q,
  locale: %q,
  currentUser: %s,
  fields: %s,
  buttons: %s,
  primaryKey: %s,
  partials: %s,
  iconRegistry: %s,
  features: %s,
  labels: %s
};
window.__crudRoot = document.currentScript && document.currentScript.parentElement ? document.currentScript.parentElement : null;
</script>
<script>
%s
</script>`,
		opts.ResourceName,
		opts.BasePath,
		opts.Locale,
		currentUserJSON,
		fieldsJSON,
		buttonsJSON,
		primaryKeyJSON,
		partialsJSON,
		iconRegistryJSON,
		featuresJSON,
		labelsJSON,
		crudClientScript(),
	))
}

func RenderIndexFragments(opts UIIndexOptions) UIIndexFragments {
	empty := template.HTML(`
<template id="crud-empty-template">
  <div class="crud-empty-state" id="crud-empty-state">
    <div class="crud-empty-icon">
      <img class="crud-icon crud-icon-empty" src="/assets/crud/icons/empty.svg" alt="">
    </div>
    <div class="crud-empty-title">No records available yet.</div>
    <p class="crud-empty-copy">No records available yet.</p>
  </div>
</template>`)
	toolbar := template.HTML(fmt.Sprintf(`
<div class="crud-toolbar">
  <div class="crud-toolbar-main">
    <label class="crud-search-shell crud-search-shell-themed" id="crud-search-shell" for="crud-search">
      <img class="crud-icon crud-search-icon" src="/assets/crud/icons/search.svg" alt="">
      <input id="crud-search" class="crud-input crud-search-input" placeholder="%s">
    </label>
  </div>
  <div class="crud-toolbar-actions">
    <button class="crud-btn crud-btn-outline-secondary crud-control-btn d-none" data-crud-reset-filters type="button">
      <span>%s</span>
    </button>
    <button class="crud-btn crud-btn-outline-secondary crud-control-btn d-none" id="crud-open-filters" type="button" data-crud-modal-open="#crud-filters-modal">
      <img class="crud-icon" src="/assets/crud/icons/filter.svg" alt="">
      <span>%s</span>
    </button>
    <button class="crud-btn crud-btn-outline-secondary crud-control-btn d-none" id="crud-reset-view" type="button">
      <span>%s</span>
    </button>
    <button class="crud-btn crud-btn-outline-secondary crud-control-btn d-none" id="crud-open-designer" type="button">
      <img class="crud-icon" src="/assets/crud/icons/layout.svg" alt="">
      <span>%s</span>
    </button>
    <button class="crud-btn crud-btn-primary crud-control-btn d-none" id="crud-open-create" type="button">
      <img class="crud-icon" src="/assets/crud/icons/plus.svg" alt="">
      <span>%s</span>
    </button>
  </div>
</div>`,
		template.HTMLEscapeString(opts.SearchPlaceholder),
		template.HTMLEscapeString(opts.ResetFiltersLabel),
		template.HTMLEscapeString(opts.FiltersLabel),
		template.HTMLEscapeString(opts.ResetViewLabel),
		template.HTMLEscapeString(opts.DesignerLabel),
		template.HTMLEscapeString(opts.CreateLabel),
	))
	table := template.HTML(fmt.Sprintf(`
<div class="crud-table-card">
  <div class="crud-table-controls crud-table-controls-top">
    <div class="crud-table-controls-left">
      <select class="crud-select crud-bulk-select crud-control-select d-none" data-crud-bulk-action></select>
      <button class="crud-btn crud-btn-outline-danger crud-control-btn d-none" data-crud-bulk-apply type="button">
        <img class="crud-icon" src="/assets/crud/icons/stack.svg" alt="">
        <span>%s</span>
      </button>
      <span class="small text-secondary crud-active-filters d-none" data-crud-active-filters></span>
    </div>
    <div class="crud-table-controls-right">
      <span class="small text-secondary" data-crud-summary></span>
      <span class="small text-secondary" data-crud-total></span>
      <div class="crud-per-page-shell">
        <select class="crud-select crud-bulk-select crud-control-select d-none" data-crud-per-page>
          <option>10</option>
          <option>25</option>
          <option>50</option>
        </select>
      </div>
      <div class="crud-pagination d-none" data-crud-pagination-group>
        <button class="crud-btn crud-btn-outline-secondary crud-page-btn" data-crud-first type="button">&laquo;</button>
        <button class="crud-btn crud-btn-outline-secondary crud-page-btn" data-crud-prev type="button">%s</button>
        <span class="crud-page-numbers" data-crud-page-numbers></span>
        <button class="crud-btn crud-btn-outline-secondary crud-page-btn" data-crud-next type="button">%s</button>
        <button class="crud-btn crud-btn-outline-secondary crud-page-btn" data-crud-last type="button">&raquo;</button>
      </div>
      <div class="crud-page-jump d-none" data-crud-page-jump>
        <input class="crud-input crud-input-sm crud-page-jump-input" type="number" min="1" value="1" data-crud-page-input>
        <button class="crud-btn crud-btn-outline-secondary crud-btn-sm" type="button" data-crud-page-apply>&rarr;</button>
      </div>
    </div>
  </div>
  <div class="crud-table-surface">
    <div class="crud-table-responsive">
      <table class="crud-table">
        <colgroup id="crud-colgroup"></colgroup>
        <thead>
          <tr id="crud-head"></tr>
        </thead>
        <tbody id="crud-body">
          <tr>
            <td colspan="99" class="p-4 text-secondary">%s</td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
  <div class="crud-footer crud-table-controls crud-table-controls-bottom">
    <div class="crud-table-controls-left">
      <select class="crud-select crud-bulk-select crud-control-select d-none" data-crud-bulk-action></select>
      <button class="crud-btn crud-btn-outline-danger crud-control-btn d-none" data-crud-bulk-apply type="button">
        <img class="crud-icon" src="/assets/crud/icons/stack.svg" alt="">
        <span>%s</span>
      </button>
      <span class="small text-secondary crud-active-filters d-none" data-crud-active-filters></span>
    </div>
    <div class="crud-table-controls-right">
      <span class="small text-secondary" data-crud-summary></span>
      <span class="small text-secondary" data-crud-total></span>
      <div class="crud-per-page-shell">
        <select class="crud-select crud-bulk-select crud-control-select d-none" data-crud-per-page>
          <option>10</option>
          <option>25</option>
          <option>50</option>
        </select>
      </div>
      <div class="crud-pagination d-none" data-crud-pagination-group>
        <button class="crud-btn crud-btn-outline-secondary crud-page-btn" data-crud-first type="button">&laquo;</button>
        <button class="crud-btn crud-btn-outline-secondary crud-page-btn" data-crud-prev type="button">%s</button>
        <span class="crud-page-numbers" data-crud-page-numbers></span>
        <button class="crud-btn crud-btn-outline-secondary crud-page-btn" data-crud-next type="button">%s</button>
        <button class="crud-btn crud-btn-outline-secondary crud-page-btn" data-crud-last type="button">&raquo;</button>
      </div>
      <div class="crud-page-jump d-none" data-crud-page-jump>
        <input class="crud-input crud-input-sm crud-page-jump-input" type="number" min="1" value="1" data-crud-page-input>
        <button class="crud-btn crud-btn-outline-secondary crud-btn-sm" type="button" data-crud-page-apply>&rarr;</button>
      </div>
    </div>
  </div>
</div>`,
		template.HTMLEscapeString(opts.ApplyLabel),
		template.HTMLEscapeString(opts.PrevLabel),
		template.HTMLEscapeString(opts.NextLabel),
		template.HTMLEscapeString(opts.LoadingLabel),
		template.HTMLEscapeString(opts.ApplyLabel),
		template.HTMLEscapeString(opts.PrevLabel),
		template.HTMLEscapeString(opts.NextLabel),
	))
	filtersModal := template.HTML(fmt.Sprintf(`
<div class="crud-modal" id="crud-filters-modal" tabindex="-1">
  <div class="crud-modal-dialog">
    <div class="crud-modal-content">
      <div class="crud-modal-header">
        <div class="crud-modal-heading">
          <img class="crud-icon" src="/assets/crud/icons/filter.svg" alt="">
          <h5 class="crud-modal-title">%s</h5>
        </div>
        <button type="button" class="crud-btn-close" data-crud-modal-close></button>
      </div>
      <div class="crud-modal-body">
        <table class="crud-filter-table">
          <tbody id="crud-filters-list"></tbody>
        </table>
      </div>
      <div class="crud-modal-footer">
        <button class="crud-btn crud-btn-outline-secondary" type="button" data-crud-modal-close>%s</button>
        <button class="crud-btn crud-btn-primary" type="button" id="crud-apply-filters">
          <img class="crud-icon" src="/assets/crud/icons/filter.svg" alt="">
          <span>%s</span>
        </button>
      </div>
    </div>
  </div>
</div>`,
		template.HTMLEscapeString(opts.AdvancedFilters),
		template.HTMLEscapeString(opts.CloseLabel),
		template.HTMLEscapeString(opts.ApplyLabel),
	))
	entryModal := template.HTML(fmt.Sprintf(`
<div class="crud-modal" id="crud-entry-modal" tabindex="-1">
  <div class="crud-modal-dialog crud-modal-dialog-xl">
    <div class="crud-modal-content">
      <div class="crud-modal-header">
        <div class="crud-modal-heading">
          <img class="crud-icon" src="/assets/crud/icons/form.svg" alt="">
          <h5 class="crud-modal-title" id="crud-entry-title">%s</h5>
        </div>
        <button type="button" class="crud-btn-close" data-crud-modal-close></button>
      </div>
      <div class="crud-modal-body">
        <div id="crud-entry-body"></div>
      </div>
      <div class="crud-modal-footer crud-entry-footer">
        <div class="small text-secondary" id="crud-entry-note"></div>
        <div class="crud-entry-actions" id="crud-entry-actions"></div>
      </div>
    </div>
  </div>
</div>`,
		template.HTMLEscapeString(opts.Title),
	))
	designerModal := template.HTML(fmt.Sprintf(`
<div class="crud-modal" id="crud-designer-modal" tabindex="-1">
  <div class="crud-modal-dialog crud-modal-dialog-designer">
    <div class="crud-modal-content">
      <div class="crud-modal-header">
        <div class="crud-modal-heading">
          <img class="crud-icon" src="/assets/crud/icons/layout.svg" alt="">
          <div>
            <h5 class="crud-modal-title">%s</h5>
            <div class="crud-designer-header-copy">%s</div>
          </div>
        </div>
        <button type="button" class="crud-btn-close" data-crud-modal-close></button>
      </div>
      <div class="crud-modal-body crud-designer-body">
        <div class="crud-designer-workspace">
          <div class="crud-designer-canvas" id="crud-designer-canvas"></div>
        </div>
      </div>
      <div class="crud-modal-footer crud-designer-footer">
        <div class="crud-designer-footer-panel">
          <div class="crud-designer-footer-main">
            <div class="crud-designer-toolbar" id="crud-designer-toolbar"></div>
          </div>
          <div class="crud-designer-footer-side">
            <div class="crud-designer-controlbar" id="crud-designer-controlbar"></div>
          </div>
          <div class="crud-designer-footer-bottom">
            <div class="crud-designer-hidden-tray" id="crud-designer-hidden-tray"></div>
            <div class="crud-designer-footer-actions">
              <button class="crud-btn crud-btn-outline-secondary" type="button" id="crud-designer-reset">%s</button>
              <button class="crud-btn crud-btn-outline-secondary" type="button" data-crud-modal-close>%s</button>
              <button class="crud-btn crud-btn-primary" type="button" id="crud-designer-save">%s</button>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</div>
<div class="crud-modal" id="crud-bulk-confirm-modal" tabindex="-1">
  <div class="crud-modal-dialog crud-modal-dialog-sm">
    <div class="crud-modal-content">
      <div class="crud-modal-header">
        <div class="crud-modal-heading">
          <img class="crud-icon" src="/assets/crud/icons/stack.svg" alt="">
          <h5 class="crud-modal-title">Confirm bulk action</h5>
        </div>
        <button type="button" class="crud-btn-close" data-crud-modal-close></button>
      </div>
      <div class="crud-modal-body">
        <div class="crud-confirm-panel">
          <div class="crud-confirm-copy">Apply this action to the selected records?</div>
          <div class="crud-confirm-meta" id="crud-bulk-confirm-meta"></div>
        </div>
      </div>
      <div class="crud-modal-footer">
        <button class="crud-btn crud-btn-outline-secondary" type="button" data-crud-modal-close>%s</button>
        <button class="crud-btn crud-btn-primary" type="button" id="crud-bulk-confirm-submit">
          <img class="crud-icon" src="/assets/crud/icons/stack.svg" alt="">
          <span>%s</span>
        </button>
      </div>
    </div>
  </div>
</div>
`,
		template.HTMLEscapeString(opts.DesignerTitle),
		template.HTMLEscapeString(opts.DesignerSubtitle),
		template.HTMLEscapeString(opts.ResetLabel),
		template.HTMLEscapeString(opts.CancelLabel),
		template.HTMLEscapeString(opts.SaveLayoutLabel),
		template.HTMLEscapeString(opts.CancelLabel),
		template.HTMLEscapeString(opts.ApplyLabel),
	))
	return UIIndexFragments{
		ToolbarHTML:         toolbar,
		TableHTML:           table,
		EmptyHTML:           empty,
		FiltersModalHTML:    filtersModal,
		EntryModalHTML:      entryModal,
		DesignerModalHTML:   designerModal,
		ClientBootstrapHTML: RenderClientBootstrap(opts),
	}
}

func RenderIndexPage(opts UIIndexOptions) string {
	fragments := RenderIndexFragments(opts)
	return string(fragments.ToolbarHTML) +
		string(fragments.TableHTML) +
		string(fragments.FiltersModalHTML) +
		string(fragments.EntryModalHTML) +
		string(fragments.DesignerModalHTML) +
		string(fragments.ClientBootstrapHTML)
}
