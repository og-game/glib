package genid

import (
	"fmt"
	"github.com/og-game/glib/stores/snowx"
	"github.com/spf13/cast"
	"github.com/zeromicro/go-zero/core/logx"
	"time"
	"unicode"
)

// OrderType represents different types of orders
type OrderType string

const (
	// DefaultOrder represents default order type
	DefaultOrder OrderType = "OR"
	// TransferIn 转入操作【用户转入中台】
	TransferIn OrderType = "TI"
	// TransferOut 转出操作【用户转出中台】
	TransferOut OrderType = "TO"
	// TransactionBet 投注订单
	TransactionBet OrderType = "GB"
	// TransactionDepositHold 押金订单
	TransactionDepositHold OrderType = "DH"
	// TransactionDepositRefund 押金退还订单
	TransactionDepositRefund OrderType = "DR"
	// TransactionSettlementCancel 结算撤单订单
	TransactionSettlementCancel OrderType = "SC"
	// TransactionBetCancel 投注取消订单
	TransactionBetCancel OrderType = "BC"
	// TransactionSettlement 派奖结算订单
	TransactionSettlement OrderType = "GS"
	// TransactionRepaySettlement 重新派奖订单
	TransactionRepaySettlement OrderType = "RS"
	// TransactionUserBalanceChange 用户账变记录
	TransactionUserBalanceChange OrderType = "TC"
	// TransactionAdjustAmount 调整金额订单
	TransactionAdjustAmount OrderType = "TA"
	// TransactionBadDebt 坏账记录订单
	TransactionBadDebt OrderType = "BD"
)

// 订单号格式常量
const (
	OrderTypeLength = 2  // 订单类型长度
	DateLength      = 8  // 日期长度 (YYYYMMDD)
	MinOrderLength  = 11 // 最小订单号长度 (类型2位 + 日期8位 + 至少1位序号)
)

// GenerateOrderNo 生成订单编号。
// 参数:
//
//	orderType: 订单类型。如果为空，默认使用 DefaultOrder。
//
// 返回值:
//
//	string: 生成的订单编号。
//	error: 如果生成唯一ID失败，则返回错误。
func GenerateOrderNo(orderType OrderType) (string, error) {
	if orderType == "" {
		orderType = DefaultOrder
	}

	orderNo, err := snowx.GenId()
	if err != nil {
		return "", fmt.Errorf("failed to generate unique ID: %w", err)
	}

	orderNoStr := fmt.Sprintf("%s%s%d",
		orderType,
		time.Now().Format("20060102"),
		orderNo,
	)

	return orderNoStr, nil
}

// ==================== 通用方法 ====================

// GenerateGameOrder 根据交易类型生成订单号
// 参数:
//
//	transactionType: 游戏交易类型枚举
//
// 返回值:
//
//	string: 生成的订单号，如果失败返回空字符串
func GenerateGameOrder(orderType OrderType) string {
	id, err := GenerateOrderNo(orderType)
	if err != nil {
		logx.Errorf("Failed to generate transaction order for type %v: %v", orderType, err)
		return ""
	}
	return id
}

func GenerateUserBalanceChangeOrder() string {
	id, err := GenerateOrderNo(TransactionUserBalanceChange)
	if err != nil {
		logx.Errorf("Failed to generate user balance change order: %v", err)
		return ""
	}
	return id
}

func GenerateUserBadDebtOrder() string {
	id, err := GenerateOrderNo(TransactionBadDebt)
	if err != nil {
		logx.Errorf("Failed to generate user bad debt order: %v", err)
		return ""
	}
	return id
}

func GenerateTransferInOrder() string {
	id, err := GenerateOrderNo(TransferIn)
	if err != nil {
		logx.Errorf("Failed to generate transfer in order: %v", err)
		return ""
	}
	return id
}

func GenerateTransferOutOrder() string {
	id, err := GenerateOrderNo(TransferOut)
	if err != nil {
		logx.Errorf("Failed to generate transfer out order: %v", err)
		return ""
	}
	return id
}

