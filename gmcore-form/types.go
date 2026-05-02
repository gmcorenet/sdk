package gmcoreform

type InputType string

const (
	TypeText       InputType = "text"
	TypeEmail      InputType = "email"
	TypePassword   InputType = "password"
	TypeNumber     InputType = "number"
	TypeInteger    InputType = "integer"
	TypeDecimal   InputType = "decimal"
	TypeFloat     InputType = "float"
	TypeBoolean   InputType = "boolean"
	TypeCheckbox  InputType = "checkbox"
	TypeRadio     InputType = "radio"
	TypeSelect       InputType = "select"
	TypeSelectAjax   InputType = "select_ajax"
	TypeTextarea     InputType = "textarea"
	TypeHidden    InputType = "hidden"
	TypeTel       InputType = "tel"
	TypeUrl       InputType = "url"
	TypeSearch    InputType = "search"
	TypeRange     InputType = "range"
	TypeDate      InputType = "date"
	TypeTime      InputType = "time"
	TypeDateTime  InputType = "datetime-local"
	TypeMonth     InputType = "month"
	TypeWeek      InputType = "week"
	TypeColor     InputType = "color"
	TypeFile      InputType = "file"
	TypeImage     InputType = "image"
	TypeSubmit    InputType = "submit"
	TypeButton    InputType = "button"
	TypeReset    InputType = "reset"
	TypeReadOnly  InputType = "readonly"
	TypeJson      InputType = "json"
	TypeUuid      InputType = "uuid"
	TypeIpAddress InputType = "ip_address"
	TypeMacAddress InputType = "mac_address"
	TypeSlug      InputType = "slug"
	TypeMoney     InputType = "money"
	TypePercent   InputType = "percent"
	TypeBirthDate InputType = "birthdate"
	TypePhone     InputType = "phone"
	TypeCaptcha   InputType = "captcha"
	TypeRecaptcha InputType = "recaptcha"
	TypeRating    InputType = "rating"
	TypeTags      InputType = "tags"
	TypeAutocomplete InputType = "autocomplete"
	TypeCkEditor  InputType = "ckeditor"
	TypeTinyMce   InputType = "tinymce"
	TypeMarkdown  InputType = "markdown"
	TypeHtml      InputType = "html"
)

type WidgetType string

const (
	WidgetText            WidgetType = "text"
	WidgetTextarea        WidgetType = "textarea"
	WidgetSelect          WidgetType = "select"
	WidgetSelectAjax      WidgetType = "select_ajax"
	WidgetSelectMultiple  WidgetType = "select_multiple"
	WidgetCheckbox        WidgetType = "checkbox"
	WidgetCheckboxGroup   WidgetType = "checkbox_group"
	WidgetRadioGroup      WidgetType = "radio_group"
	WidgetRadioButton     WidgetType = "radio_button"
	WidgetSwitch         WidgetType = "switch"
	WidgetToggle         WidgetType = "toggle"
	WidgetDatePicker     WidgetType = "date_picker"
	WidgetDateTimePicker WidgetType = "datetime_picker"
	WidgetTimePicker     WidgetType = "time_picker"
	WidgetMonthPicker    WidgetType = "month_picker"
	WidgetWeekPicker     WidgetType = "week_picker"
	WidgetDaterangePicker WidgetType = "daterange_picker"
	WidgetColorPicker    WidgetType = "color_picker"
	WidgetRangeSlider    WidgetType = "range_slider"
	WidgetFileUpload     WidgetType = "file_upload"
	WidgetImageUpload    WidgetType = "image_upload"
	WidgetDropzone       WidgetType = "dropzone"
	WidgetAvatarUpload   WidgetType = "avatar_upload"
	WidgetEditor         WidgetType = "editor"
	WidgetMarkdown      WidgetType = "markdown"
	WidgetHtmlEditor    WidgetType = "html_editor"
	WidgetSummernote     WidgetType = "summernote"
	WidgetTrumbowyg     WidgetType = "trumbowyg"
	WidgetTagInput      WidgetType = "tag_input"
	WidgetTokenfield    WidgetType = "tokenfield"
	WidgetSelect2       WidgetType = "select2"
	WidgetSelectize      WidgetType = "selectize"
	WidgetTypeahead     WidgetType = "typeahead"
	WidgetAutocomplete  WidgetType = "autocomplete"
	WidgetRating        WidgetType = "rating"
	WidgetCaptcha       WidgetType = "captcha"
	WidgetRecaptcha     WidgetType = "recaptcha"
	WidgetPhoneInput    WidgetType = "phone_input"
	WidgetCurrencyInput  WidgetType = "currency_input"
	WidgetPercentInput   WidgetType = "percent_input"
	WidgetHidden        WidgetType = "hidden"
	WidgetReadOnly      WidgetType = "readonly"
	WidgetStatic        WidgetType = "static"
	WidgetButton        WidgetType = "button"
	WidgetSubmit        WidgetType = "submit"
	WidgetReset         WidgetType = "reset"
	WidgetForm          WidgetType = "form"
	WidgetFormRow       WidgetType = "form_row"
	WidgetFormColumn    WidgetType = "form_column"
	WidgetCollection    WidgetType = "collection"
	WidgetRepeatable    WidgetType = "repeatable"
	WidgetEmbed         WidgetType = "embed"
	WidgetTable        WidgetType = "table"
	WidgetList         WidgetType = "list"
	WidgetAccordion    WidgetType = "accordion"
	WidgetTabs         WidgetType = "tabs"
	WidgetModal        WidgetType = "modal"
	WidgetAlert        WidgetType = "alert"
	WidgetBadge        WidgetType = "badge"
	WidgetLabel        WidgetType = "label"
	WidgetHelpBlock    WidgetType = "help_block"
	WidgetErrorBlock   WidgetType = "error_block"
	WidgetPlaceholder  WidgetType = "placeholder"
	WidgetImage        WidgetType = "image"
	WidgetAvatar       WidgetType = "avatar"
	WidgetFileIcon     WidgetType = "file_icon"
	WidgetDownloadLink WidgetType = "download_link"
	WidgetUrlLink      WidgetType = "url_link"
	WidgetBreadcrumb   WidgetType = "breadcrumb"
	WidgetPagination   WidgetType = "pagination"
	WidgetProgress      WidgetType = "progress"
	WidgetSpinner       WidgetType = "spinner"
	WidgetSkeleton      WidgetType = "skeleton"
)

