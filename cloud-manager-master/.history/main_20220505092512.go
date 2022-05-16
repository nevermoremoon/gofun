package main

import (
	"cloud-manager/app/config"
	"cloud-manager/app/route"
	"context"
	"flag"
	"fmt"
	"github.com/gin-gonic/gin"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"
)

func init() {
	//log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	cfg := flag.String("c", "config.yml", "configuration file")
	version := flag.Bool("v", false, "show version")
	flag.Parse()

	handleVersion(*version)
	handleConfig(*cfg)

}

func handleVersion(displayVersion bool) {
	if displayVersion {
		fmt.Println(config.Version)
		os.Exit(0)
	}
}

func handleConfig(configFile string) {
	err := config.Parse(configFile)
	if err != nil {
		log.Fatalln(err)
	}
}


func main() {
	gin.SetMode(config.AppMode)

	engine := gin.Default()
	// 性能分析 - 正式环境不要使用！！！
	//pprof.Register(engine)

	// 设置路由
	route.SetupRouter(engine)

	server := &http.Server{
		Addr         : config.AppPort,
		Handler      : engine,
		ReadTimeout  : config.AppReadTimeout * time.Second,
		WriteTimeout : config.AppWriteTimeout * time.Second,
	}

	fmt.Println("|-----------------------------------|")
	fmt.Println("|           cloud-manager           |")
	fmt.Println("|-----------------------------------|")
	fmt.Println("|  Go Http Server Start Successful  |")
	fmt.Println("|    Port" + config.AppPort + "     Pid:" + fmt.Sprintf("%d", os.Getpid()) + " |")
	fmt.Println("|-----------------------------------|")
	fmt.Println("")
/*
	logo := `
 ███║██║   ██║██║  ██║█████╗██╔████╔██║███████║██╔██╗ ██║███████║██║  ███╗█████╗  ██████╔╝
██║     ██║     ██║   ██║██║   ██║██║  ██║╚════╝██║╚██╔╝██║██╔══██║██║╚██╗██║██╔══██║██║   ██║██╔══╝  ██╔══██╗
╚██████╗███████╗╚██████╔╝╚██████╗██╗      ██████╗ ██╗   ██╗██████╗       ███╗   ███╗ █████╗ ███╗   ██╗ █████╗  ██████╗ ███████╗██████╗
██╔════╝██║     ██╔═══██╗██║   ██║██╔══██╗      ████╗ ████║██╔══██╗████╗  ██║██╔══██╗██╔════╝ ██╔════╝██╔══██╗
██║     ██║     ██║   █████╔╝██████╔╝      ██║ ╚═╝ ██║██║  ██║██║ ╚████║██║  ██║╚██████╔╝███████╗██║  ██║
 ╚═════╝╚══════╝ ╚═════╝  ╚═════╝ ╚═════╝       ╚═╝     ╚═╝╚═╝  ╚═╝╚═╝  ╚═══╝╚═╝  ╚═╝ ╚═════╝ ╚══════╝╚═╝  ╚═╝

`
	fmt.Println(logo)
*/
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server listen: %s\n", err)
		}
	}()

	// 等待中断信号以优雅地关闭服务器（设置 5 秒的超时时间）
	signalChan := make(chan os.Signal)
	signal.Notify(signalChan, os.Interrupt)
	sig := <-signalChan
	log.Println("Get Signal:", sig)
	log.Println("Shutdown Server ...")

	ctx, cancel := context.WithTimeout(context.Background(), 5 * time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Fatal("Server Shutdown:", err)
	}
	log.Println("Server exiting")
}
