package pkg

import (
	"context"
	"github.com/spf13/viper"
	"golang.ngrok.com/ngrok"
	"golang.ngrok.com/ngrok/config"
	"io"
	"net"

	"github.com/rs/zerolog/log"
	_ "github.com/spf13/viper"
	"golang.org/x/sync/errgroup"
)

// RunNgrok creates a ngrok tunnel, listens for incoming connections, and starts a goroutine to handle each connection
// with context passed to it, also it logs the errors and sets the url in viper. RunNgrok is called to run in goroutine
// the caller needs to cancel the context to stop the tunnel.
func RunNgrok(ctx context.Context) {
	defer func() {
		log.Info().Msg("RunNgrok context was cancelled, conn closed and not accepting new connections, and function exited")
	}()
	tunnel, err := ngrok.Listen(ctx, config.HTTPEndpoint(), ngrok.WithAuthtokenFromEnv())
	if err != nil {
		log.Error().Err(err).Msg("")
	}

	// inform when tunnel is not accepting new connections
	go func() {
		select {
		case <-ctx.Done():
			log.Info().Msg("Ngrok is closed, and not accepting new connections")
		}
	}()

	log.Info().Msgf("tunnel created: %s", tunnel.URL())
	viper.Set("github.atlantis.webhook.url", tunnel.URL()+"/events")
	viper.Set("ngrok.url", tunnel.URL())
	err = viper.WriteConfig()
	if err != nil {
		log.Error().Err(err).Msg("")
		return
	}

	for {
		log.Debug().Msgf("current state of the ngrok context is (if nil, its healthy): %s", ctx.Err())

		conn, err := tunnel.Accept()
		if err != nil {
			log.Error().Err(err).Msg("")
			break
		}

		if ctx.Err() == nil {
			log.Info().Msgf("Ngrok is accepting connections: %s", conn.RemoteAddr())
			go func() {
				err = handleConn(ctx, conn)
				if err == nil {
					return
				}
				log.Info().Err(err).Msg("connection closed: ")
			}()
		} else {
			err := conn.Close()
			if err != nil {
				println(err)
			}
			break
		}
	}
}

// handleConn handles the connection by copying the data from the connection to the destination address and vice versa
// it also logs the errors and closes the connection when the context is cancelled
func handleConn(ctx context.Context, conn net.Conn) error {
	next, err := net.Dial("tcp", ":80")
	if err != nil {
		return err
	}

	g, _ := errgroup.WithContext(ctx)

	g.Go(func() error {
		_, err := io.Copy(next, conn)
		return err
	})
	g.Go(func() error {
		_, err := io.Copy(conn, next)
		return err
	})

	return g.Wait()
}
