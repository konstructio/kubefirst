package pkg

import (
	"context"
	"fmt"
	"github.com/rs/zerolog/log"
	"io"
	"net"

	"github.com/ngrok/ngrok-go"
	"github.com/ngrok/ngrok-go/config"
	"github.com/spf13/viper"
	"golang.org/x/sync/errgroup"
)

func RunNgrok(ctx context.Context, dest string) {
	tunnel, err := ngrok.StartTunnel(ctx, config.HTTPEndpoint(), ngrok.WithAuthtokenFromEnv())
	if err != nil {
		log.Error().Err(err).Msg("")
	}

	fmt.Println("tunnel created: ", tunnel.URL())
	viper.Set("github.atlantis.webhook.url", tunnel.URL()+"/events")
	viper.WriteConfig()

	for {
		conn, err := tunnel.Accept()
		if err != nil {
			log.Error().Err(err).Msg("")
		}

		log.Info().Msgf("accepted connection from %s", conn.RemoteAddr())

		go func() {
			err := handleConn(ctx, dest, conn)
			log.Info().Msgf("connection closed: %s", err)
		}()
	}
}

func handleConn(ctx context.Context, dest string, conn net.Conn) error {
	next, err := net.Dial("tcp", dest)
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
