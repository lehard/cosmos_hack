package world

import (
	"fmt"
	"reflect"
)

// resolveTimes разбирает все T и Span описания (рекурсивно по структурам,
// срезам и указателям) в месяце и поясе сценария.
func (c clock) resolveTimes(v reflect.Value) error {
	switch v.Kind() {
	case reflect.Pointer:
		if v.IsNil() {
			return nil
		}
		return c.resolveTimes(v.Elem())
	case reflect.Slice:
		for i := 0; i < v.Len(); i++ {
			if err := c.resolveTimes(v.Index(i)); err != nil {
				return err
			}
		}
	case reflect.Struct:
		switch x := v.Addr().Interface().(type) {
		case *T:
			return c.parseT(x)
		case *Span:
			return c.parseSpan(x)
		}
		for i := 0; i < v.NumField(); i++ {
			if !v.Type().Field(i).IsExported() {
				continue
			}
			if err := c.resolveTimes(v.Field(i)); err != nil {
				return fmt.Errorf("%s: %w", v.Type().Field(i).Name, err)
			}
		}
	}
	return nil
}
