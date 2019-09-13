package main

import "reflect"

func IsNil(obj interface{}) bool {
	if obj == nil {
		return true
	}
	return false
}

func test1(i *interface{}) {
	rv := reflect.ValueOf(i)
	rv.CanAddr()
	rv.Addr()
}

func test2(i *interface{}) {
	rv := reflect.ValueOf(i)
	if true {
		rv.CanAddr()
	} else {
		rv.CanAddr()
	}
	rv.Addr() // want `reflect.CanAddr should be called before calling reflect.Addr`
}

func main() {
	println(IsNil(nil)) // true

	test1(nil)
	test2(nil)

	var x *int = nil
	println(IsNil(x)) // false
}
