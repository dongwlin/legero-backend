package handler

import (
	"bufio"
	"fmt"
	"time"

	"github.com/dongwlin/legero-backend/internal/pkg/broker"
	"github.com/gofiber/fiber/v2"
	"github.com/valyala/fasthttp"
)

type SSE struct {
	broker *broker.Broker
}

func NewSSE(broker *broker.Broker) *SSE {
	return &SSE{
		broker: broker,
	}
}

func (h *SSE) RegisterRoutes(r fiber.Router) {
	r.Get("/sse", h.SSE)
}

func (h *SSE) SSE(c *fiber.Ctx) error {

	c.Set("Content-Type", "text/event-stream")
	c.Set("Cache-Control", "no-cache")
	c.Set("Connection", "keep-alive")
	c.Set("Transfer-Encoding", "chunked")

	c.Context().SetBodyStreamWriter(fasthttp.StreamWriter(func(w *bufio.Writer) {

		clientChan := h.broker.AddClient()
		defer h.broker.RemoveClient(clientChan)

		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()

		if _, err := fmt.Fprintf(w, "data: \n\n"); err != nil {
			return
		}
		if err := w.Flush(); err != nil {
			return
		}

		for {
			select {
			case <-ticker.C:
				if _, err := fmt.Fprintf(w, "data: \n\n"); err != nil {
					return
				}
				if err := w.Flush(); err != nil {
					return
				}

			case msg, ok := <-clientChan:
				if !ok {
					return
				}
				if _, err := fmt.Fprintf(w, "data: %s\n\n", msg); err != nil {
					return
				}
				if err := w.Flush(); err != nil {
					return
				}

			}
		}
	}))

	return nil
}
