package bitnode

import "testing"

type myTestFac struct {
}

var _ Factory = &myTestFac{}

func (m myTestFac) Parse(data any) (FactoryImplementation, error) {
	//TODO implement me
	panic("implement me")
}

func (m myTestFac) Serialize(impl FactoryImplementation) (any, error) {
	//TODO implement me
	panic("implement me")
}

func TestFactory1(t *testing.T) {
	f1 := myTestFac{}
	f2 := myTestFac{}
	n1 := NewNode()
	n2 := NewNode()

	_ = n1.AddFactory("f", f1)
	_ = n2.AddFactory("f", f2)
}
