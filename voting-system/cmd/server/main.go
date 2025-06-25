package main

import (
    "fmt"
    "log"
    "net/http"
    
    "voting-system/pkg/config"
    "github.com/gin-gonic/gin"
)

func main() {
    cfg, err := config.LoadConfig("configs/server.yaml")
    if err != nil {
        log.Fatal("Failed to load config:", err)
    }
    
    r := gin.Default()
    r.GET("/", func(c *gin.Context) {
        c.JSON(200, gin.H{
            "message": "Central Voting Server is running",
            "status":  "active",
        })
    })
    
    addr := fmt.Sprintf("%s:%s", cfg.Server.Host, cfg.Server.Port)
    fmt.Printf("Starting central server on %s\n", addr)
    log.Fatal(http.ListenAndServe(addr, r))
}
