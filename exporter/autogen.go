package exporter

import "reflect"
import "github.com/guelfey/go.dbus"
import "errors"

func getTypeOf(ifc interface{}) (r reflect.Type) {
	r = reflect.TypeOf(ifc)
	if r.Kind() == reflect.Ptr {
		r = r.Elem()
	}
	return
}

func getValueOf(ifc interface{}) (r reflect.Value) {
	r = reflect.ValueOf(ifc)
	if r.Kind() == reflect.Ptr {
		r = r.Elem()
	}
	return
}

func genInterfaceInfo(ifc interface{}) *Interface {
	ifc_info := new(Interface)
	o_type := reflect.TypeOf(ifc)
	n := o_type.NumMethod()

	for i := 0; i < n; i++ {
		name := o_type.Method(i).Name
		method := Method{}
		method.Name = name

		m := o_type.Method(i).Type
		n_in := m.NumIn()
		n_out := m.NumOut()
		args := make([]Arg, 0)
		//Method's first paramter is the struct which this method bound to.
		for i := 1; i < n_in; i++ {
			args = append(args, Arg{
				Type:      dbus.SignatureOfType(m.In(i)).String(),
				Direction: "in",
			})
		}
		for i := 0; i < n_out; i++ {
			if m.Out(i) != reflect.TypeOf(&dbus.Error{}) {
				args = append(args, Arg{
					Type:      dbus.SignatureOfType(m.Out(i)).String(),
					Direction: "out",
				})
			}
		}
		method.Args = args
		ifc_info.Methods = append(ifc_info.Methods, method)
	}

	// generate properties if any
	if o_type.Kind() == reflect.Ptr {
		o_type = o_type.Elem()
	}
	n = o_type.NumField()
	for i := 0; i < n; i++ {
		field := o_type.Field(i)
		if field.Type.Kind() == reflect.Func {
			ifc_info.Signals = append(ifc_info.Signals, Signal{
				Name: field.Name,
				Args: func() []Arg {
					n := field.Type.NumIn()
					ret := make([]Arg, n)
					for i := 0; i < n; i++ {
						arg := field.Type.In(i)
						ret[i] = Arg{
							Type: dbus.SignatureOfType(arg).String(),
						}
					}
					return ret
				}(),
			})
		} else if field.PkgPath == "" {
			access := field.Tag.Get("access")
			if access != "read" {
				access = "readwrite"
			}
			ifc_info.Properties = append(ifc_info.Properties, Property{
				Name:   field.Name,
				Type:   dbus.SignatureOfType(field.Type).String(),
				Access: access,
			})
		}
	}

	return ifc_info
}

type DBusInfo struct {
	Dest, ObjectPath, Interface string
}
type DBusObject interface {
	GetDBusInfo() DBusInfo
}

func InstallOnSession(obj DBusObject) error {
	info := obj.GetDBusInfo()
	path := dbus.ObjectPath(info.ObjectPath)
	if path.IsValid() {
		return installOnSessionAny(obj, info.Dest, path, info.Interface)
	}
	return errors.New("ObjectPath " + info.ObjectPath + " is invalid")
}

//TODO: Need exported?
func installOnSessionAny(v interface{}, dest_name string, path dbus.ObjectPath, iface string) error {
	conn, err := dbus.SessionBus()
	if err != nil {
		return err
	}
	return export(conn, v, dest_name, path, iface)
}

//TODO: Need exported?
func export(c *dbus.Conn, v interface{}, name string, path dbus.ObjectPath, iface string) error {
	not_registered := true
	for _, _name := range c.Names() {
		if _name == name {
			not_registered = false
			break
		}

	}
	if not_registered {
		reply, _ := c.RequestName(name, dbus.NameFlagDoNotQueue)
		if reply != dbus.RequestNameReplyPrimaryOwner {
			return errors.New("name " + name + " already taken")
		}
	}

	err := c.Export(v, path, iface)
	if err != nil {
		return err
	}
	infos := c.GetObjectInfos(path)
	if _, ok := infos["org.freedesktop.DBus.Introspectable"]; !ok {
		infos["org.freedesktop.DBus.Introspectable"] = IntrospectProxy{infos}
	}
	if _, ok := infos["org.freedesktop.DBus.Properties"]; !ok {
		infos["org.freedesktop.DBus.Properties"] = PropertiesProxy{infos}
	}
	return nil
}
