package jx

import "testing"

func TestOrderedMarshalKeepsInsertionOrder(t *testing.T) {
	o := NewOrdered().Set("key", "P-1").Set("summary", "a <b> & c").Set("key", "P-2")
	out, err := Marshal(o)
	if err != nil {
		t.Fatal(err)
	}
	want := `{"key":"P-2","summary":"a <b> & c"}`
	if string(out) != want {
		t.Errorf("got %s want %s", out, want)
	}
}

func TestOrderedSortIsStable(t *testing.T) {
	o := NewOrdered().Set("b", 1).Set("a", 1).Set("c", 2)
	o.Sort(func(x, y string) bool {
		vx, _ := o.Get(x)
		vy, _ := o.Get(y)
		return vx.(int) > vy.(int)
	})
	if keys := o.Keys(); keys[0] != "c" || keys[1] != "b" || keys[2] != "a" {
		t.Errorf("keys = %v", keys)
	}
}
