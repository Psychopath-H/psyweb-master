package binding

import (
	"fmt"
	"github.com/go-playground/validator/v10"
	"reflect"
	"strings"
	"sync"
)

var Validator StructValidator = &defaultValidator{}

type StructValidator interface {
	// ValidateStruct 结构体验证，如果错误返回对应的错误信息
	ValidateStruct(any) error
	// Engine 返回对应使用的验证器
	Engine() any
}

// defaultValidator 是默认的验证器
type defaultValidator struct {
	one      sync.Once           //通常用于确保某个操作只会执行一次，即使在多个 goroutine 中被调用。它在并发编程中用于实现一次性初始化、全局变量的初始化或其他需要确保只执行一次的操作。
	validate *validator.Validate //第三方验证器
}

type SliceValidationError []error

func (err SliceValidationError) Error() string {
	n := len(err)
	switch n {
	case 0:
		return ""
	default:
		var b strings.Builder
		if err[0] != nil {
			fmt.Fprintf(&b, "[%d]: %s", 0, err[0].Error())
		}
		if n > 1 {
			for i := 1; i < n; i++ {
				if err[i] != nil {
					b.WriteString("\n")
					fmt.Fprintf(&b, "[%d]: %s", i, err[i].Error())
				}
			}
		}
		return b.String()
	}
}

// ValidateStruct 封装了使用了第三方验证器的方法
func (d *defaultValidator) ValidateStruct(obj any) error {
	if obj == nil {
		return nil
	}
	valueOf := reflect.ValueOf(obj)
	switch valueOf.Kind() {
	case reflect.Ptr:
		return d.ValidateStruct(valueOf.Elem().Interface())
	case reflect.Struct:
		return d.validateStruct(obj)
	case reflect.Slice, reflect.Array:
		count := valueOf.Len()
		validateRet := make(SliceValidationError, 0)
		for i := 0; i < count; i++ { //每一个都要验证一下，是否符合规范
			if err := d.validateStruct(valueOf.Index(i).Interface()); err != nil {
				validateRet = append(validateRet, err)
			}
		}
		if len(validateRet) == 0 {
			return nil
		}
		return validateRet
	default:
		return nil
	}
}

// Engine 返回该结构体的验证器
func (d *defaultValidator) Engine() any {
	d.LazyInit()
	return d.validate
}

// LazyInit 延迟初始化，即使在多个goroutine中被调用，也只进行一次验证器的初始化
func (d *defaultValidator) LazyInit() {
	d.one.Do(func() {
		d.validate = validator.New()
	})
}

// validateStruct 使用第三方验证器进行对数据进行验证
func (d *defaultValidator) validateStruct(obj any) error {
	d.LazyInit()
	return d.validate.Struct(obj)
}

func threePartValidate(obj any) error {
	return Validator.ValidateStruct(obj)
}
