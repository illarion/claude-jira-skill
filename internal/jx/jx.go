// Package jx holds small helpers for working with untyped JSON values
// (map[string]any / []any / float64 / string / bool / nil).
package jx

import (
	"bytes"
	"encoding/json"
)

// Obj casts v to a map, returning nil when v is not an object.
func Obj(v any) map[string]any {
	m, _ := v.(map[string]any)
	return m
}

// Arr casts v to a slice, returning nil when v is not an array.
func Arr(v any) []any {
	a, _ := v.([]any)
	return a
}

// Str returns v as a string, or "" for any other type.
func Str(v any) string {
	s, _ := v.(string)
	return s
}

// Bool returns v as a bool, or def when v is absent or not a bool.
func Bool(v any, def bool) bool {
	b, ok := v.(bool)
	if !ok {
		return def
	}
	return b
}

// Int returns v as an int (JSON numbers arrive as float64), or 0.
func Int(v any) int {
	if f, ok := v.(float64); ok {
		return int(f)
	}
	return 0
}

// Ordered is a JSON object whose keys marshal in insertion order, like a
// Python dict. Setting an existing key keeps its position.
type Ordered struct {
	keys []string
	vals map[string]any
}

// NewOrdered returns an empty ordered object.
func NewOrdered() *Ordered {
	return &Ordered{vals: map[string]any{}}
}

// Set stores v under k and returns o for chaining.
func (o *Ordered) Set(k string, v any) *Ordered {
	if _, ok := o.vals[k]; !ok {
		o.keys = append(o.keys, k)
	}
	o.vals[k] = v
	return o
}

// Get returns the value stored under k.
func (o *Ordered) Get(k string) (any, bool) {
	v, ok := o.vals[k]
	return v, ok
}

// Keys returns the keys in insertion order.
func (o *Ordered) Keys() []string {
	return append([]string{}, o.keys...)
}

// Len returns the number of keys.
func (o *Ordered) Len() int {
	return len(o.keys)
}

// Sort reorders the keys with less; the sort is stable.
func (o *Ordered) Sort(less func(a, b string) bool) {
	keys := o.keys
	for i := 1; i < len(keys); i++ {
		for j := i; j > 0 && less(keys[j], keys[j-1]); j-- {
			keys[j], keys[j-1] = keys[j-1], keys[j]
		}
	}
}

// MarshalJSON writes the object with keys in insertion order and without
// HTML escaping.
func (o *Ordered) MarshalJSON() ([]byte, error) {
	var b bytes.Buffer
	b.WriteByte('{')
	for i, k := range o.keys {
		if i > 0 {
			b.WriteByte(',')
		}
		key, err := Marshal(k)
		if err != nil {
			return nil, err
		}
		val, err := Marshal(o.vals[k])
		if err != nil {
			return nil, err
		}
		b.Write(key)
		b.WriteByte(':')
		b.Write(val)
	}
	b.WriteByte('}')
	return b.Bytes(), nil
}

// Marshal encodes v as compact JSON without HTML escaping.
func Marshal(v any) ([]byte, error) {
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	return bytes.TrimRight(b.Bytes(), "\n"), nil
}
