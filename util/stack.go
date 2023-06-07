package util

import (
	"fmt"
	"strings"
)

type Stack[T comparable] struct {
	values []T
}

func NewStack[T comparable]() *Stack[T] {
	return &Stack[T]{}
}

func (v *Stack[T]) Push(t T) {
	v.values = append(v.values, t)
}

func (v *Stack[T]) Pop() T {
	if len(v.values) == 0 {
		var t T
		return t
	}
	t := v.values[len(v.values)-1]
	v.values = v.values[:len(v.values)-1]
	return t
}

func (v *Stack[T]) Value() T {
	if len(v.values) == 0 {
		var t T
		return t
	}
	return v.values[len(v.values)-1]
}

func (v *Stack[T]) Index(n int) T {
	return v.values[len(v.values)-1-n]
}

func (v *Stack[T]) Empty() []T {
	ts := v.values
	v.values = nil
	return ts
}

func (v *Stack[T]) Clear() {
	v.values = nil
}

func (v *Stack[T]) Size() int {
	return len(v.values)
}

func (v *Stack[T]) String() string {
	return fmt.Sprintf("%v", v.values)
}

func (v *Stack[T]) Join(sep string) string {
	strs := []string{}
	for _, s := range v.values {
		strs = append(strs, fmt.Sprintf("%v", s))
	}
	return strings.Join(strs, sep)
}
