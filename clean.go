package cii

import (
	"reflect"
	"strings"
)

// cleanDocument strips the Unicode replacement character (U+FFFD) from every
// string in the parsed document. A sender emits it where its own encoding
// handling gave up, and a CharsetReader emits it for a byte undefined in the
// declared encoding; either way gobl's canonical JSON refuses it and the whole
// document fails to digest over damaged free text.
//
// It runs once, on the built document, so no field can be forgotten and no
// caller has to remember. Working on decoded values also means XML character
// references such as &#xFFFD; are already resolved.
func cleanDocument(doc any) {
	cleanValue(reflect.ValueOf(doc))
}

func cleanValue(v reflect.Value) {
	switch v.Kind() {
	case reflect.Pointer, reflect.Interface:
		if !v.IsNil() {
			cleanValue(v.Elem())
		}
	case reflect.Struct:
		for i := range v.NumField() {
			if v.Type().Field(i).IsExported() {
				cleanValue(v.Field(i))
			}
		}
	case reflect.Slice, reflect.Array:
		for i := range v.Len() {
			cleanValue(v.Index(i))
		}
	case reflect.Map:
		for _, k := range v.MapKeys() {
			e := v.MapIndex(k)
			if e.Kind() == reflect.String {
				if s := clean(e.String()); s != e.String() {
					n := reflect.New(e.Type()).Elem()
					n.SetString(s)
					v.SetMapIndex(k, n)
				}
				continue
			}
			cleanValue(e)
		}
	case reflect.String:
		if v.CanSet() {
			if s := clean(v.String()); s != v.String() {
				v.SetString(s)
			}
		}
	}
}

func clean(s string) string {
	return strings.ReplaceAll(s, "�", "")
}
