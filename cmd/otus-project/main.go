
package main

import (
    "log"
    "os"

    "github.com/k-tatiana/otus-project/internal/server"
)

func main() {
    // PORT can be set by environment variable; defaults handled by server
    if os.Getenv("PORT") == "" {
        os.Setenv("PORT", "8080")
    }
    log.Println("Starting otus-project server")
    server.RunServer()
}
