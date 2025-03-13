package broker

import (
	"sync"

	"github.com/dongwlin/legero-backend/internal/config"
	"github.com/rs/zerolog/log"
)

type Broker struct {
	clients    map[chan []byte]struct{}
	mutex      sync.RWMutex
	bufferSize int
}

func NewBroker(conf *config.Config) *Broker {
	return &Broker{
		clients:    make(map[chan []byte]struct{}),
		bufferSize: 50, // 暂时先不使用config
	}
}

func (b *Broker) AddClient() chan []byte {

	client := make(chan []byte, b.bufferSize)
	b.mutex.Lock()
	b.clients[client] = struct{}{}
	b.mutex.Unlock()
	log.Info().
		Int("current clinet count", len(b.clients)).
		Msg("client added")

	return client
}

func (b *Broker) RemoveClient(client chan []byte) {

	b.mutex.Lock()
	defer b.mutex.Unlock()

	if _, exists := b.clients[client]; exists {
		delete(b.clients, client)
		close(client)
		log.Info().
			Int("current client count", len(b.clients)).
			Msg("client removed")
	}
}

func (b *Broker) Broadcast(message []byte) {
	b.mutex.RLock()
	defer b.mutex.RUnlock()

	for client := range b.clients {
		select {
		case client <- message:
		default:
			log.Warn().
				Int("current backlog", len(client)).
				Msg("client buffer is full, dropping message")
		}
	}
}