var DefaultWidgetForType = map[InputType]WidgetType{
	TypeText:        WidgetText,
	TypeEmail:       WidgetText,
	TypePassword:    WidgetText,
	TypeNumber:      WidgetText,
	TypeInteger:     WidgetText,
	TypeDecimal:     WidgetText,
	TypeFloat:       WidgetText,
	TypeTel:         WidgetText,
	TypeUrl:         WidgetText,
	TypeSearch:      WidgetText,
	TypeTextarea:    WidgetTextarea,
	TypeHidden:      WidgetHidden,
	TypeCheckbox:    WidgetCheckbox,
	TypeRadio:       WidgetRadioGroup,
	TypeSelect:      WidgetSelect,
	TypeDate:        WidgetDatePicker,
	TypeTime:        WidgetTimePicker,
	TypeDateTime:    WidgetDateTimePicker,
	TypeMonth:       WidgetMonthPicker,
	TypeWeek:        WidgetWeekPicker,
	TypeColor:       WidgetColorPicker,
	TypeRange:       WidgetRangeSlider,
	TypeFile:        WidgetFileUpload,
	TypeImage:       WidgetImageUpload,
	TypeSubmit:      WidgetSubmit,
	TypeButton:      WidgetButton,
	TypeReset:       WidgetReset,
	TypeReadOnly:    WidgetReadOnly,
	TypeBoolean:     WidgetSwitch,
	TypeJson:        WidgetTextarea,
	TypeUuid:        WidgetText,
	TypeIpAddress:   WidgetText,
	TypeMacAddress:  WidgetText,
	TypeSlug:        WidgetText,
	TypeMoney:       WidgetCurrencyInput,
	TypePercent:     WidgetPercentInput,
	TypeBirthDate:   WidgetDatePicker,
	TypePhone:       WidgetPhoneInput,
	TypeCaptcha:     WidgetCaptcha,
	TypeRecaptcha:   WidgetRecaptcha,
	TypeRating:      WidgetRating,
	TypeTags:        WidgetTagInput,
	TypeAutocomplete: WidgetAutocomplete,
	TypeCkEditor:    WidgetEditor,
	TypeTinyMce:     WidgetEditor,
	TypeMarkdown:    WidgetMarkdown,
	TypeHtml:        WidgetHtmlEditor,
}

var InputTypesWithoutWidget = map[InputType]bool{
	TypeSubmit: true,
	TypeButton: true,
	TypeReset: true,
	TypeReadOnly: true,
}

var FileTypes = map[InputType]bool{
	TypeFile: true,
	TypeImage: true,
}

var MultipleSupportedTypes = map[InputType]bool{
	TypeSelect: true,
	TypeFile: true,
	TypeImage: true,
	TypeCheckbox: true,
	TypeTags: true,
}

func GetDefaultWidget(inputType InputType) WidgetType {
	if widget, ok := DefaultWidgetForType[inputType]; ok {
		return widget
	}
	return WidgetText
}

func SupportsMultiple(inputType InputType) bool {
	return MultipleSupportedTypes[inputType]
}

func IsFileType(inputType InputType) bool {
	return FileTypes[inputType]
}

func RequiresAjaxOptions(inputType InputType) bool {
	return inputType == TypeSelectAjax || inputType == TypeAutocomplete
}

func IsEditable(inputType InputType) bool {
	return !InputTypesWithoutWidget[inputType] && inputType != TypeReadOnly
}
