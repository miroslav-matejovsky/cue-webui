package webui

import (
	"net/url"
	"sort"
	"strings"
)

func applyStoredValues(formData FormData, values map[string]string) FormData {
	cloned := formData
	cloned.Sections = cloneSectionsWithValues(formData.Sections, values)
	return cloned
}

func cloneSectionsWithValues(sections []Section, values map[string]string) []Section {
	if len(sections) == 0 {
		return nil
	}

	cloned := make([]Section, 0, len(sections))
	for _, section := range sections {
		sectionCopy := section
		if len(section.Fields) > 0 {
			sectionCopy.Fields = make([]Field, len(section.Fields))
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

func mergeSubmittedValues(formData FormData, existing map[string]string, submitted url.Values) map[string]string {
	merged := cloneValueMap(existing)
	for key, values := range submitted {
		merged[key] = strings.Join(values, ", ")
	}

	visitFields(formData.Sections, func(field Field) {
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

func resultDataFromValues(title string, values map[string]string) ResultData {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	result := ResultData{Title: title, Values: make([]KeyValue, 0, len(keys))}
	for _, key := range keys {
		result.Values = append(result.Values, KeyValue{Key: key, Value: values[key]})
	}
	return result
}

func visitFields(sections []Section, visit func(Field)) {
	for _, section := range sections {
		for _, field := range section.Fields {
			visit(field)
		}
		visitFields(section.Sections, visit)
	}
}

func fieldIsDisabled(field Field) bool {
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
