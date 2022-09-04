package main

import "sync"

type GracefulShutdown interface {
	GracefulShutdown()
}

func shutdownGracefully(services ...GracefulShutdown) error {

	wg := sync.WaitGroup{}

	for _, service := range services {

			// Run Service
			go func() {
				err := runGrpc()
				if err != nil {
					return err
				}
			}()


			chSig := make(chan os.Signal, 1)
			signal.Notify(chSig, syscall.SIGINT, syscall.SIGTERM)
			<-chSig

			go func() {
				defer service.GracefulShutdown()
			}()

		}
	}

	ru

}
