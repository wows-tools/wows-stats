package stats

import (
	"bytes"
	"fmt"
	"html/template"
	"io"
	"reflect"
	"regexp"

	"github.com/go-echarts/go-echarts/v2/opts"
	tpls "github.com/wows-tools/wows-stats/templates"
)

type Renderer interface {
	Render(w io.Writer) error
}

var (
	pat = regexp.MustCompile(`(__f__")|("__f__)|(__f__)`)
)

const (
	ModChart = "chart"
	ModPage  = "page"
)

type pageRender struct {
	c      interface{}
	before []func()
}

// isSet check if the field exist in the chart instance
// Shamed copy from https://stackoverflow.com/questions/44675087/golang-template-variable-isset
func isSet(name string, data interface{}) bool {
	v := reflect.ValueOf(data)

	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		return false
	}

	return v.FieldByName(name).IsValid()
}

// NewPageRender returns a render implementation for Page.
func NewPageRender(c interface{}, before ...func()) Renderer {
	return &pageRender{c: c, before: before}
}

// Render renders the page into the given io.Writer.
func (r *pageRender) Render(w io.Writer) error {
	for _, fn := range r.before {
		fn()
	}

	contents := []string{tpls.HeaderTpl, tpls.BaseTpl, tpls.PageTpl}
	tpl := MustTemplate(ModPage, contents)

	var buf bytes.Buffer
	if err := tpl.ExecuteTemplate(&buf, ModPage, r.c); err != nil {
		return err
	}

	content := pat.ReplaceAll(buf.Bytes(), []byte(""))

	_, err := w.Write(content)
	return err
}

// MustTemplate creates a new template with the given name and parsed contents.
func MustTemplate(name string, contents []string) *template.Template {
	tpl := template.Must(template.New(name).Parse(contents[0])).Funcs(template.FuncMap{
		"safeJS": func(s interface{}) template.JS {
			return template.JS(fmt.Sprint(s))
		},
		"isSet": isSet,
	})

	for _, cont := range contents[1:] {
		tpl = template.Must(tpl.Parse(cont))
	}
	return tpl
}

type Layout string

const (
	PageNoneLayout   Layout = "none"
	PageCenterLayout Layout = "center"
	PageFlexLayout   Layout = "flex"
)

// Charter represents a chart value which provides its type, assets and can be validated.
type Charter interface {
	Type() string
	GetAssets() opts.Assets
	FillDefaultValues()
	Validate()
}

// Page represents a page chart.
type Page struct {
	Renderer
	opts.Initialization
	opts.Assets

	Charts []interface{}
	Layout Layout
}

// NewPage creates a new page.
func NewPage() *Page {
	page := &Page{}
	page.Assets.InitAssets()
	page.Renderer = NewPageRender(page, page.Validate)
	page.Layout = PageCenterLayout
	return page
}

// SetLayout sets the layout of the Page.
func (page *Page) SetLayout(layout Layout) *Page {
	page.Layout = layout
	return page
}

// AddCharts adds new charts to the page.
func (page *Page) AddCharts(charts ...Charter) *Page {
	for i := 0; i < len(charts); i++ {
		assets := charts[i].GetAssets()
		for _, v := range assets.JSAssets.Values {
			page.JSAssets.Add(v)
		}

		for _, v := range assets.CSSAssets.Values {
			page.CSSAssets.Add(v)
		}
		charts[i].Validate()
		page.Charts = append(page.Charts, charts[i])
	}
	return page
}

// Validate validates the given configuration.
func (page *Page) Validate() {
	page.Initialization.Validate()
	page.Assets.Validate(page.AssetsHost)
}
