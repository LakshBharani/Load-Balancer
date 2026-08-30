package main

import (
	"flag"
	"log"
	"os"
	"time"

	"github.com/fsnotify/fsnotify"
	"gopkg.in/yaml.v3"

	"github.com/LakshBharani/Load-Balancer/internal/backend"
	appconfig "github.com/LakshBharani/Load-Balancer/internal/config"
	"github.com/LakshBharani/Load-Balancer/internal/router"
)

func main() {
	configPath := flag.String("c", "config.yaml", "path to config file")
	flag.StringVar(configPath, "config", "config.yaml", "path to config file")
	flag.Parse()

	if _, err := os.Stat(*configPath); err != nil {
		log.Fatalf("config file not found or not accessible: %v", err)
	}

	mgr := router.NewManager()
	state := &appState{}

	log.Printf("reading config from %s", *configPath)
	if err := reload(*configPath, mgr, state); err != nil {
		log.Printf("config file loading failed: %v", err)
	}

	watchConfig(*configPath, mgr, state)
}

// appState tracks what's already running across reloads so we only
// rebind the health listener when its address actually changes.
type appState struct {
	healthServer     *backend.HealthServer
	healthServerAddr string
}

func reload(path string, mgr *router.Manager, state *appState) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	var cfg appconfig.AppConfig
	if err := yaml.NewDecoder(f).Decode(&cfg); err != nil {
		log.Printf("error parsing config: %v", err)
		return nil
	}
	cfg.Normalize()

	log.Printf("loaded config, with %d backends, %d rules", len(cfg.Backends), len(cfg.Rules))

	result, err := appconfig.Build(&cfg)
	if err != nil {
		log.Printf("config has logical errors: %v", err)
		return nil
	}

	switch {
	case state.healthServer == nil:
		state.healthServer = backend.NewHealthServer(result.Healths)
		state.healthServerAddr = cfg.HealthcheckAddr
		addr := cfg.HealthcheckAddr
		hs := state.healthServer
		go func() {
			if err := hs.Serve(addr); err != nil {
				log.Printf("health check listener failed: %v", err)
			}
		}()
	case state.healthServerAddr != cfg.HealthcheckAddr:
		log.Printf("warning: healthcheck_addr changed from %s to %s; restart the process to rebind it", state.healthServerAddr, cfg.HealthcheckAddr)
		state.healthServer.SetHealths(result.Healths)
	default:
		state.healthServer.SetHealths(result.Healths)
	}

	mgr.Reconcile(result.Listeners)
	log.Println("reload complete")
	return nil
}

// watchConfig watches configPath for writes and reloads on change,
// debouncing bursts of events some editors fire on a single save.
func watchConfig(configPath string, mgr *router.Manager, state *appState) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatalf("failed to create config watcher: %v", err)
	}
	defer watcher.Close()

	if err := watcher.Add(configPath); err != nil {
		log.Fatalf("failed to watch %s: %v", configPath, err)
	}
	log.Printf("watching for changes to %s", configPath)

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			if event.Op&(fsnotify.Write|fsnotify.Create) == 0 {
				continue
			}

			drain(watcher)
			time.Sleep(50 * time.Millisecond)
			drain(watcher)

			if err := reload(configPath, mgr, state); err != nil {
				log.Printf("loading config failed: %v", err)
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			log.Printf("config watcher error: %v", err)
		}
	}
}

func drain(watcher *fsnotify.Watcher) {
	for {
		select {
		case <-watcher.Events:
		default:
			return
		}
	}
}
