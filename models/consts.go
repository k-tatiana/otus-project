package models

// refer to table balance_reasons in database
const (
	ReasonMarketingActivity = iota + 1
	ReasonOrder
	ReasonUsingPoints
	ReasonBirthday
	ReasonExpirePoints
)

type HandlersMainServer string

const (
	HandlerPointsAdd HandlersMainServer = "/api/points/add"
	HandlerPointsUse HandlersMainServer = "/api/points/use"
	HandlerUsersList HandlersMainServer = "/api/users/list"
)

const OrdersSessionID = "cron-session-id"
