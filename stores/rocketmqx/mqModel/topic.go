package mq_model

const (
	TopicUserGameLog             = "user_game_log"
	TopicUserBalanceChangeRecord = "user_balance_change_record" // 用户账变记录topic
	TopicUserGameBetRecord       = "user_game_bet_record"       // 用户游戏记录topic

	// 通知相关topic和通知大类一一对应

	TopicNotifySystemEvent = "system_event_notify" // 系统事件通知topic
	TopicNotifyRiskEvent   = "risk_event_notify"   // 风险事件通知topic
	TopicNotifyGameEvent   = "game_event_notify"   // 游戏事件通知topic
	TopicNotifyReportEvent = "report_event_notify" // 报表事件通知topic
)
