// Command migrator serves the rater → Helix migration UI and API, and cleans up
// what it wrote.
//
//	migrator serve   [-addr 127.0.0.1:8080] [-env-file .env] [-ledger data/ledger.jsonl] [-web-dir web/dist]
//	migrator cleanup [-env-file .env] [-ledger data/ledger.jsonl] -yes
package main

import (
	"context"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/ankitkumarflarre/datamigration/internal/api"
	"github.com/ankitkumarflarre/datamigration/internal/execute"
	"github.com/ankitkumarflarre/datamigration/internal/helix"
	"github.com/ankitkumarflarre/datamigration/internal/ledger"
	"github.com/ankitkumarflarre/datamigration/internal/schema"
	"github.com/ankitkumarflarre/datamigration/web"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: migrator serve|cleanup [flags]")
		os.Exit(2)
	}
	switch os.Args[1] {
	case "serve":
		serve(os.Args[2:])
	case "cleanup":
		cleanup(os.Args[2:])
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q (serve, cleanup)\n", os.Args[1])
		os.Exit(2)
	}
}

type common struct {
	envFile, ledgerPath, helixURL string
}

func (c *common) register(fs *flag.FlagSet) {
	fs.StringVar(&c.envFile, "env-file", envOr("MIGRATOR_ENV_FILE", ".env"), "file with HARNESS_URL and HARNESS_WRITER_KEY")
	fs.StringVar(&c.ledgerPath, "ledger", envOr("MIGRATOR_LEDGER", "data/ledger.jsonl"), "ledger of created records")
	fs.StringVar(&c.helixURL, "helix-url", "", "Helix URL (overrides HELIX_URL / HARNESS_URL)")
}

func envOr(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}

func (c *common) open() (*helix.Client, *ledger.Ledger) {
	h, err := helix.New(helix.Config{URL: c.helixURL, EnvFile: c.envFile})
	if err != nil {
		log.Fatal(err)
	}
	l, err := ledger.Open(c.ledgerPath)
	if err != nil {
		log.Fatal(err)
	}
	return h, l
}

func serve(args []string) {
	fset := flag.NewFlagSet("serve", flag.ExitOnError)
	var c common
	c.register(fset)
	addr := fset.String("addr", "127.0.0.1:8080", "listen address (local only by default, D11)")
	webDir := fset.String("web-dir", "", "serve the UI from this directory instead of the embedded build")
	_ = fset.Parse(args)
	h, l := c.open()
	defer l.Close()
	var ui fs.FS = web.Dist()
	if *webDir != "" {
		ui = os.DirFS(*webDir)
	}
	srv := api.NewServer(h, h, schema.New(h), l, ui)
	ln, err := net.Listen("tcp", *addr)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("migrator: http://%s  (Helix %s, ledger %s, %d records)", ln.Addr(), h.Base(), c.ledgerPath, len(l.All()))
	hs := &http.Server{Handler: srv.Handler(), ReadHeaderTimeout: 10 * time.Second}
	go func() {
		stop := make(chan os.Signal, 1)
		signal.Notify(stop, os.Interrupt)
		<-stop
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = hs.Shutdown(ctx)
	}()
	if err := hs.Serve(ln); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}

func cleanup(args []string) {
	fset := flag.NewFlagSet("cleanup", flag.ExitOnError)
	var c common
	c.register(fset)
	yes := fset.Bool("yes", false, "really delete every record in the ledger from Helix")
	_ = fset.Parse(args)
	h, l := c.open()
	defer l.Close()
	entries := l.All()
	if !*yes {
		fmt.Printf("%d records created by the migrator are in %s on %s.\nRun again with -yes to delete them from Helix.\n", len(entries), c.ledgerPath, h.Base())
		return
	}
	deleted, failed := execute.Cleanup(context.Background(), h, l, func(s string) { fmt.Println(s) })
	fmt.Printf("cleanup: %d deleted, %d failed\n", deleted, failed)
	if failed > 0 {
		os.Exit(1)
	}
}
