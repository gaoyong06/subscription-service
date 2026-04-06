package biz

import (
	"fmt"
	"strings"
	"time"

	"subscription-service/internal/constants"
)

// 本包内所有账期边界均在 UTC 下按「日历日期 00:00:00」计算；EndTime 为「不包含」边界（有效期内 now < EndTime）。

// ValidatePlanPeriod 校验套餐周期配置
func ValidatePlanPeriod(periodType string, intervalCount int32) error {
	pt := strings.ToUpper(strings.TrimSpace(periodType))
	switch pt {
	case constants.PeriodTypeForever:
		return nil
	case constants.PeriodTypeDay, constants.PeriodTypeMonth, constants.PeriodTypeYear:
		if intervalCount < 1 {
			return fmt.Errorf("interval_count must be >= 1 for period_type %s", pt)
		}
		return nil
	default:
		return fmt.Errorf("invalid period_type: %s", periodType)
	}
}

// NormalizePeriodType 返回大写规范枚举值
func NormalizePeriodType(periodType string) string {
	return strings.ToUpper(strings.TrimSpace(periodType))
}

// ForeverEndUTC 终身/免费档等费用周期的「远端」结束时间（排他边界）
func ForeverEndUTC() time.Time {
	return time.Date(9999, 12, 31, 0, 0, 0, 0, time.UTC)
}

// SubscriptionCalendarDateUTC 取订阅生效「日历日」的 UTC 零点（用于自然月/自然年起算）
func SubscriptionCalendarDateUTC(t time.Time) time.Time {
	t = t.UTC()
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

// BillingAnchorDayFromPurchase 自然月/自然年使用的锚点日（1–31）；DAY、FOREVER 返回 0
func BillingAnchorDayFromPurchase(purchase time.Time, periodType string) int32 {
	pt := NormalizePeriodType(periodType)
	if pt != constants.PeriodTypeMonth && pt != constants.PeriodTypeYear {
		return 0
	}
	return int32(SubscriptionCalendarDateUTC(purchase).Day())
}

// FirstPeriodEndUTC 首期结束时间（排他）：从购买时刻起算第一个完整计费周期
func FirstPeriodEndUTC(purchase time.Time, periodType string, intervalCount int32) (time.Time, error) {
	if err := ValidatePlanPeriod(periodType, intervalCount); err != nil {
		return time.Time{}, err
	}
	pt := NormalizePeriodType(periodType)
	switch pt {
	case constants.PeriodTypeForever:
		return ForeverEndUTC(), nil
	case constants.PeriodTypeDay:
		start := SubscriptionCalendarDateUTC(purchase)
		return start.AddDate(0, 0, int(intervalCount)), nil
	case constants.PeriodTypeMonth:
		start := SubscriptionCalendarDateUTC(purchase)
		anchor := start.Day()
		return addCalendarMonthsUTC(start, int(intervalCount), anchor), nil
	case constants.PeriodTypeYear:
		start := SubscriptionCalendarDateUTC(purchase)
		anchor := start.Day()
		return addCalendarYearsMultiUTC(start, int(intervalCount), anchor), nil
	default:
		return time.Time{}, fmt.Errorf("unsupported period_type: %s", periodType)
	}
}

// NextPeriodEndUTC 续费：从当前周期结束时刻（已是下一期的 UTC 零点边界）再顺延一个周期
func NextPeriodEndUTC(currentEnd time.Time, anchorDay int32, periodType string, intervalCount int32) (time.Time, error) {
	if err := ValidatePlanPeriod(periodType, intervalCount); err != nil {
		return time.Time{}, err
	}
	pt := NormalizePeriodType(periodType)
	switch pt {
	case constants.PeriodTypeForever:
		return ForeverEndUTC(), nil
	case constants.PeriodTypeDay:
		return currentEnd.UTC().AddDate(0, 0, int(intervalCount)), nil
	case constants.PeriodTypeMonth:
		return addCalendarMonthsUTC(currentEnd.UTC(), int(intervalCount), int(anchorDay)), nil
	case constants.PeriodTypeYear:
		return addCalendarYearsMultiUTC(currentEnd.UTC(), int(intervalCount), int(anchorDay)), nil
	default:
		return time.Time{}, fmt.Errorf("unsupported period_type: %s", periodType)
	}
}

func daysInMonth(year int, month time.Month) int {
	// 下月1日减1天即本月最后一天
	first := time.Date(year, month, 1, 0, 0, 0, 0, time.UTC)
	return first.AddDate(0, 1, -1).Day()
}

func addCalendarMonthsUTC(from time.Time, months int, anchorDay int) time.Time {
	from = from.UTC()
	y, m, _ := from.Date()
	// 转为 (year, monthIndex) 其中 monthIndex 0–11
	totalMonths := int(m)-1 + months + y*12
	newY := totalMonths / 12
	newM := time.Month((totalMonths % 12) + 1)
	dim := daysInMonth(newY, newM)
	day := anchorDay
	if day > dim {
		day = dim
	}
	return time.Date(newY, newM, day, 0, 0, 0, 0, time.UTC)
}

// addCalendarYearsMultiUTC 从周期边界日一次性增加 yearCount 个自然年（锚点日 + 闰年 2 月规则）
func addCalendarYearsMultiUTC(from time.Time, yearCount int, anchorDay int) time.Time {
	from = from.UTC()
	y, m, _ := from.Date()
	y += yearCount
	dim := daysInMonth(y, m)
	day := anchorDay
	if day > dim {
		day = dim
	}
	return time.Date(y, m, day, 0, 0, 0, 0, time.UTC)
}
