package webui

import (
	"net/url"
	"strings"

	"github.com/miroslav-matejovsky/cue-webui/internal/webui/webform"
)

func applyStoredValues(formData webform.FormData, values map[string]string) webform.FormData {
	cloned := formData
	cloned.Sections = cloneSectionsWithValues(formData.Sections, values)
	return cloned
}

func cloneSectionsWithValues(sections []webform.Section, values map[string]string) []webform.Section {
	if len(sections) == 0 {
		return nil
	}

	cloned := make([]webform.Section, 0, len(sections))
	for _, section := range sections {
		sectionCopy := section
		if len(section.Fields) > 0 {
			sectionCopy.Fields = make([]webform.Field, len(section.Fields))
			for index, field := range section.Fields {
				fieldCopy := field
				if storedValue, ok := values[field.Path]; ok {
					fieldCopy.Default = storedValue
				}
				if len(field.Options) > 0 {
					fieldCopy.Options = append([]string(nil), field.Options...)
				}
				sectionCopy.Fields[index] = fieldCopy
			}
		}
		sectionCopy.Sections = cloneSectionsWithValues(section.Sections, values)
		cloned = append(cloned, sectionCopy)
	}

	return cloned
}

func mergeSubmittedValues(formData webform.FormData, existing map[string]string, submitted url.Values) map[string]string {
	merged := cloneValueMap(existing)
	for key, values := range submitted {
		merged[key] = strings.Join(values, ", ")
	}

	visitFields(formData.Sections, func(field webform.Field) {
		values, ok := submitted[field.Path]
		if ok {
			merged[field.Path] = strings.Join(values, ", ")
			return
		}

		if field.Hidden || fieldIsDisabled(field) {
			return
		}

		if field.Widget == "checkbox" {
			merged[field.Path] = "false"
			return
		}

		delete(merged, field.Path)
	})
	return merged
}

func visitFields(sections []webform.Section, visit func(webform.Field)) {
	for _, section := range sections {
		for _, field := range section.Fields {
			visit(field)
		}
		visitFields(section.Sections, visit)
	}
}

func fieldIsDisabled(field webform.Field) bool {
	if !field.Readonly {
		return false
	}

	switch field.Widget {
	case "checkbox", "radio", "select":
		return true
	default:
		return false
	}
}

func cloneValueMap(values map[string]string) map[string]string {
	if len(values) == 0 {
		return map[string]string{}
	}
	cloned := make(map[string]string, len(values))
	for key, value := range values {
		cloned[key] = value
	}
	return cloned
}

// CollectFieldTypes returns a map from field path to CUE kind string (e.g. "int", "bool")
// by walking all sections of formData recursively. It is used by the config package to
// coerce flat string values to the correct JSON types when saving.
func CollectFieldTypes(formData webform.FormData) map[string]string {
	types := map[string]string{}
	visitFields(formData.Sections, func(f webform.Field) {
		types[f.Path] = f.Type
	})
	return types
}
