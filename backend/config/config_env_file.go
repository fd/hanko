package config

import (
	"fmt"
	"os"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

// fileSuffix is appended to a configuration environment variable name to
// reference a file whose contents should be used as the value instead, e.g.
// SECRETS_KEYS_FILE. This allows secrets to be provided through a dedicated
// file, e.g. a systemd credential, rather than through the environment or the
// configuration file.
const fileSuffix = "_FILE"

var (
	gatherEnvWords  = regexp.MustCompile("([^A-Z]+|[A-Z]+[^A-Z]+|[A-Z]+)")
	acronymEnvWords = regexp.MustCompile("([A-Z]+)([A-Z][^A-Z]+)")
)

// applyFileEnvOverrides loads configuration values from files referenced by
// <ENV>_FILE environment variables. The field is assigned directly instead of
// going through the environment so that the secret never shows up in the
// process environment (e.g. /proc/<pid>/environ).
//
// Precedence, from highest to lowest:
//  1. an explicitly set <ENV> environment variable
//  2. a file referenced by <ENV>_FILE
//  3. the configuration file
func applyFileEnvOverrides(spec any) error {
	v := reflect.ValueOf(spec)
	if v.Kind() != reflect.Pointer || v.IsNil() || v.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("config must be a non-nil pointer to a struct")
	}

	return walkEnvFields(v.Elem(), "", func(envKey string, field reflect.Value) error {
		filePath, ok := os.LookupEnv(envKey + fileSuffix)
		if !ok || filePath == "" {
			return nil
		}

		// An explicitly set environment variable always wins over the file.
		if _, ok := os.LookupEnv(envKey); ok {
			return nil
		}

		data, err := os.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("failed to read %s%s=%q: %w", envKey, fileSuffix, filePath, err)
		}

		return assignFromFile(envKey, field, data)
	})
}

// walkEnvFields visits every leaf field of a config struct, deriving the
// environment variable name the same way envconfig does.
func walkEnvFields(v reflect.Value, prefix string, fn func(string, reflect.Value) error) error {
	t := v.Type()

	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		if !field.CanSet() {
			continue
		}

		structField := t.Field(i)
		envKey := envFieldName(structField, prefix)

		for field.Kind() == reflect.Pointer {
			if field.IsNil() {
				if field.Type().Elem().Kind() != reflect.Struct {
					break
				}
				field.Set(reflect.New(field.Type().Elem()))
			}
			field = field.Elem()
		}

		switch field.Kind() {
		case reflect.Struct:
			innerPrefix := envKey
			if structField.Anonymous {
				innerPrefix = prefix
			}
			if err := walkEnvFields(field, innerPrefix, fn); err != nil {
				return err
			}
		case reflect.Map:
			// Struct-valued maps (e.g. custom third party providers) are not
			// addressable through environment variables, so skip them.
			continue
		default:
			if err := fn(envKey, field); err != nil {
				return err
			}
		}
	}

	return nil
}

// envFieldName derives the environment variable name for a struct field using
// the same rules as envconfig (envconfig tag, split_words and nested prefixes).
func envFieldName(field reflect.StructField, prefix string) string {
	name := field.Name
	if alt := field.Tag.Get("envconfig"); alt != "" {
		name = alt
	} else if isTrue(field.Tag.Get("split_words")) {
		name = splitWords(field.Name)
	}

	if prefix != "" {
		name = prefix + "_" + name
	}

	return strings.ToUpper(name)
}

func splitWords(s string) string {
	words := gatherEnvWords.FindAllStringSubmatch(s, -1)
	if len(words) == 0 {
		return s
	}

	parts := make([]string, 0, len(words))
	for _, w := range words {
		if m := acronymEnvWords.FindStringSubmatch(w[0]); len(m) == 3 {
			parts = append(parts, m[1], m[2])
		} else {
			parts = append(parts, w[0])
		}
	}

	return strings.Join(parts, "_")
}

func isTrue(s string) bool {
	b, _ := strconv.ParseBool(s)
	return b
}

func assignFromFile(envKey string, field reflect.Value, data []byte) error {
	switch {
	case field.Kind() == reflect.String:
		field.SetString(strings.TrimRight(string(data), "\r\n"))
		return nil
	case field.Kind() == reflect.Slice && field.Type().Elem().Kind() == reflect.String:
		values := splitListValues(string(data))
		slice := reflect.MakeSlice(field.Type(), len(values), len(values))
		for i, value := range values {
			slice.Index(i).SetString(value)
		}
		field.Set(slice)
		return nil
	default:
		return fmt.Errorf("%s%s is set but %s is not a string or a list of strings", envKey, fileSuffix, envKey)
	}
}

// splitListValues splits the contents of a file into list values, accepting
// either one value per line or comma-separated values.
func splitListValues(data string) []string {
	fields := strings.FieldsFunc(data, func(r rune) bool {
		return r == '\n' || r == '\r' || r == ','
	})

	values := make([]string, 0, len(fields))
	for _, f := range fields {
		if f = strings.TrimSpace(f); f != "" {
			values = append(values, f)
		}
	}

	return values
}
