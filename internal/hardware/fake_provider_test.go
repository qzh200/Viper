package hardware

import (
	"fmt"

	"viper/internal/wmi"
)

// The fake provider answers the calls of the backend from a table, so
// that the tests of the hardware layer do not need the rights of an
// administrator and do not touch the machine.

// fakeCall is one call that the backend made.
type fakeCall struct {
	// Class is the class the method belongs to.
	Class string
	// Method is the name of the method.
	Method string
	// Path is the object path the method was called on.
	Path string
	// Input is the input parameter of the call.
	Input uint64
	// Bytes is the input parameter of a call that takes a buffer.
	Bytes []byte
}

// fakeProvider is a provider that answers from a table.
type fakeProvider struct {
	// outputs holds the answer of every method, keyed by the name of the
	// method.
	outputs map[string]map[string]wmi.Value
	// inputs holds the input word of every method, keyed by the name of
	// the method.
	inputs map[string]uint64
	// buffers holds the input buffer of every method, keyed by the name
	// of the method.
	buffers map[string][]byte
	// sequences holds the answers of the methods that are called several
	// times, in the order of the calls.
	sequences map[string][]uint64
	// counts counts the calls of every method.
	counts map[string]int
	// calls records every call that was made.
	calls []fakeCall
	// err is the error that the provider answers with, it stops the call
	// before the table is looked at.
	err error
}

// newFakeProvider builds a provider that answers nothing.
func newFakeProvider() *fakeProvider {
	return &fakeProvider{
		outputs:   map[string]map[string]wmi.Value{},
		inputs:    map[string]uint64{},
		buffers:   map[string][]byte{},
		sequences: map[string][]uint64{},
		counts:    map[string]int{},
	}
}

// answer sets the answer of a method.
func (p *fakeProvider) answer(method string, outputs map[string]wmi.Value) *fakeProvider {
	p.outputs[method] = outputs
	return p
}

// output sets the answer of a method to a single number.
func (p *fakeProvider) output(method, name string, value uint64) *fakeProvider {
	existing := p.outputs[method]
	if existing == nil {
		existing = map[string]wmi.Value{}
		p.outputs[method] = existing
	}
	existing[name] = wmi.Uint64(value)
	return p
}

// fail makes the provider refuse every call.
func (p *fakeProvider) fail(err error) *fakeProvider {
	p.err = err
	return p
}

// sequence sets the answers of a method that is called several times.
//
// The answers are handed out in the order of the calls, the last one
// repeats itself.
func (p *fakeProvider) sequence(method string, values []uint64) *fakeProvider {
	p.sequences[method] = values
	return p
}

// Call answers from the table.
func (p *fakeProvider) Call(class, method, instancePath string, args map[string]wmi.Value) (map[string]wmi.Value, error) {
	call := fakeCall{Class: class, Method: method, Path: instancePath}
	if args != nil {
		if value, ok := args["gmInput"]; ok {
			call.Input = value.UInt64()
			if value.Kind == wmi.KindBytes {
				call.Bytes = value.Bytes
			}
		}
		if value, ok := args["uiInput"]; ok {
			call.Input = value.UInt64()
		}
	}
	p.calls = append(p.calls, call)

	if p.err != nil {
		return nil, p.err
	}

	// A method that is called several times answers with the values of
	// the sequence, one after the other.
	if values, ok := p.sequences[method]; ok && len(values) > 0 {
		index := p.counts[method]
		p.counts[method]++
		if index >= len(values) {
			index = len(values) - 1
		}
		return map[string]wmi.Value{
			"gmOutput": wmi.Uint64(values[index]),
			"uiOutput": wmi.Uint64(values[index]),
		}, nil
	}

	outputs, ok := p.outputs[method]
	if !ok {
		// A method that writes a setting answers with nothing, the
		// provider confirms the write by echoing the value.
		if isWriteMethod(method) {
			return map[string]wmi.Value{}, nil
		}
		return nil, fmt.Errorf("hardware: the fake provider does not know %s", method)
	}
	return outputs, nil
}

// isWriteMethod reports whether a method only writes a setting.
//
// The provider answers a write with the new value rather than with a
// status, so a write that is not in the table is confirmed with an empty
// answer.
func isWriteMethod(method string) bool {
	switch method {
	case "SetGamingFanBehavior",
		"SetGamingFanSpeed",
		"SetGamingFanTable",
		"SetGamingLEDColor",
		"SetGamingLEDBehavior",
		"SetGamingKBBacklight",
		"SetGamingRgbKb",
		"SetGamingMiscSetting",
		"SetGamingProfile",
		"SetFunction":
		return true
	}
	return false
}

// Instances answers with the paths of the two classes.
func (p *fakeProvider) Instances(class string) ([]string, error) {
	if p.err != nil {
		return nil, p.err
	}
	return []string{class + "\\instance"}, nil
}

// Close does nothing.
func (p *fakeProvider) Close() {}

// callOf returns the first call of a method.
func (p *fakeProvider) callOf(method string) (fakeCall, bool) {
	for _, call := range p.calls {
		if call.Method == method {
			return call, true
		}
	}
	return fakeCall{}, false
}

// callsOf returns every call of a method.
func (p *fakeProvider) callsOf(method string) []fakeCall {
	var out []fakeCall
	for _, call := range p.calls {
		if call.Method == method {
			out = append(out, call)
		}
	}
	return out
}

// countOf returns the number of calls of a method.
func (p *fakeProvider) countOf(method string) int {
	return len(p.callsOf(method))
}

// newFakeBackend builds a backend that answers from a table.
func newFakeBackend(provider *fakeProvider) *Backend {
	return NewBackendWith(provider, "gaming\\instance", "action\\instance")
}

// newBackendWith builds a backend on top of the given provider.
func newBackendWith(provider Provider) *Backend {
	return NewBackendWith(provider, "gaming\\instance", "action\\instance")
}
