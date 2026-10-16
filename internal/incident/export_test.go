package incident

var (
	SeverityToSDK         = severityToSDK
	SeverityToString      = severityToString
	StatusToSDK           = statusToSDK
	StatusToString        = statusToString
	ParseStatuses         = parseStatuses
	EventTypeToString     = eventTypeToString
	FormatUser            = formatUser
	ShortUser             = shortUser
	DisplayStatus         = displayStatus
	ReadTextArg           = readTextArg
	NextStepHint          = nextStepHint
	NeedsConfirmation     = needsConfirmation
	ListFooter            = listFooter
	BuildListRequest      = buildListRequest
	BuildDeclareRequest   = buildDeclareRequest
	MissingDeclareFlags   = missingDeclareFlags
	BuildUpdateRequest    = buildUpdateRequest
	BuildSetStatusRequest = buildSetStatusRequest
	BuildNoteRequest      = buildNoteRequest
	InfoRows              = infoRows
	StatusDoneWord        = statusDoneWord
)
