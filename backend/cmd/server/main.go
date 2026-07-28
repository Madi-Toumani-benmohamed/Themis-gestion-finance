package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"themis/internal/api"
	"themis/internal/config"
	"themis/internal/db"
	"themis/internal/llm"
	"themis/internal/parser"
)

func main() {
	cfg := config.Load()

	database, err := db.Open(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("connexion base: %v", err)
	}
	if err := db.Migrate(database); err != nil {
		log.Fatalf("migrations: %v", err)
	}
	log.Println("migrations appliquées")

	srv := &api.Server{DB: database, Cfg: cfg}
	srv.BackfillDisplayLabels() // noms d'affichage : rattrapage de l'existant
	mux := http.NewServeMux()
	mux.Handle("/api/", srv.Router())

	// Sert le build du frontend si présent (image Docker unique possible)
	if _, err := os.Stat("./static/index.html"); err == nil {
		fs := http.FileServer(http.Dir("./static"))
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			if _, err := os.Stat("./static" + r.URL.Path); err != nil && r.URL.Path != "/" {
				http.ServeFile(w, r, "./static/index.html") // fallback SPA
				return
			}
			fs.ServeHTTP(w, r)
		})
	}

	httpSrv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       2 * time.Minute, // large : un import PDF peut être gros
		WriteTimeout:      3 * time.Minute, // large : l'OCR d'un relevé prend du temps
		IdleTimeout:       2 * time.Minute,
	}

	go func() {
		if ok, detail := parser.OCRStatus(); ok {
		log.Printf("OCR opérationnel (pdftoppm + tesseract fra)")
	} else {
		log.Printf("ATTENTION — OCR indisponible : %s ; les relevés sans couche texte (Crédit Agricole…) nécessiteront ANTHROPIC_API_KEY", detail)
	}
	srv.InitAI() // env, ou config admin persistée en base si présente
	srv.StartSessionCleanup()
	if o, ok := srv.GetLLMForCheck().(*llm.Ollama); ok {
		if err := o.Ping(context.Background()); err != nil {
			log.Printf("ATTENTION — %v", err)
		}
	}
	log.Printf("thémis démarre sur :%s", cfg.Port)
		if err := httpSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("serveur: %v", err)
		}
	}()

	// Arrêt propre (SIGTERM envoyé par Kubernetes)
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	log.Println("arrêt en cours…")
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	_ = httpSrv.Shutdown(ctx)
}
