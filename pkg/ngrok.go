package pkg

import (
	"context"
	"fmt"
	"github.com/spf13/viper"
	"golang.ngrok.com/ngrok"
	"golang.ngrok.com/ngrok/config"
	"io"
	"net"

	"github.com/rs/zerolog/log"
	_ "github.com/spf13/viper"
	"golang.org/x/sync/errgroup"
)

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

	fmt.Println("tunnel created: ", tunnel.URL())
	viper.Set("github.atlantis.webhook.url", tunnel.URL()+"/events")
	viper.Set("ngrok.url", tunnel.URL())
	err = viper.WriteConfig()
	if err != nil {
		log.Error().Err(err).Msg("")
		return
	}

	for {
		fmt.Println("---debug---")
		fmt.Println("current state of the ngrok context is:", ctx.Err())
		fmt.Println("---debug---")

		fmt.Println(tunnel.URL())
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
