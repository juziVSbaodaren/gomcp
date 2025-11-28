package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"mcptool/mcpclient"
	"sync"
	"time"
)

type GeocodeResult struct {
	Address string  `json:"address"`
	Lat     float64 `json:"lat"`
	Lng     float64 `json:"lng"`
	City    string  `json:"city"`
}

func main() {
	ctx := context.Background()

	// 用接口统一声明客户端

	// 你可以切换成 HTTP / WS / SSE
	// client := mcpclient.NewUnifiedClientHTTP("http://localhost:8074/mcp")
	client, err := mcpclient.NewUnifiedClientWS("ws://localhost:8074/ws")
	if err != nil {
		log.Fatalln("Error:", err)
	}
	defer client.Close()

	if out, err := client.ServerToolsList(ctx); err != nil {
		log.Println("Error:", err)
	} else {
		fmt.Println("Server tools list:", out)
	}

	if out, err := client.ServerInfo(ctx); err != nil {
		log.Println("Error:", err)
	} else {
		fmt.Println("Server Info:", out)
	}

	// 调用 geocode 工具
	var geoRes GeocodeResult
	err = client.CallTool(ctx, "geocode", map[string]interface{}{"address": "天安门"}, &geoRes)
	if err != nil {
		log.Println("Error:", err)
	} else {
		fmt.Println("Geocode Result:", geoRes)
	}

	var wg sync.WaitGroup
	wg.Add(1)
	// 如果是 SSE 客户端，可以订阅事件
	go func() {
		defer wg.Done()
		if err := client.WatchEvents(func(event string, data json.RawMessage) {
			fmt.Println("SSE Event:", event, "Data:", string(data))
		}); err != nil {
			log.Println("SSE Error (or unsupported):", err)
		}
	}()
	// 保持主线程运行，观察 SSE
	wg.Wait()
	time.Sleep(5 * time.Second)
	log.Println("Done")
}
