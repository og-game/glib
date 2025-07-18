package mq_model

import (
	"github.com/shopspring/decimal"
)

type (
	UserGameLog struct {
		UserID         int64  `json:"user_id"`
		GameID         int64  `json:"game_id"`
		PlatformID     int64  `json:"platform_id"`
		MerchantID     int64  `json:"merchant_id"`
		MerchantUserID string `json:"merchant_user_id"`
		DeviceID       string `json:"device_id"`
		DeviceIp       string `json:"device_ip"`
		DeviceOs       string `json:"device_os"`
		StartPlayTime  int64  `json:"start_play_time"`
	}

	// UserTransferRecord 用户账变记录
	UserTransferRecord struct {
		MerchantID      int64           `json:"merchant_id"`   // 商户ID
		UserID          int64           `json:"user_id"`       // 用户ID
		PlatformID      int64           `json:"platform_id"`   // 平台ID
		TransferType    int64           `json:"transfer_type"` // 转账类型（对应 v1.UserBalanceTransactionType）
		Amount          decimal.Decimal `json:"amount"`        // 金额
		BalanceBefore   decimal.Decimal `json:"balance_before"`
		BalanceAfter    decimal.Decimal `json:"balance_after"`
		CurrencyCode    string          `json:"currency_code"`     // 货币代码
		RelatedOrderID  string          `json:"related_order_id"`  // 账变记录关联的交易ID
		PlatformOrderID string          `json:"platform_order_id"` // 平台订单ID
		MerchantOrderID string          `json:"merchant_order_id"` // 商户订单ID
		TransactionID   string          `json:"transaction_id"`    // 交易ID
		TradeTime       int64           `json:"trade_time"`        // 交易时间（hao毫秒）
		Remark          string          `json:"remark"`            // 备注
		Description     string          `json:"description"`       // 描述
		ClientIp        string          `json:"client_ip"`         // 客户端IP
		UserAgent       string          `json:"user_agent"`        // 用户代理
		ExtData         string          `json:"ext_data"`          // 扩展数据
	}
	// TransferBetRecord 用户转账游戏记录
	TransferBetRecord struct {
		Status        int64           `json:"status"`         // 投注状态 ( 1-投注, 2-结算, 3-投注取消, 4-废弃)
		BetAmount     decimal.Decimal `json:"bet_amount"`     // 投注金额
		ThirdOrderNo  string          `json:"third_order_no"` // 三方订单号，通常是唯一的业务标识。
		ThirdGameID   string          `json:"third_game_id"`  // 三方平台的游戏ID。
		RoundID       string          `json:"round_id"`       // 牌局或游戏回合的唯一编号。
		UserID        string          `json:"user_id"`        // 中台用户ID。
		MerchantID    int64           `json:"merchant_id"`    // 中台商户ID。
		PlatformID    int64           `json:"platform_id"`    // 中台游戏厂商ID。
		SettledAmount decimal.Decimal `json:"settled_amount"` // // 结算金额，即玩家的输赢金额。
		CurrencyCode  string          `json:"currency_code"`  // 货币代码，遵循 ISO 4217 标准，例如 "CNY", "USD"。
		CreatedAt     int64           `json:"created_at"`     // 创建时间[投注时间]，Unix 时间戳。
		SettledAt     int64           `json:"settled_at"`     // 结算时间，Unix 时间戳。
	}
)
