package gmcoreform

type Option struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

type Field struct {
	Name         string   `json:"name"`
	Label        string   `json:"label"`
	LabelKey     string   `json:"label_key"`
	Type         string   `json:"type"`
	Widget       string   `json:"widget"`
	Required     bool     `json:"required"`
	ReadOnly     bool     `json:"read_only"`
	Disabled     bool     `json:"disabled,omitempty"`
	WriteOnly    bool     `json:"write_only"`
	AutoManaged  bool     `json:"auto_managed"`
	Multiple     bool     `json:"multiple"`
	Placeholder  string   `json:"placeholder"`
	HelpKey      string   `json:"help_key"`
	Step         string   `json:"step"`
	Rows         int      `json:"rows"`
	Height       int      `json:"height"`
	ColSpan      int      `json:"col_span"`
	Color        string   `json:"color"`
	BackgroundColor string `json:"background_color,omitempty"`
	BorderColor   string   `json:"border_color,omitempty"`
	Hidden       bool     `json:"hidden"`
	Relation     string   `json:"relation,omitempty"`
	AsyncOptions bool     `json:"async_options,omitempty"`
	AsyncLimit   int      `json:"async_limit,omitempty"`
	AsyncDebounce int     `json:"async_debounce,omitempty"`
	LoadAllLimit int      `json:"load_all_limit,omitempty"`
	ValueField   string   `json:"value_field,omitempty"`
	DisplayField string   `json:"display_field,omitempty"`
	Options      []Option `json:"options"`
	Validation   []string `json:"validation,omitempty"`
}

type Button struct {
	Name     string `json:"name"`
	LabelKey string `json:"label_key"`
	Style    string `json:"style"`
}

type Definition struct {
	Name    string   `json:"name"`
	Title   string   `json:"title"`
	Fields  []Field  `json:"fields"`
	Buttons []Button `json:"buttons"`
}

func (d Definition) NormalizeButtons(mode string) []Button {
	if len(d.Buttons) > 0 {
		return d.Buttons
	}
	if mode == "show" {
		return []Button{
			{Name: "back", LabelKey: "crud.action.cancel", Style: "outline-secondary"},
		}
	}
	return []Button{
		{Name: "save", LabelKey: "crud.action.save", Style: "primary"},
		{Name: "save_exit", LabelKey: "crud.action.save_and_exit", Style: "outline-primary"},
		{Name: "save_new", LabelKey: "crud.action.save_and_new", Style: "outline-primary"},
		{Name: "cancel", LabelKey: "crud.action.cancel", Style: "outline-secondary"},
	}
}
