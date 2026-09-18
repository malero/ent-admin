package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"time"

	entadminexample "github.com/malero/ent-admin/example"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:8080", "HTTP listen address")
	dataSourceName := flag.String("db", "file:ent-admin.db?_fk=1", "SQLite data source name")
	basePath := flag.String("base-path", "/control", "admin mount path")
	flag.Parse()

	ctx := context.Background()
	client, err := entadminexample.Open(ctx, *dataSourceName)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()
	if err := entadminexample.Seed(ctx, client); err != nil {
		log.Fatal(err)
	}

	admin, err := entadminexample.NewHandler(client, *basePath, requestLogMiddleware)
	if err != nil {
		log.Fatal(err)
	}
	mux := http.NewServeMux()
	mux.Handle(entadminexample.MountPath(admin), admin)

	server := &http.Server{
		Addr:              *addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	log.Printf("Ent Admin example listening at http://%s%s", *addr, admin.BasePath())
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}

func requestLogMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s (%s)", r.Method, r.URL.Path, time.Since(started).Round(time.Millisecond))
	})
}
