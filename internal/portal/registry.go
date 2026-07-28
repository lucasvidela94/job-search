package portal

import "fmt"

var portals = make(map[string]Portal)

// Register registers a portal by name.
// Panics if a portal with the same name is already registered.
func Register(p Portal) {
	name := p.Name()
	if _, ok := portals[name]; ok {
		panic(fmt.Sprintf("portal already registered: %s", name))
	}
	portals[name] = p
}

// Get returns a portal by name.
func Get(name string) (Portal, bool) {
	p, ok := portals[name]
	return p, ok
}

// All returns all registered portals.
func All() map[string]Portal {
	// Return a copy
	cp := make(map[string]Portal, len(portals))
	for k, v := range portals {
		cp[k] = v
	}
	return cp
}

// Names returns all registered portal names.
func Names() []string {
	names := make([]string, 0, len(portals))
	for k := range portals {
		names = append(names, k)
	}
	return names
}
