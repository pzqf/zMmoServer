package config

import (
	"fmt"
	"os"
	"reflect"
	"strconv"

	"github.com/pzqf/zEngine/zLog"
	"gopkg.in/ini.v1"
	"go.uber.org/zap"
)

type Loader struct {
	file  *ini.File
	env   bool
}

type LoaderOption func(*Loader)

func WithEnvOverride() LoaderOption {
	return func(l *Loader) {
		l.env = true
	}
}

func Load(configPath string, opts ...LoaderOption) (*Loader, error) {
	file, err := ini.Load(configPath)
	if err != nil {
		return nil, fmt.Errorf("load config file %s: %w", configPath, err)
	}

	loader := &Loader{
		file: file,
		env:  false,
	}

	for _, opt := range opts {
		opt(loader)
	}

	return loader, nil
}

func LoadFromString(data []byte, opts ...LoaderOption) (*Loader, error) {
	file, err := ini.Load(data)
	if err != nil {
		return nil, fmt.Errorf("load config from data: %w", err)
	}

	loader := &Loader{
		file: file,
		env:  false,
	}

	for _, opt := range opts {
		opt(loader)
	}

	return loader, nil
}

func (l *Loader) Unmarshal(cfg interface{}) error {
	v := reflect.ValueOf(cfg)
	if v.Kind() != reflect.Ptr || v.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("cfg must be a pointer to struct")
	}

	return l.unmarshalStruct(v.Elem())
}

func (l *Loader) unmarshalStruct(v reflect.Value) error {
	t := v.Type()

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		fieldValue := v.Field(i)

		if !fieldValue.CanSet() {
			continue
		}

		sectionName := field.Tag.Get("ini")
		if sectionName == "" || sectionName == "-" {
			continue
		}

		if fieldValue.Kind() == reflect.Struct {
			section, err := l.file.GetSection(sectionName)
			if err != nil {
				zLog.Debug("Config section not found, using defaults",
					zap.String("section", sectionName))
				continue
			}
			if err := l.unmarshalSection(section, fieldValue); err != nil {
				return fmt.Errorf("section %s: %w", sectionName, err)
			}
		}
	}

	return nil
}

func (l *Loader) unmarshalSection(section *ini.Section, v reflect.Value) error {
	t := v.Type()

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		fieldValue := v.Field(i)

		if !fieldValue.CanSet() {
			continue
		}

		keyName := field.Tag.Get("ini")
		if keyName == "" || keyName == "-" {
			continue
		}

		key, err := section.GetKey(keyName)
		if err != nil {
			continue
		}

		if l.env {
			if envVal := os.Getenv(keyName); envVal != "" {
				if err := setFieldValue(fieldValue, envVal); err == nil {
					continue
				}
			}
			envKeyName := toEnvName(t.Name(), keyName)
			if envVal := os.Getenv(envKeyName); envVal != "" {
				if err := setFieldValue(fieldValue, envVal); err == nil {
					continue
				}
			}
		}

		if err := setFieldValue(fieldValue, key.String()); err != nil {
			return fmt.Errorf("field %s: %w", keyName, err)
		}
	}

	return nil
}

func setFieldValue(v reflect.Value, value string) error {
	switch v.Kind() {
	case reflect.String:
		v.SetString(value)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		intVal, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return fmt.Errorf("parse int: %w", err)
		}
		v.SetInt(intVal)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		uintVal, err := strconv.ParseUint(value, 10, 64)
		if err != nil {
			return fmt.Errorf("parse uint: %w", err)
		}
		v.SetUint(uintVal)
	case reflect.Float32, reflect.Float64:
		floatVal, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return fmt.Errorf("parse float: %w", err)
		}
		v.SetFloat(floatVal)
	case reflect.Bool:
		boolVal, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("parse bool: %w", err)
		}
		v.SetBool(boolVal)
	default:
		return fmt.Errorf("unsupported type: %s", v.Kind())
	}
	return nil
}

func toEnvName(sectionName, keyName string) string {
	result := sectionName + "_" + keyName
	result = replaceNonAlphaNum(result)
	return result
}

func replaceNonAlphaNum(s string) string {
	result := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') {
			if c >= 'a' && c <= 'z' {
				c -= 32
			}
			result = append(result, c)
		} else if c == '_' || c == '-' || c == '.' {
			result = append(result, '_')
		}
	}
	return string(result)
}

func (l *Loader) GetSection(name string) (*ini.Section, error) {
	return l.file.GetSection(name)
}

func (l *Loader) GetKey(section, key string) (string, error) {
	s, err := l.file.GetSection(section)
	if err != nil {
		return "", err
	}
	k, err := s.GetKey(key)
	if err != nil {
		return "", err
	}
	return k.String(), nil
}
