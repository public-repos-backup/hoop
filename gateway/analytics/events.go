package analytics

const (
	EventLogin  = "hoop-login"
	EventSignup = "hoop-signup"

	EventFetchUsers             = "hoop-fetch-users"
	EventUpdateUser             = "hoop-update-user"
	EventCreateUser             = "hoop-create-user"
	EventCreateConnection       = "hoop-create-connection"
	EventUpdateConnection       = "hoop-update-connection"
	EventDeleteConnection       = "hoop-delete-connection"
	EventFetchConnections       = "hoop-fetch-connections"
	EventApiExecConnection      = "hoop-api-exec-connection"
	EventApiProxymanagerConnect = "hoop-api-proxymanager-connect"
	EventCreateAgent            = "hoop-create-agent"
	EventDeleteAgent            = "hoop-delete-agent"
	EventCreatePlugin           = "hoop-create-plugin"
	EventUdpatePlugin           = "hoop-udpate-plugin"
	EventUdpatePluginConfig     = "hoop-udpate-plugin-config"
	EventApiExecSession         = "hoop-api-exec-session"
	EventFetchSessions          = "hoop-fetch-sessions"
	EventListRunbooks           = "hoop-list-runbooks"
	EventExecRunbook            = "hoop-exec-runbook"
	EventUpdateReview           = "hoop-update-review"
	EventFetchReviews           = "hoop-fetch-reviews"
	EventApiExecReview          = "hoop-api-exec-review"
	EventSearch                 = "hoop-search"

	EventGrpcExec                = "hoop-grpc-exec"
	EventGrpcConnect             = "hoop-grpc-connect"
	EventGrpcProxyManagerConnect = "hoop-grpc-proxy-manager-connect"
)
