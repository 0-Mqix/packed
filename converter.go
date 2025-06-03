package packed

import (
	"reflect"
)

func ValidateSize(converter any) bool {
	if converter == nil {
		return false
	}

	val := reflect.ValueOf(converter)
	if !val.IsValid() {
		return false
	}

	method := val.MethodByName("Size")
	if !method.IsValid() {
		return false
	}

	if method.Type().NumIn() != 0 || method.Type().NumOut() != 1 {
		return false
	}

	if method.Type().Out(0) != reflect.TypeOf(int(0)) {
		return false
	}

	return true
}

func ValidateConverter(methodName string, converter any) (reflect.Type, bool) {
	if converter == nil {
		return nil, false
	}

	converterVal := reflect.ValueOf(converter)
	if !converterVal.IsValid() {
		return nil, false
	}

	method := converterVal.MethodByName(methodName)
	if !method.IsValid() {
		return nil, false
	}

	methodType := method.Type()
	if methodType.NumIn() != 3 || methodType.NumOut() != 0 {
		return nil, false
	}

	receiverType := methodType.In(0)
	if receiverType.Kind() != reflect.Ptr {
		return nil, false
	}

	if methodType.In(1) != reflect.TypeOf([]byte{}) {
		return nil, false
	}

	if methodType.In(2) != reflect.TypeOf(int(0)) {
		return nil, false
	}

	return receiverType.Elem(), true
}

func implementsConverterInterface(converter any) (reflect.Type, bool) {

	if !ValidateSize(converter) {
		return nil, false
	}

	recievers := []reflect.Type{}

	for _, methodName := range []string{"ToBytesLittleEndian", "FromBytesLittleEndian", "ToBytesBigEndian", "FromBytesBigEndian"} {
		reciever, valid := ValidateConverter(methodName, converter)
		if !valid {
			return nil, false
		}
		recievers = append(recievers, reciever)
	}

	first := recievers[0]

	for _, reciever := range recievers[1:] {
		if first != reciever {
			return nil, false
		}
	}

	return first, true
}
