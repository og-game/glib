package mq_model

type (
	UserGameLog struct {
		UserID        int64  `json:"user_id"`
		GameID        int64  `json:"game_id"`
		PlatformID    int64  `json:"platform_id"`
		DeviceID      string `json:"device_id"`
		DeviceIp      string `json:"device_ip"`
		DeviceOs      string `json:"device_os"`
		StartPlayTime int64  `json:"start_play_time"`
	}
)