// ValidateOrderNo 验证订单号是否有效。
// 订单号的有效性基于以下两个条件：
// 1. 订单号的长度必须不少于11位。
// 2. 订单号的前缀必须是有效的订单类型前缀之一。
// 参数:
//
//	orderNo - 待验证的订单号字符串。
//
// 返回值:
//
//	如果订单号有效，返回true；否则返回false。
func ValidateOrderNo(orderNo string) bool {
	//
	if len(orderNo) < MinOrderLength {
		return false
	}
	//
	prefix := orderNo[:OrderTypeLength]
	validPrefixes := map[string]bool{
		string(TransferIn):                   true,
		string(TransferOut):                  true,
		string(DefaultOrder):                 true,
		string(TransactionBet):               true,
		string(TransactionDepositHold):       true,
		string(TransactionDepositRefund):     true,
		string(TransactionSettlementCancel):  true,
		string(TransactionBetCancel):         true,
		string(TransactionSettlement):        true,
		string(TransactionRepaySettlement):   true,
		string(TransactionUserBalanceChange): true,
		string(TransactionAdjustAmount):      true,
		string(TransactionBadDebt):           true,
	}
	return validPrefixes[prefix]
}

// ParsedOrder 表示解析后的订单号的各个组成部分
type ParsedOrder struct {
	OrderType   OrderType // 订单类型 (TI, TO, RB, OR, GB)
	CreatedAt   time.Time // 从订单号中解析出的创建日期
	SequenceID  int64     // 序列号部分（去掉日期后的数字部分）
	FullOrderID int64     // 去除前缀后的完整ID（日期+序列号）
}

// ParseOrderNo 解析订单号并返回解析后的订单信息。
// orderNo 是要解析的订单号字符串。
// 返回一个指向 ParsedOrder 的指针和一个错误（如果解析失败）。
func ParseOrderNo(orderNo string) (*ParsedOrder, error) {
	// 首先验证订单号格式
	if !ValidateOrderNo(orderNo) {
		return nil, fmt.Errorf("无效的订单号格式")
	}

	// 提取订单类型（前2个字符）
	orderTypeStr := orderNo[:OrderTypeLength]
	orderType := OrderType(orderTypeStr)

	// 提取并解析日期
	dateStr := orderNo[OrderTypeLength : OrderTypeLength+DateLength]
	createdAt, err := time.Parse("20060102", dateStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse date from order number: %w", err)
	}

	// 提取序列号
	sequenceStr := orderNo[OrderTypeLength+DateLength:]
	if len(sequenceStr) == 0 {
		return nil, fmt.Errorf("missing sequence ID in order number")
	}

	sequenceID, err := cast.ToInt64E(sequenceStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse sequence ID from order number: %w", err)
	}

	// 验证序列号是否为正数
	if sequenceID <= 0 {
		return nil, fmt.Errorf("sequence ID must be positive, got: %d", sequenceID)
	}

	// 构建去除前缀后的完整ID（日期+序列号）
	fullOrderIDStr := orderNo[OrderTypeLength:] // 去除前2位订单类型前缀

	fullOrderID, err := cast.ToInt64E(fullOrderIDStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse sequence ID from order number: %w", err)
	}

	// 验证序列号是否为正数
	if fullOrderID <= 0 {
		return nil, fmt.Errorf("fullOrderID ID must be positive, got: %d", fullOrderID)
	}

	// 返回解析后的订单信息
	return &ParsedOrder{
		OrderType:   orderType,
		CreatedAt:   createdAt,
		SequenceID:  sequenceID,
		FullOrderID: fullOrderID,
	}, nil
}

// ReplaceOrderPrefix 替换订单号前缀字母
// prefix: 新的前缀，必须是2个字母
// orderNo: 原订单号
// 返回: 替换后的订单号，错误信息
func ReplaceOrderPrefix(prefix OrderType, orderNo string) (string, error) {
	// 检查前缀格式
	if len(prefix) != 2 {
		return "", fmt.Errorf("prefix must be 2 characters, got: %s", prefix)
	}
	if !isAlpha(prefix) {
		return "", fmt.Errorf("prefix must contain only letters, got: %s", prefix)
	}

	// 检查订单号长度
	if len(orderNo) <= 2 {
		return "", fmt.Errorf("invalid order number length: %s", orderNo)
	}

	// 替换前缀
	return string(prefix) + orderNo[2:], nil
}

// isAlpha 检查字符串是否只包含字母
func isAlpha(s OrderType) bool {
	for _, r := range s {
		if !unicode.IsLetter(r) {
			return false
		}
	}
	return true
}

// String 实现了 fmt.Stringer 接口，用于生成 ParsedOrder 类型的字符串表示。
// 这个方法主要用于日志记录和调试，通过格式化字符串的形式展示订单的关键信息。
// 返回值是一个字符串，包含了订单类型、创建日期和唯一标识符。
func (p *ParsedOrder) String() string {
	return fmt.Sprintf("OrderType: %s, CreatedAt: %s, SequenceID: %d",
		p.OrderType,
		p.CreatedAt.Format("20060102"),
		p.SequenceID)
}
