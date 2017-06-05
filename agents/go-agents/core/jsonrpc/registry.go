package jsonrpc

import "sync"

var DefaultRegistry = &ChannelRegistry{channels: make(map[string]Channel)}

func Save(channel Channel) {
	DefaultRegistry.Save(channel)
}

func Drop(id string) (Channel, bool) {
	return DefaultRegistry.Drop(id)
}

func GetChannels() []Channel {
	return DefaultRegistry.GetChannels()
}

func Get(id string) (Channel, bool) {
	return DefaultRegistry.Get(id)
}

type ChannelRegistry struct {
	sync.RWMutex
	channels map[string]Channel
}

func (cm *ChannelRegistry) Save(channel Channel) {
	cm.Lock()
	defer cm.Unlock()
	cm.channels[channel.ID] = channel
}

func (cm *ChannelRegistry) Drop(id string) (Channel, bool) {
	cm.Lock()
	defer cm.Unlock()
	channel, ok := cm.channels[id]
	if ok {
		delete(cm.channels, id)
	}
	return channel, ok
}

func (cm *ChannelRegistry) GetChannels() []Channel {
	cm.RLock()
	defer cm.RUnlock()
	channels := make([]Channel, 0)
	for _, v := range cm.channels {
		channels = append(channels, v)
	}
	return channels
}

func (cm *ChannelRegistry) Get(id string) (Channel, bool) {
	cm.Lock()
	defer cm.Unlock()
	channel, ok := cm.channels[id]
	return channel, ok
}
