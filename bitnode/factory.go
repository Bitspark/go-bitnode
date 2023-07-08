package bitnode

// A FactoryImplementation contain all required data for the Factory to implement a system.
type FactoryImplementation interface {
	Implement(sys System) (FactorySystem, error)
}

// A Factory allows adding custom implementations to a system.
type Factory interface {
	// The System providing system-level access to this Factory.
	//System() System

	// Parse parses the provided interface into a FactoryImplementation.
	Parse(data any) (FactoryImplementation, error)

	// Serialize serializes a FactoryImplementation.
	Serialize(impl FactoryImplementation) (any, error)
}
