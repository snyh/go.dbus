package dbus

import "encoding/xml"
import "bytes"
import "reflect"

type IntrospectProxy struct {
	infos map[string]interface{}
	child map[string]bool
}

func (i IntrospectProxy) String() string {
	// i.infos reference i so can't use default String()
	ret := "IntrospectProxy ["
	comma := false
	for k, _ := range i.infos {
		if comma {
			ret += ","
		}
		comma = true
		ret += `"` + k + `"`
	}
	ret += "]"
	return ret
}

func (i IntrospectProxy) Introspect() (string, *Error) {
	var node = new(NodeInfo)
	for k, _ := range i.child {
		node.Children = append(node.Children, NodeInfo{
			Name: k,
		})
	}
	for name, ifc := range i.infos {
		info := genInterfaceInfo(ifc)
		info.Name = name
		node.Interfaces = append(node.Interfaces, *info)
	}
	var buffer bytes.Buffer

	writer := xml.NewEncoder(&buffer)
	writer.Indent("", "     ")
	writer.Encode(node)
	return buffer.String(), nil
}

type PropertiesProxy struct {
	infos             map[string]interface{}
	PropertiesChanged func(string, map[string]Variant, []string)
}
type Property interface {
	Get() interface{}
	Set(interface{})
	GetType() reflect.Type
}

var errUnknownProperty = Error{
	"org.freedesktop.DBus.Error.UnknownProperty",
	[]interface{}{"Unknown / invalid Property"},
}
var errUnKnowInterface = Error{
	"org.freedesktop.DBus.Error.NoSuchInterface",
	[]interface{}{"No such interface"},
}
var errPropertyNotWritable = Error{
	"org.freedesktop.DBus.Error.NoWritable",
	[]interface{}{"Can't write this property."},
}

func (i PropertiesProxy) GetAll(ifc_name string) map[string]Variant {
	props := make(map[string]Variant)
	if ifc, ok := i.infos[ifc_name]; ok {
		o_type := getTypeOf(ifc)
		n := o_type.NumField()
		for i := 0; i < n; i++ {
			field := o_type.Field(i)
			if field.Type.Kind() != reflect.Func && field.PkgPath == "" {
				props[field.Name] = MakeVariant(getValueOf(ifc).Field(i).Interface())
			}
		}
	}
	return props
}

func (i PropertiesProxy) Set(ifc_name string, prop_name string, value Variant) *Error {
	if ifc, ok := i.infos[ifc_name]; ok {
		ifc_t := getTypeOf(ifc)
		t, ok := ifc_t.FieldByName(prop_name)
		v := getValueOf(ifc).FieldByName(prop_name)
		if ok && v.IsValid() && "read" != t.Tag.Get("access") {
			if v.Type().Implements(propertyType) {
				if reflect.TypeOf(value.Value()) == v.MethodByName("GetType").Interface().(func() reflect.Type)() {
					v.MethodByName("Set").Interface().(func(interface{}))(value.Value())
					fn := getValueOf(ifc).MethodByName("OnPropertiesChanged")
					if fn.IsValid() && !fn.IsNil() {
						fn.Call([]reflect.Value{reflect.ValueOf(prop_name), reflect.Zero(reflect.TypeOf(value.Value()))})
					}
					return nil
				} else {
					return &errPropertyNotWritable
				}
			}
			if v.CanAddr() && v.Type() == reflect.TypeOf(value.Value()) {
				prop_val := reflect.ValueOf(value.Value())
				prop_old_val := v.Interface()
				v.Set(prop_val)
				fn := getValueOf(ifc).MethodByName("OnPropertiesChanged")
				if fn.IsValid() && !fn.IsNil() {
					fn.Call([]reflect.Value{reflect.ValueOf(prop_name), reflect.ValueOf(prop_old_val)})
				}
				return nil
			} else {
				return &errPropertyNotWritable
			}
		} else {
			return &errUnknownProperty
		}
	}
	return &errUnKnowInterface
}
func (i PropertiesProxy) Get(ifc_name string, prop_name string) (Variant, *Error) {
	if ifc, ok := i.infos[ifc_name]; ok {
		value := getValueOf(ifc).FieldByName(prop_name)
		if value.Type().Implements(propertyType) {
			t := value.MethodByName("Get").Interface().(func() interface{})()
			return MakeVariant(t), nil
		}
		if value.IsValid() {
			return MakeVariant(value.Interface()), nil
		} else {
			return MakeVariant(""), &errUnknownProperty
		}
	} else {
		return MakeVariant(""), &errUnKnowInterface
	}
}
