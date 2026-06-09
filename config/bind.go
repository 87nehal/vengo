package config

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"
)

func Bind(cfg *Config, target any) error {
	value := reflect.ValueOf(target)
	if value.Kind() != reflect.Ptr || value.IsNil() {
		return fmt.Errorf("bind target must be a non-nil pointer to a struct")
	}
	element := value.Elem()
	if element.Kind() != reflect.Struct {
		return fmt.Errorf("bind target must be a pointer to a struct, got pointer to %s", element.Kind())
	}
	return bindStruct(cfg, element, "")
}

func bindStruct(cfg *Config, value reflect.Value, prefix string) error {
	typ := value.Type()
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		fieldValue := value.Field(i)

		if !fieldValue.CanSet() {
			continue
		}

		if field.Anonymous {
			if fieldValue.Kind() == reflect.Struct {
				if err := bindStruct(cfg, fieldValue, prefix); err != nil {
					return err
				}
			}
			continue
		}

		configKey := field.Tag.Get("config")
		defaultValue, hasDefault := field.Tag.Lookup("default")

		if configKey == "" && fieldValue.Kind() == reflect.Struct && !isSpecialType(fieldValue.Type()) {
			nestedPrefix := prefix
			if nestedPrefix != "" {
				nestedPrefix += "." + strings.ToLower(field.Name)
			} else {
				nestedPrefix = strings.ToLower(field.Name)
			}
			if err := bindStruct(cfg, fieldValue, nestedPrefix); err != nil {
				return fmt.Errorf("field %s: %w", field.Name, err)
			}
			continue
		}

		if configKey == "" {
			configKey = prefix + "." + strings.ToLower(field.Name)
			if prefix == "" {
				configKey = strings.ToLower(field.Name)
			}
		}

		raw, found := cfg.Get(configKey)
		if !found {
			if hasDefault {
				raw = defaultValue
			} else {
				continue
			}
		}

		if err := setField(fieldValue, raw, field.Name); err != nil {
			return fmt.Errorf("field %s (key %q): %w", field.Name, configKey, err)
		}
	}
	return nil
}

func isSpecialType(t reflect.Type) bool {
	return t == reflect.TypeOf(time.Duration(0))
}

func setField(field reflect.Value, raw string, name string) error {
	if field.Type() == reflect.TypeOf(time.Duration(0)) {
		d, err := time.ParseDuration(raw)
		if err != nil {
			return fmt.Errorf("parse duration: %w", err)
		}
		field.Set(reflect.ValueOf(d))
		return nil
	}

	switch field.Kind() {
	case reflect.String:
		field.SetString(raw)
	case reflect.Bool:
		b, err := strconv.ParseBool(raw)
		if err != nil {
			return fmt.Errorf("parse bool: %w", err)
		}
		field.SetBool(b)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		n, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			return fmt.Errorf("parse int: %w", err)
		}
		field.SetInt(n)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		n, err := strconv.ParseUint(raw, 10, 64)
		if err != nil {
			return fmt.Errorf("parse uint: %w", err)
		}
		field.SetUint(n)
	case reflect.Float32, reflect.Float64:
		f, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			return fmt.Errorf("parse float: %w", err)
		}
		field.SetFloat(f)
	default:
		return fmt.Errorf("unsupported field type %s", field.Kind())
	}
	return nil
}
