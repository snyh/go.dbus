package exporter

import "github.com/guelfey/go.dbus"
import "encoding/xml"
import "bytes"
import "reflect"

type IntrospectProxy struct {
	infos map[string]interface{}
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

func (i IntrospectProxy) Introspect() (string, *dbus.Error) {
	var node = new(Node)
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
	infos map[string]interface{}
}

var errUnknownProperty = dbus.Error{
	"org.freedesktop.DBus.Error.UnknownProperty",
	[]interface{}{"Unknown / invalid Property"},
}
var errUnKnowInterface = dbus.Error{
	"org.freedesktop.DBus.Error.NoSuchInterface",
	[]interface{}{"No such interface"},
}
var errPropertyNotWritable = dbus.Error{
	"org.freedesktop.DBus.Error.NoWritable",
	[]interface{}{"Can't write this property."},
}

func (i PropertiesProxy) GetAll(ifc_name string) map[string]dbus.Variant {
	props := make(map[string]dbus.Variant)
	if ifc, ok := i.infos[ifc_name]; ok {
		o_type := getTypeOf(ifc)
		n := o_type.NumField()
		for i := 0; i < n; i++ {
			field := o_type.Field(i)
			if field.Type.Kind() != reflect.Func && field.PkgPath == "" {
				props[field.Name] = dbus.MakeVariant(getValueOf(ifc).Field(i).Interface())
			}
		}
	}
	return props
}

func (i PropertiesProxy) Set(ifc_name string, prop_name string, value dbus.Variant) *dbus.Error {
	if ifc, ok := i.infos[ifc_name]; ok {
		ifc_t := getTypeOf(ifc)
		t, ok := ifc_t.FieldByName(prop_name)
		v := getValueOf(ifc).FieldByName(prop_name)
		if ok && v.IsValid() {
			if v.CanAddr() && "read" != t.Tag.Get("access") && v.Type() == reflect.TypeOf(value.Value()) {
				v.Set(reflect.ValueOf(value.Value()))
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
func (i PropertiesProxy) Get(ifc_name string, prop_name string) (dbus.Variant, *dbus.Error) {
	if ifc, ok := i.infos[ifc_name]; ok {
		value := getValueOf(ifc).FieldByName(prop_name)
		if value.IsValid() {
			return dbus.MakeVariant(value.Interface()), nil
		} else {
			return dbus.MakeVariant(""), &errUnknownProperty
		}
	} else {
		return dbus.MakeVariant(""), &errUnKnowInterface
	}
}
