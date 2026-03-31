package models

import (
	"fmt"
	"reflect"
)

type TagValidator struct {
	errors []string
}

func NewTagValidator() *TagValidator {
	return &TagValidator{
		errors: make([]string, 0),
	}
}

func (v *TagValidator) checkStructTags(model interface{}) {
	t := reflect.TypeOf(model)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	if t.Kind() != reflect.Struct {
		return
	}

	structName := t.Name()

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)

		if field.PkgPath != "" {
			continue
		}

		dbTag := field.Tag.Get("db")
		bsonTag := field.Tag.Get("bson")

		if dbTag == "" || dbTag == "-" {
			v.errors = append(v.errors, fmt.Sprintf("[%s.%s] 缺少 db 标签", structName, field.Name))
		}

		if bsonTag == "" || bsonTag == "-" {
			v.errors = append(v.errors, fmt.Sprintf("[%s.%s] 缺少 bson 标签", structName, field.Name))
		}
	}
}

func (v *TagValidator) ValidateAllModels() error {
	v.checkStructTags(Account{})
	v.checkStructTags(Player{})
	v.checkStructTags(Auction{})
	v.checkStructTags(AuctionLog{})
	v.checkStructTags(LoginLog{})
	v.checkStructTags(PlayerBuff{})
	v.checkStructTags(PlayerItem{})
	v.checkStructTags(PlayerQuest{})
	v.checkStructTags(PlayerSkill{})
	v.checkStructTags(QuestLog{})

	if len(v.errors) > 0 {
		errMsg := "模型结构体标签验证失败:\n"
		for _, err := range v.errors {
			errMsg += "  - " + err + "\n"
		}
		return fmt.Errorf("%s", errMsg)
	}

	return nil
}

func ValidateModelTags() error {
	validator := NewTagValidator()
	return validator.ValidateAllModels()
}
