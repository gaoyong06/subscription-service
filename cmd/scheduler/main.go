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

	pkgutils "github.com/gaoyong06/go-pkg/utils"
	"github.com/go-kratos/kratos/v2/config"
	"github.com/go-kratos/kratos/v2/config/file"
	"github.com/robfig/cron/v3"
	_ "go.uber.org/automaxprocs"
)

var (
	flagconf string
	runMode  string
)

func init() {
	flag.StringVar(&flagconf, "conf", "", "config path, eg: -conf config.yaml (deprecated, use -mode instead)")
	flag.StringVar(&runMode, "mode", "debug", "Run mode (debug, release)")
}

func main() {
	flag.Parse()

	// 根据 mode 自动选择配置文件
	configPath := flagconf
	if configPath == "" {
		// 使用 go-pkg/utils 中的通用配置文件路径解析函数
		// 支持从不同目录运行（项目根目录、cmd/scheduler 目录等）
		configPath = pkgutils.FindConfigFileWithMode(runMode, []string{
			"configs",       // 从项目根目录运行
			"../../configs", // 从 cmd/scheduler 目录运行
			"../configs",    // 从 cmd 目录运行
		})
	}

	// 初始化配置
	c := config.New(
		config.WithSource(
			file.NewSource(configPath),
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

	// 从配置中获取参数
	// 订阅业务配置
	autoRenewDaysBefore := 3  // 默认值
	expiryCheckDays := 7       // 默认值
	if bc.GetSubscription() != nil {
		subConf := bc.GetSubscription()
		if subConf.GetAutoRenewDaysBefore() > 0 {
			autoRenewDaysBefore = int(subConf.GetAutoRenewDaysBefore())
		}
		if subConf.GetExpiryCheckDays() > 0 {
			expiryCheckDays = int(subConf.GetExpiryCheckDays())
		}
	}

	// Scheduler 调度配置
	cronExpiryCheck := "0 0 2 * * *"      // 默认: 每天凌晨 2 点
	cronRenewalReminder := "0 0 10 * * *" // 默认: 每天上午 10 点
	cronAutoRenewal := "0 0 3 * * *"      // 默认: 每天凌晨 3 点

	if bc.GetScheduler() != nil {
		schedulerConf := bc.GetScheduler()
		if schedulerConf.GetExpiryCheck() != nil {
			expiryCheckTask := schedulerConf.GetExpiryCheck()
			if expiryCheckTask.GetCron() != "" {
				cronExpiryCheck = expiryCheckTask.GetCron()
			}
		}
		if schedulerConf.GetRenewalReminder() != nil {
			renewalReminderTask := schedulerConf.GetRenewalReminder()
			if renewalReminderTask.GetCron() != "" {
				cronRenewalReminder = renewalReminderTask.GetCron()
			}
		}
		if schedulerConf.GetAutoRenewal() != nil {
			autoRenewalTask := schedulerConf.GetAutoRenewal()
			if autoRenewalTask.GetCron() != "" {
				cronAutoRenewal = autoRenewalTask.GetCron()
			}
		}
	}

	// 创建定时任务调度器（支持秒级调度）
	cronScheduler := cron.New(cron.WithSeconds())

	// 1. 订阅过期检查任务
	if bc.GetScheduler() != nil && bc.GetScheduler().GetExpiryCheck() != nil {
		expiryCheckTask := bc.GetScheduler().GetExpiryCheck()
		if expiryCheckTask.GetEnabled() {
			cronExpr := cronExpiryCheck
			if expiryCheckTask.GetCron() != "" {
				cronExpr = expiryCheckTask.GetCron()
			}
			_, err := cronScheduler.AddFunc(cronExpr, func() {
				log.Println("[SCHEDULER] Starting subscription expiration check...")
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
				defer cancel()

				count, uids, err := app.subscriptionUsecase.UpdateExpiredSubscriptions(ctx)
				if err != nil {
					log.Printf("[SCHEDULER] Error updating expired subscriptions: %v", err)
				} else {
					log.Printf("[SCHEDULER] Updated %d expired subscriptions: %v", count, uids)
					log.Println("[SCHEDULER] Finished subscription expiration check")
				}
			})
			if err != nil {
				log.Printf("Failed to add expiration check job: %v", err)
				panic(err)
			}
			log.Printf("Expiration check task registered: cron=%s", cronExpr)
		} else {
			log.Println("Expiration check task is disabled")
		}
	}

	// 2. 续费提醒任务
	if bc.GetScheduler() != nil && bc.GetScheduler().GetRenewalReminder() != nil {
		renewalReminderTask := bc.GetScheduler().GetRenewalReminder()
		if renewalReminderTask.GetEnabled() {
			cronExpr := cronRenewalReminder
			if renewalReminderTask.GetCron() != "" {
				cronExpr = renewalReminderTask.GetCron()
			}
			_, err := cronScheduler.AddFunc(cronExpr, func() {
				log.Println("[SCHEDULER] Starting renewal reminder check...")
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
				defer cancel()

				subscriptions, total, err := app.subscriptionUsecase.GetExpiringSubscriptions(ctx, expiryCheckDays, 1, 100)
				if err != nil {
					log.Printf("[SCHEDULER] Error getting expiring subscriptions: %v", err)
					return
				}

				log.Printf("[SCHEDULER] Found %d subscriptions expiring within %d days", total, expiryCheckDays)
				for _, sub := range subscriptions {
					// TODO: 发送续费提醒通知
					log.Printf("[SCHEDULER] Reminder: User %s subscription (plan: %s) expires at %s",
						sub.UserID, sub.PlanID, sub.EndTime.Format("2006-01-02 15:04:05"))
				}
				log.Println("[SCHEDULER] Finished renewal reminder check")
			})
			if err != nil {
				log.Printf("Failed to add renewal reminder job: %v", err)
				panic(err)
			}
			log.Printf("Renewal reminder task registered: cron=%s", cronExpr)
		} else {
			log.Println("Renewal reminder task is disabled")
		}
	}

	// 3. 自动续费任务
	if bc.GetScheduler() != nil && bc.GetScheduler().GetAutoRenewal() != nil {
		autoRenewalTask := bc.GetScheduler().GetAutoRenewal()
		if autoRenewalTask.GetEnabled() {
			cronExpr := cronAutoRenewal
			if autoRenewalTask.GetCron() != "" {
				cronExpr = autoRenewalTask.GetCron()
			}
			_, err := cronScheduler.AddFunc(cronExpr, func() {
				log.Println("[SCHEDULER] Starting auto-renewal process...")
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
				defer cancel()

				totalCount, successCount, failedCount, results, err := app.subscriptionUsecase.ProcessAutoRenewals(ctx, autoRenewDaysBefore, false)
				if err != nil {
					log.Printf("[SCHEDULER] Error processing auto-renewals: %v", err)
				} else {
					log.Printf("[SCHEDULER] Auto-renewal completed: total=%d, success=%d, failed=%d",
						totalCount, successCount, failedCount)

					// 记录详细结果
					for _, result := range results {
						if result.Success {
							log.Printf("[SCHEDULER] Auto-renewal success: user=%s, plan=%s, order=%s",
								result.UserID, result.PlanID, result.OrderID)
						} else {
							log.Printf("[SCHEDULER] Auto-renewal failed: user=%s, plan=%s, error=%s",
								result.UserID, result.PlanID, result.ErrorMessage)
						}
					}
				}
				log.Println("[SCHEDULER] Finished auto-renewal process")
			})
			if err != nil {
				log.Printf("Failed to add auto-renewal job: %v", err)
				panic(err)
			}
			log.Printf("Auto-renewal task registered: cron=%s", cronExpr)
		} else {
			log.Println("Auto-renewal task is disabled")
		}
	}

	// 启动定时任务
	cronScheduler.Start()
	log.Println("========================================")
	log.Println("Scheduler started successfully")
	log.Println("Scheduled jobs:")
	if bc.GetScheduler() != nil {
		if bc.GetScheduler().GetExpiryCheck() != nil && bc.GetScheduler().GetExpiryCheck().GetEnabled() {
			cronExpr := cronExpiryCheck
			if bc.GetScheduler().GetExpiryCheck().GetCron() != "" {
				cronExpr = bc.GetScheduler().GetExpiryCheck().GetCron()
			}
			log.Printf("  - Expiration check:  %s", cronExpr)
		}
		if bc.GetScheduler().GetRenewalReminder() != nil && bc.GetScheduler().GetRenewalReminder().GetEnabled() {
			cronExpr := cronRenewalReminder
			if bc.GetScheduler().GetRenewalReminder().GetCron() != "" {
				cronExpr = bc.GetScheduler().GetRenewalReminder().GetCron()
			}
			log.Printf("  - Renewal reminder:  %s", cronExpr)
		}
		if bc.GetScheduler().GetAutoRenewal() != nil && bc.GetScheduler().GetAutoRenewal().GetEnabled() {
			cronExpr := cronAutoRenewal
			if bc.GetScheduler().GetAutoRenewal().GetCron() != "" {
				cronExpr = bc.GetScheduler().GetAutoRenewal().GetCron()
			}
			log.Printf("  - Auto-renewal:      %s", cronExpr)
		}
	}
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
		log.Println("Scheduler stopped gracefully")
	case <-time.After(5 * time.Second):
		log.Println("Scheduler forced to stop after timeout")
	}
}
