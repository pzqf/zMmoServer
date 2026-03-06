package constants

type ErrorCode int32

const (
	Success       ErrorCode = 0
	UnknownError  ErrorCode = 1
	InvalidParam  ErrorCode = 2
	Unauthorized  ErrorCode = 3
	Forbidden     ErrorCode = 4
	NotFound      ErrorCode = 5
	AlreadyExists ErrorCode = 6
	ServerBusy    ErrorCode = 7
	ServerError   ErrorCode = 8
	Timeout       ErrorCode = 9
	DBError       ErrorCode = 10
	CacheError    ErrorCode = 11
	NetworkError  ErrorCode = 12

	AccountNotExist      ErrorCode = 1000
	AccountAlreadyExists ErrorCode = 1001
	PasswordWrong        ErrorCode = 1002
	AccountLocked        ErrorCode = 1003
	AccountBanned        ErrorCode = 1004
	TokenInvalid         ErrorCode = 1005
	TokenExpired         ErrorCode = 1006
	LoginTooManyTimes    ErrorCode = 1007
	SessionInvalid       ErrorCode = 1008

	PlayerNotExist       ErrorCode = 2000
	PlayerAlreadyExists  ErrorCode = 2001
	PlayerOffline        ErrorCode = 2002
	PlayerOnline         ErrorCode = 2003
	PlayerLevelNotEnough ErrorCode = 2004
	PlayerNameInvalid    ErrorCode = 2005
	PlayerNameExists     ErrorCode = 2006
	TooManyPlayers       ErrorCode = 2007

	ItemNotExist       ErrorCode = 3000
	ItemCountNotEnough ErrorCode = 3001
	ItemMaxCount       ErrorCode = 3002
	ItemBagFull        ErrorCode = 3003
	ItemBind           ErrorCode = 3004
	ItemExpired        ErrorCode = 3005

	MapNotExist     ErrorCode = 4000
	MapEnterFailed  ErrorCode = 4001
	MapPlayerMax    ErrorCode = 4002
	MapInvalidPos   ErrorCode = 4003
	MapNotReachable ErrorCode = 4004

	GuildNotExist         ErrorCode = 5000
	GuildAlreadyExists    ErrorCode = 5001
	GuildMemberMax        ErrorCode = 5002
	GuildLevelNotEnough   ErrorCode = 5003
	GuildPermissionDenied ErrorCode = 5004
	AlreadyInGuild        ErrorCode = 5005
	NotInGuild            ErrorCode = 5006

	SkillNotExist    ErrorCode = 6000
	SkillCD          ErrorCode = 6001
	SkillMPNotEnough ErrorCode = 6002
	TargetTooFar     ErrorCode = 6003
	TargetInvalid    ErrorCode = 6004
	AlreadyDead      ErrorCode = 6005

	ServiceUnavailable ErrorCode = 7000
	ServiceBusy        ErrorCode = 7001
	ServiceClosed      ErrorCode = 7002
)
