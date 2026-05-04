package gmcore_form

type Option struct {
	Value       string `json:"value"`
	Label       string `json:"label"`
	LabelKey   string `json:"label_key,omitempty"`
	Disabled   bool   `json:"disabled,omitempty"`
	Attr       map[string]string `json:"attr,omitempty"`
}

type Field struct {
	Name            string       `json:"name"`
	Label          string       `json:"label"`
	LabelKey       string       `json:"label_key,omitempty"`
	Type           InputType    `json:"type"`
	Widget         WidgetType   `json:"widget,omitempty"`
	Required       bool         `json:"required"`
	ReadOnly       bool         `json:"read_only,omitempty"`
	Disabled       bool         `json:"disabled,omitempty"`
	WriteOnly      bool         `json:"write_only,omitempty"`
	AutoManaged    bool         `json:"auto_managed,omitempty"`
	Multiple       bool         `json:"multiple,omitempty"`
	Placeholder    string       `json:"placeholder,omitempty"`
	Help           string       `json:"help,omitempty"`
	HelpKey        string       `json:"help_key,omitempty"`
	ColSpan        int          `json:"col_span,omitempty"`
	RowSpan        int          `json:"row_span,omitempty"`
	Class          string       `json:"class,omitempty"`
	Style          string       `json:"style,omitempty"`
	Attr           map[string]string `json:"attr,omitempty"`
	DataAttr       map[string]string `json:"data_attr,omitempty"`

	Validation      []string     `json:"validation,omitempty"`
	ErrorMessage    string       `json:"error_message,omitempty"`
	ErrorMessages   map[string]string `json:"error_messages,omitempty"`

	Options         []Option     `json:"options,omitempty"`
	OptionUrl       string       `json:"option_url,omitempty"`
	OptionRemote    *RemoteOptions `json:"option_remote,omitempty"`

	ValueField      string       `json:"value_field,omitempty"`
	DisplayField    string       `json:"display_field,omitempty"`
	GroupField     string       `json:"group_field,omitempty"`

	Min             interface{}  `json:"min,omitempty"`
	Max             interface{}  `json:"max,omitempty"`
	Step            interface{}  `json:"step,omitempty"`
	MinLength       int          `json:"min_length,omitempty"`
	MaxLength       int          `json:"max_length,omitempty"`
	Pattern         string       `json:"pattern,omitempty"`
	Accept          string       `json:"accept,omitempty"`
	MaxSize         int64        `json:"max_size,omitempty"`
	MaxSizeFormatted string      `json:"max_size_formatted,omitempty"`
	AllowedTypes    []string     `json:"allowed_types,omitempty"`
	AllowedExtensions []string   `json:"allowed_extensions,omitempty"`
	MimeTypes       []string     `json:"mime_types,omitempty"`
	UploadUrl       string       `json:"upload_url,omitempty"`
	UploadFolder    string       `json:"upload_folder,omitempty"`
	ThumbnailSize   int          `json:"thumbnail_size,omitempty"`

	Relation        string       `json:"relation,omitempty"`
	Sortable        bool         `json:"sortable,omitempty"`
	Filterable      bool         `json:"filterable,omitempty"`
	Searchable      bool         `json:"searchable,omitempty"`
	DefaultValue    interface{}  `json:"default_value,omitempty"`
	HelpDisplay     string       `json:"help_display,omitempty"`

	DefaultOptions   []Option     `json:"default_options,omitempty"`
	Translated      bool         `json:"translated,omitempty"`
	Compound        bool         `json:"compound,omitempty"`
	Children        []Field      `json:"children,omitempty"`

	Hidden          bool         `json:"hidden,omitempty"`
	Invisible       bool         `json:"invisible,omitempty"`
	Delegated       bool         `json:"delegated,omitempty"`
}

type RemoteOptions struct {
	Url             string       `json:"url"`
	Method          string       `json:"method,omitempty"`
	QueryParam      string       `json:"query_param,omitempty"`
	Delay           int          `json:"delay,omitempty"`
	MinInputLength  int          `json:"min_input_length,omitempty"`
	Cache          bool         `json:"cache,omitempty"`
}

type Button struct {
	Name      string            `json:"name"`
	Label     string            `json:"label,omitempty"`
	LabelKey  string            `json:"label_key,omitempty"`
	Style     string            `json:"style,omitempty"`
	Class     string            `json:"class,omitempty"`
	Attr      map[string]string `json:"attr,omitempty"`
	Type      string            `json:"type,omitempty"`
	Icon      string            `json:"icon,omitempty"`
	IconClass string            `json:"icon_class,omitempty"`
	Confirmation string          `json:"confirmation,omitempty"`
	Route      string           `json:"route,omitempty"`
	Url       string            `json:"url,omitempty"`
}

type Definition struct {
	Name     string   `json:"name"`
	Title    string   `json:"title,omitempty"`
	TitleKey string   `json:"title_key,omitempty"`
	Class    string   `json:"class,omitempty"`
	Action   string   `json:"action,omitempty"`
	Method   string   `json:"method,omitempty"`
	Enctype  string   `json:"enctype,omitempty"`
	Fields   []Field  `json:"fields"`
	Buttons  []Button `json:"buttons,omitempty"`
}

func (d Definition) NormalizeButtons(mode string) []Button {
	if len(d.Buttons) > 0 {
		return d.Buttons
	}
	if mode == "show" {
		return []Button{
			{Name: "back", LabelKey: "crud.action.back", Style: "outline-secondary"},
		}
	}
	return []Button{
		{Name: "save", LabelKey: "crud.action.save", Style: "primary", Type: "submit"},
		{Name: "save_exit", LabelKey: "crud.action.save_and_exit", Style: "outline-primary", Type: "submit"},
		{Name: "save_new", LabelKey: "crud.action.save_and_new", Style: "outline-primary", Type: "submit"},
		{Name: "cancel", LabelKey: "crud.action.cancel", Style: "outline-secondary", Type: "button"},
	}
}

func (f *Field) NormalizeWidget() {
	if f.Widget != "" {
		return
	}
	if InputTypesWithoutWidget[f.Type] {
		return
	}
	f.Widget = GetDefaultWidget(f.Type)
}

func (f *Field) SupportsMultiple() bool {
	return SupportsMultiple(f.Type)
}

func (f *Field) IsFileType() bool {
	return IsFileType(f.Type)
}
