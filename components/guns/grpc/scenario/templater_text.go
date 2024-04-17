package scenario

import (
	"fmt"
	"strings"
	"sync"
	"text/template"

	"github.com/yandex/pandora/components/providers/scenario/templater"
)

func NewTextTemplater() Templater {
	return &TextTemplater{}
}

type TextTemplater struct {
	templatesCache sync.Map
}

func (t *TextTemplater) Apply(payload []byte, metadata map[string]string, variables map[string]any, scenarioName, stepName string) ([]byte, error) {
	const op = "scenario/TextTemplater.Apply"

	strBuilder := &strings.Builder{}
	tmpl, err := t.getTemplate(string(payload), scenarioName, stepName, "payload")
	if err != nil {
		return nil, fmt.Errorf("%s, template.getTemplate payload, %w", op, err)
	}
	err = tmpl.Execute(strBuilder, variables)
	if err != nil {
		return nil, fmt.Errorf("%s, template.Execute payload, %w", op, err)
	}
	payloadStr := strBuilder.String()
	strBuilder.Reset()

	for k, v := range metadata {
		tmpl, err = t.getTemplate(v, scenarioName, stepName, k)
		if err != nil {
			return nil, fmt.Errorf("%s, template.Execute Header %s, %w", op, k, err)
		}
		err = tmpl.Execute(strBuilder, variables)
		if err != nil {
			return nil, fmt.Errorf("%s, template.Execute Header %s, %w", op, k, err)
		}
		metadata[k] = strBuilder.String()
		strBuilder.Reset()
	}
	return []byte(payloadStr), nil
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
