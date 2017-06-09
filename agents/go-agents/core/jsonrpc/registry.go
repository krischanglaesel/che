package jsonrpc

import "sync"

// DefaultRegistry is package registry, which is used by NewManagedTunnel.
var DefaultRegistry = NewRegistry()

// Save saves a given tunnel is the default registry.
func Save(tun *Tunnel) {
	DefaultRegistry.Save(tun)
}

// Rm removes a given tunnel from the default registry.
func Rm(id string) (*Tunnel, bool) {
	return DefaultRegistry.Rm(id)
}

// GetTunnels gets tunnels managed by default registry.
func GetTunnels() []*Tunnel {
	return DefaultRegistry.GetTunnels()
}

// Get gets a single tunnel managed by default registry.
func Get(id string) (*Tunnel, bool) {
	return DefaultRegistry.Get(id)
}

// TunnelRegistry is a simple storage for tunnels.
type TunnelRegistry struct {
	sync.RWMutex
	tuns map[string]*Tunnel
}

// NewRegistry creates a new registry.
func NewRegistry() *TunnelRegistry {
	return &TunnelRegistry{tuns: make(map[string]*Tunnel)}
}

// Save saves a tunnel with a given id in this registry, overrides existing one.
func (cm *TunnelRegistry) Save(tun *Tunnel) {
	cm.Lock()
	defer cm.Unlock()
	cm.tuns[tun.id] = tun
}

// Rm removes tunnel with given id from the registry.
func (cm *TunnelRegistry) Rm(id string) (*Tunnel, bool) {
	cm.Lock()
	defer cm.Unlock()
	tun, ok := cm.tuns[id]
	if ok {
		delete(cm.tuns, id)
	}
	return tun, ok
}

// GetTunnels returns all the tunnels which the registry keeps.
func (cm *TunnelRegistry) GetTunnels() []*Tunnel {
	cm.RLock()
	defer cm.RUnlock()
	tuns := make([]*Tunnel, 0)
	for _, v := range cm.tuns {
		tuns = append(tuns, v)
	}
	return tuns
}

// Get returns tunnel with a given id.
func (cm *TunnelRegistry) Get(id string) (*Tunnel, bool) {
	cm.Lock()
	defer cm.Unlock()
	tun, ok := cm.tuns[id]
	return tun, ok
}
