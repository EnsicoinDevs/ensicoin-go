package peer

import (
	"github.com/EnsicoinDevs/ensicoincoin/network"
	"sync"
)

type PeersManager struct {
	lock sync.RWMutex

	peers []*Peer
}

func NewPeersManager() *PeersManager {
	return &PeersManager{}
}

func (pm *PeersManager) Broadcast(message network.Message) {
	pm.lock.RLock()
	defer pm.lock.RUnlock()

	for i := 0; i < len(pm.peers); i++ {
		if err := pm.peers[i].Send(message); err != nil {
			pm.peers = append(pm.peers[:i], pm.peers[i+1:]...)
			i++
		}
	}
}

func (pm *PeersManager) Add(peer *Peer) {
	pm.lock.Lock()
	defer pm.lock.Unlock()

	pm.peers = append(pm.peers, peer)
}
