package templater

import (
	"fmt"
	"strings"
	"sync"
	"text/template"

	gun "github.com/yandex/pandora/components/guns/http_scenario"
	"github.com/yandex/pandora/components/providers/scenario/templater"
)

func NewTextTemplater() Templater {
	return &TextTemplater{}
}

type TextTemplater struct {
	templatesCache sync.Map
}

func (t *TextTemplater) Apply(parts *gun.RequestParts, vs map[string]any, scenarioName, stepName string) error {
	const op = "scenario/TextTemplater.Apply"
	tmpl, err := t.getTemplate(parts.URL, scenarioName, stepName, "url")
	if err != nil {
		return fmt.Errorf("%s, template.New, %w", op, err)
	}

	strBuilder := &strings.Builder{}
	err = tmpl.Execute(strBuilder, vs)
	if err != nil {
		return fmt.Errorf("%s, template.Execute url, %w", op, err)
	}
	parts.URL = strBuilder.String()
	strBuilder.Reset()

	for k, v := range parts.Headers {
		tmpl, err = t.getTemplate(v, scenarioName, stepName, k)
		if err != nil {
			return fmt.Errorf("%s, template.Execute Header %s, %w", op, k, err)
		}
		err = tmpl.Execute(strBuilder, vs)
		if err != nil {
			return fmt.Errorf("%s, template.Execute Header %s, %w", op, k, err)
		}
		parts.Headers[k] = strBuilder.String()
		strBuilder.Reset()
	}
	if parts.Body != nil {
		tmpl, err = t.getTemplate(string(parts.Body), scenarioName, stepName, "body")
		if err != nil {
			return fmt.Errorf("%s, template.Execute body, %w", op, err)
		}
		err = tmpl.Execute(strBuilder, vs)
		if err != nil {
			return fmt.Errorf("%s, template.Execute body, %w", op, err)
		}
		parts.Body = []byte(strBuilder.String())
		strBuilder.Reset()
	}
	return nil
}

func (t *TextTemplater) getTemplate(tmplBody, scenarioName, stepName, key string) (*template.Template, error) {
	urlKey := fmt.Sprintf("%s_%s_%s", scenarioName, stepName, key)
	tmpl, ok := t.templatesCache.Load(urlKey)
	if !ok {
		var err error
		tmpl, err = template.New(urlKey).Funcs(templater.GetFuncs()).Parse(tmplBody)
		if err != nil {
			return nil, fmt.Errorf("scenario/TextTemplater.Apply, template.New, %w", err)
		}
		t.templatesCache.Store(urlKey, tmpl)
	}
	return tmpl.(*template.Template), nil
}
