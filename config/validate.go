package config

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
)

type Validator interface {
	Validate() error
}

func Validate(target any) error {
	value := reflect.ValueOf(target)
	if value.Kind() == reflect.Ptr {
		if value.IsNil() {
			return fmt.Errorf("validate target cannot be nil")
		}
		value = value.Elem()
	}
	if value.Kind() != reflect.Struct {
		return fmt.Errorf("validate target must be a struct, got %s", value.Kind())
	}

	if v, ok := target.(Validator); ok {
		if err := v.Validate(); err != nil {
			return err
		}
	}

	return validateFields(value, "")
}

func validateFields(value reflect.Value, prefix string) error {
	typ := value.Type()
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		fieldValue := value.Field(i)

		if !fieldValue.CanSet() && !field.IsExported() {
			continue
		}

		if field.Anonymous && fieldValue.Kind() == reflect.Struct {
			if err := validateFields(fieldValue, prefix); err != nil {
				return err
			}
			continue
		}

		if fieldValue.Kind() == reflect.Struct && !isSpecialType(fieldValue.Type()) {
			nestedPrefix := field.Name
			if prefix != "" {
				nestedPrefix = prefix + "." + field.Name
			}
			if err := validateFields(fieldValue, nestedPrefix); err != nil {
				return err
			}
			continue
		}

		tags := field.Tag.Get("validate")
		if tags == "" {
			continue
		}

		fieldPath := field.Name
		if prefix != "" {
			fieldPath = prefix + "." + field.Name
		}

		for _, tag := range strings.Split(tags, ",") {
			tag = strings.TrimSpace(tag)
			switch tag {
			case "required":
				if err := validateRequired(fieldValue, fieldPath); err != nil {
					return err
				}
			case "nonzero":
				if err := validateNonZero(fieldValue, fieldPath); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func validateRequired(field reflect.Value, path string) error {
	switch field.Kind() {
	case reflect.String:
		if field.String() == "" {
			return fmt.Errorf("field %s is required", path)
		}
	case reflect.Ptr, reflect.Interface:
		if field.IsNil() {
			return fmt.Errorf("field %s is required", path)
		}
	case reflect.Slice, reflect.Map:
		if field.IsNil() || field.Len() == 0 {
			return fmt.Errorf("field %s is required", path)
		}
	}
	return nil
}

func validateNonZero(field reflect.Value, path string) error {
	if field.IsZero() {
		return fmt.Errorf("field %s must not be zero", path)
	}
	return nil
}

func BindAndValidate(cfg *Config, target any) error {
	if err := Bind(cfg, target); err != nil {
		return err
	}
	return Validate(target)
}

func BindAndValidateFromApp(app interface{ Config() (any, bool) }, target any) error {
	raw, exists := app.Config()
	if !exists {
		return errors.New("no config registered on app")
	}
	cfg, ok := raw.(*Config)
	if !ok {
		return fmt.Errorf("registered config has type %T, want *config.Config", raw)
	}
	return BindAndValidate(cfg, target)
}
