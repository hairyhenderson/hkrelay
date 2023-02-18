package main

import (
	"context"
	"flag"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/brutella/hc"
	"github.com/brutella/hc/accessory"
	hclog "github.com/brutella/hc/log"

	"github.com/rs/zerolog"

	"periph.io/x/conn/v3/gpio"
	"periph.io/x/conn/v3/gpio/gpioreg"
	"periph.io/x/host/v3"
)

func main() {
	var (
		storagePath string
		setupCode   string
		accName     string
		addr        string
		debug       bool
		pinName     string
	)
	const (
		defaultPath = ""
	)

	flag.StringVar(&storagePath, "p", defaultPath, "storage path for HomeControl data")
	flag.StringVar(&addr, "addr", "", "address to listen to")
	flag.StringVar(&pinName, "pin", "P1_7", "GPIO pin name")
	flag.StringVar(&setupCode, "code", "12344321", "setup code")
	flag.StringVar(&accName, "name", "HomeKit Light", "accessory name")
	flag.BoolVar(&debug, "debug", false, "Enable debug logging")

	flag.Parse()

	parts := strings.SplitN(addr, ":", 2)
	ip := parts[0]
	port := ""
	if len(parts) == 2 {
		port = parts[1]
	}

	ctx, mainCancel := context.WithCancel(context.Background())
	defer mainCancel()

	ctx, log := initLogger(ctx)
	if debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
		hclog.Debug.Enable()
	}

	initMetrics()
	go func() {
		mux := http.NewServeMux()
		mux.Handle("/metrics", promhttp.Handler())
		if err := http.ListenAndServe(":8080", mux); err != nil {
			log.Error().Err(err).Send()
		}
	}()

	_, err := host.Init()
	if err != nil {
		log.Fatal().Err(err).Send()
	}

	relay := gpioreg.ByName(pinName)
	if relay == nil {
		log.Fatal().Str("pin", pinName).Msg("no pin found for name")
	}
	level := relay.Read()
	log.Info().Str("pin", pinName).Bool("level", bool(level)).Msg("read relay pin level")

	info := accessory.Info{
		Name:         accName,
		SerialNumber: "0123456789",
		Model:        "a",
		// FirmwareRevision:
		Manufacturer: "Dave Henderson",
	}

	acc := accessory.NewOutlet(info)

	initResponders(ctx, acc, relay)

	t, err := hc.NewIPTransport(hc.Config{
		Pin:         setupCode,
		StoragePath: storagePath,
		IP:          ip,
		Port:        port,
	}, acc.Accessory)
	if err != nil {
		log.Fatal().Err(err).Send()
	}

	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt)
	signal.Notify(c, syscall.SIGTERM)

	go func(ctx context.Context) {
		select {
		case sig := <-c:
			zerolog.Ctx(ctx).Error().Stringer("signal", sig).Msg("terminating due to signal")
			<-t.Stop()
		case <-ctx.Done():
			zerolog.Ctx(ctx).Error().Err(ctx.Err()).Msg("context done")
			<-t.Stop()
		}
	}(ctx)

	log.Info().Msgf("starting up '%s'. setup code is %s", accName, setupCode)
	t.Start()
}

func initResponders(ctx context.Context, acc *accessory.Outlet, relay gpio.PinIO) {
	lb := acc.Outlet
	log := zerolog.Ctx(ctx)

	lb.On.OnValueRemoteGet(func() bool {
		start := time.Now()
		log.Debug().Msg("lb.On.OnValueRemoteGet()")
		isOn := bool(relay.Read())
		observeUpdateDuration("on", "remoteGet", start)
		return isOn
	})

	lb.On.OnValueRemoteUpdate(func(on bool) {
		start := time.Now()
		log.Debug().Bool("on", on).Msg("lb.On.OnValueRemoteUpdate")
		var err error
		if on {
			err = relay.Out(gpio.High)
		} else {
			err = relay.Out(gpio.Low)
		}
		if err != nil {
			log.Error().Err(err).Bool("on", on).Msg("error during lb.On.OnValueRemoteUpdate")
		}
		lb.On.SetValue(on)
		observeUpdateDuration("on", "remoteUpdate", start)
	})

	acc.OnIdentify(func() {
		start := time.Now()
		log.Debug().Msg("acc.OnIdentify()")

		level := relay.Read()
		for i := 0; i < 4; i++ {
			level = !level
			err := relay.Out(level)
			if err != nil {
				log.Error().Err(err).Stringer("level", level).Msg("error during acc.OnIdentify")
			}
			time.Sleep(500 * time.Millisecond)
		}

		observeUpdateDuration("acc", "identify", start)
	})
}
