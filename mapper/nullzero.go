package mapper

import (
	"reflect"
)

type nullZeroAgent struct {
	target reflect.Value
	agent  reflect.Value
}

func newNullZeroAgent(target reflect.Value) *nullZeroAgent {
	return &nullZeroAgent{
		target: target,
		agent:  reflect.New(reflect.PointerTo(target.Type())),
	}
}
func (s *nullZeroAgent) Agent() any {
	return s.agent.Interface()
}

func (s *nullZeroAgent) Apply() {
	value := s.agent.Elem()
	if value.IsNil() {
		s.target.Set(reflect.Zero(s.target.Type()))
	} else {
		s.target.Set(value.Elem())
	}
}
