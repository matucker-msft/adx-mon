// Code generated by 'yaegi extract net/http/pprof'. DO NOT EDIT.

//go:build go1.20
// +build go1.20

package stdlib

import (
	"net/http/pprof"
	"reflect"
)

func init() {
	Symbols["net/http/pprof/pprof"] = map[string]reflect.Value{
		// function, constant and variable definitions
		"Cmdline": reflect.ValueOf(pprof.Cmdline),
		"Handler": reflect.ValueOf(pprof.Handler),
		"Index":   reflect.ValueOf(pprof.Index),
		"Profile": reflect.ValueOf(pprof.Profile),
		"Symbol":  reflect.ValueOf(pprof.Symbol),
		"Trace":   reflect.ValueOf(pprof.Trace),
	}
}
