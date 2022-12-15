package pkg

import (
	"context"
	"fmt"
	"io"
	"net"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/ngrok/ngrok-go"
	"github.com/ngrok/ngrok-go/config"
	"github.com/spf13/viper"
	"golang.org/x/sync/errgroup"
)

func RunNgrok(ctx context.Context) {

	tunnel, err := ngrok.StartTunnel(ctx, config.HTTPEndpoint(), ngrok.WithAuthtokenFromEnv())
	if err != nil {
		log.Error().Err(err).Msg("")
	}

	retry(3, time.Second, "create ngrok tunnel", func() error {
		tunnel, err = ngrok.StartTunnel(ctx, config.HTTPEndpoint(), ngrok.WithAuthtokenFromEnv())
		if err != nil {
			log.Debug().Err(err).Msg("")
			return err
		}
		return nil
	})

	fmt.Println("tunnel created: ", tunnel.URL())
	viper.Set("github.atlantis.webhook.url", tunnel.URL()+"/events")
	viper.Set("ngrok.url", tunnel.URL())
	viper.WriteConfig()

	for {
		conn, err := tunnel.Accept()
		if err != nil {
			log.Error().Err(err).Msg("")
		}

		log.Info().Msgf("accepted connection from %s", conn.RemoteAddr())

		go func() {

			err := handleConn(ctx, conn)
			log.Info().Msgf("connection closed: %v", err)
		}()
	}
}

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
