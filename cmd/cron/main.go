package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"xinyuan_tech/subscription-service/internal/conf"

	"github.com/go-kratos/kratos/v2/config"
	"github.com/go-kratos/kratos/v2/config/file"
	"github.com/robfig/cron/v3"
	_ "go.uber.org/automaxprocs"
)

var (
	flagconf string
)

func init() {
	flag.StringVar(&flagconf, "conf", "configs/config.yaml", "config path, eg: -conf config.yaml")
}

func main() {
	flag.Parse()

	// 初始化配置
	c := config.New(
		config.WithSource(
			file.NewSource(flagconf),
		),
	)
	defer c.Close()

	if err := c.Load(); err != nil {
		panic(err)
	}

	var bc conf.Bootstrap
	if err := c.Scan(&bc); err != nil {
		panic(err)
	}

	// 初始化应用
	app, cleanup, err := wireApp(&bc)
	if err != nil {
		panic(err)
	}
	defer cleanup()

	// 创建定时任务调度器（支持秒级调度）
	cronScheduler := cron.New(cron.WithSeconds())

	// 1. 订阅过期检查 - 每天凌晨 2 点执行
	_, err = cronScheduler.AddFunc("0 0 2 * * *", func() {
		log.Println("[CRON] Starting subscription expiration check...")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()

		count, uids, err := app.subscriptionUsecase.UpdateExpiredSubscriptions(ctx)
		if err != nil {
			log.Printf("[CRON] Error updating expired subscriptions: %v", err)
		} else {
			log.Printf("[CRON] Updated %d expired subscriptions: %v", count, uids)
			log.Println("[CRON] Finished subscription expiration check")
		}
	})
	if err != nil {
		log.Printf("Failed to add expiration check job: %v", err)
	}

	// 2. 续费提醒 - 每天上午 10 点执行
	_, err = cronScheduler.AddFunc("0 0 10 * * *", func() {
		log.Println("[CRON] Starting renewal reminder check...")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()

		subscriptions, total, err := app.subscriptionUsecase.GetExpiringSubscriptions(ctx, 7, 1, 100)
		if err != nil {
			log.Printf("[CRON] Error getting expiring subscriptions: %v", err)
			return
		}

		log.Printf("[CRON] Found %d subscriptions expiring within 7 days", total)
		for _, sub := range subscriptions {
			// TODO: 发送续费提醒通知
			log.Printf("[CRON] Reminder: User %d subscription (plan: %s) expires at %s", 
				sub.UserID, sub.PlanID, sub.EndTime.Format("2006-01-02 15:04:05"))
		}
		log.Println("[CRON] Finished renewal reminder check")
	})
	if err != nil {
		log.Printf("Failed to add renewal reminder job: %v", err)
	}

	// 3. 自动续费处理 - 每天凌晨 3 点执行
	_, err = cronScheduler.AddFunc("0 0 3 * * *", func() {
		log.Println("[CRON] Starting auto-renewal process...")
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()

		totalCount, successCount, failedCount, results, err := app.subscriptionUsecase.ProcessAutoRenewals(ctx, 3, false)
		if err != nil {
			log.Printf("[CRON] Error processing auto-renewals: %v", err)
		} else {
			log.Printf("[CRON] Auto-renewal completed: total=%d, success=%d, failed=%d",
				totalCount, successCount, failedCount)
			
			// 记录详细结果
			for _, result := range results {
				if result.Success {
					log.Printf("[CRON] Auto-renewal success: user=%d, plan=%s, order=%s",
						result.UID, result.PlanID, result.OrderID)
				} else {
					log.Printf("[CRON] Auto-renewal failed: user=%d, plan=%s, error=%s",
						result.UID, result.PlanID, result.ErrorMessage)
				}
			}
		}
		log.Println("[CRON] Finished auto-renewal process")
	})
	if err != nil {
		log.Printf("Failed to add auto-renewal job: %v", err)
	}

	// 启动定时任务
	cronScheduler.Start()
	log.Println("========================================")
	log.Println("Cron jobs started successfully")
	log.Println("Scheduled jobs:")
	log.Println("  - Expiration check:  Every day at 02:00")
	log.Println("  - Renewal reminder:  Every day at 10:00")
	log.Println("  - Auto-renewal:      Every day at 03:00")
	log.Println("========================================")

	// 优雅退出
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down gracefully...")

	// 停止定时任务
	ctx := cronScheduler.Stop()
	select {
	case <-ctx.Done():
		log.Println("Cron jobs stopped gracefully")
	case <-time.After(5 * time.Second):
		log.Println("Cron jobs forced to stop after timeout")
	}
}

