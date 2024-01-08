package binding

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
)

var DisallowUnknownFields = false
var UsingLocalValidate = true

type jsonBinding struct {
}

func (j jsonBinding) Bind(r *http.Request, obj any) error {
	if r == nil || r.Body == nil {
		return errors.New("invalid request")
	}
	return j.decodeJson(r.Body, obj)
}

func (j jsonBinding) Name() string {
	return "json"
}

func (j jsonBinding) decodeJson(body io.Reader, obj any) error {
	decoder := json.NewDecoder(body) //创建一个新的 JSON 解码器,解码器将从该输入流中读取数据并解码为 Go 数据结构。
	if DisallowUnknownFields {
		decoder.DisallowUnknownFields()
	}
	if UsingLocalValidate { //使用了本地验证器
		return validateParam(obj, decoder) //那就不应该进入第三方验证器,直接返回
	}
	if err := decoder.Decode(obj); err != nil { //将json数据解码到obj变量中
		return err
	}
	return threePartValidate(obj) //使用第三方验证器
}

// validateParam 是本地自己实现的Json数据验证器
func validateParam(obj any, decoder *json.Decoder) error {
	if obj == nil {
		return nil
	}
	//解析为对应的map，根据map中的key进行对比
	//判断类型为结构体才能解析为map
	//反射
	//obj和valueOf都是我们预先设定好的某个数据结构，从Post传递过来的参数要和它进行比对
	valueOf := reflect.ValueOf(obj) //它接受一个普通的 Go 值 obj，然后返回一个 reflect.Value 类型的值，这个值包含了 obj 的类型信息和值。
	//判断是否为指针类型
	if valueOf.Kind() != reflect.Pointer {
		return errors.New("this argument must have a pointer type")
	}
	elem := valueOf.Elem().Interface() //把指针指向的具体类型拿出来
	of := reflect.ValueOf(elem)        //包装为reflect.Value类型，之后做下一步判断
	switch of.Kind() {
	case reflect.Struct:
		return checkParam(of, obj, decoder)
	case reflect.Slice, reflect.Array:
		elem := of.Type().Elem() //Elem() 是 reflect.Type 类型的方法，用于获取切片或数组中元素的类型。这通常用于检索切片或数组的元素类型
		elemType := elem.Kind()
		if elemType == reflect.Struct {
			return checkParamSlice(elem, obj, decoder)
		}
	default:
		_ = decoder.Decode(obj)
	}
	return nil
}

// checkParam 比对post传递过来的参数是否符合所需要的
func checkParam(valueOf reflect.Value, obj any, decoder *json.Decoder) error {
	//解析为map，然后根据map中的key进行对比
	//判断类型为结构体才能解析为参数
	mapValue := make(map[string]interface{})
	_ = decoder.Decode(&mapValue) //将post传递过来的json参数解析到mapValue结构体中
	if len(mapValue) <= 0 {
		return nil
	}
	for i := 0; i < valueOf.NumField(); i++ {
		field := valueOf.Type().Field(i)
		name := field.Name
		required := field.Tag.Get("binding")
		jsonName := field.Tag.Get("json")
		if jsonName != "" {
			name = jsonName
		}
		value := mapValue[name]                     //查询
		if value == nil && required == "required" { //如果说解析出来的map中没有所需要的特定某个字段，那就报错
			return errors.New(fmt.Sprintf("field [%s] not exist but [%s] is required", jsonName, jsonName))
		}
	}
	//为什么要用这两行代码，因为decoder的流在上面 _ = decoder.Decode(&mapValue) 使用过一次decode以后就无法再进行decode操作
	//所以obj无法拿到值了，所以重新编码解析一下，放进去，obj传进来的是个地址，所以这里对obj的解析是重要的，万一能够正常返回数据
	b, _ := json.Marshal(mapValue)
	_ = json.Unmarshal(b, obj)
	return nil
}

func checkParamSlice(valueType reflect.Type, obj any, decoder *json.Decoder) error {
	//解析为map，然后根据map中的key进行对比
	//判断类型为结构体才能解析为参数
	mapValue := make([]map[string]interface{}, 0)
	_ = decoder.Decode(&mapValue)
	if len(mapValue) <= 0 {
		return nil
	}
	for i := 0; i < valueType.NumField(); i++ {
		field := valueType.Field(i)
		name := field.Name
		required := field.Tag.Get("binding")
		jsonName := field.Tag.Get("json")
		if jsonName != "" {
			name = jsonName
		}
		for _, v := range mapValue { //post过来里的json数据没有包含要求的属性就报错
			value := v[name]
			if value == nil && required == "required" {
				return errors.New(fmt.Sprintf("filed [%s] is not exist, because [%s] is required", jsonName, jsonName))
			}
		}
		fmt.Println("fieldName", name)
	}
	b, _ := json.Marshal(mapValue)
	_ = json.Unmarshal(b, obj)
	return nil
}
