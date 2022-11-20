package egnyte

const URI_PREFIX_V1 = "/pubapi/v1/"
const URI_PREFIX_V2 = "/pubapi/v2/"

const (
	// Filesystem URIS
	URI_LIST          = URI_PREFIX_V1 + "fs%s"
	URI_DELETE_OBJECT = URI_PREFIX_V1 + "fs%s"
	URI_GET_FILE      = URI_PREFIX_V1 + "fs-content%s"
	URI_CREATE_FOLDER = URI_PREFIX_V1 + "fs%s"

	URI_CHUNKED_UPLOAD = URI_PREFIX_V1 + "fs-content-chunked%s"

	// Auth
	URI_OAUTH = "/puboauth/token"

	// Event URIS
	URI_FETCH_EVENT_ID = URI_PREFIX_V1 + "events/cursor"

	// Group Management URIs
	URI_GROUPS = URI_PREFIX_V2 + "groups"

	// User Management URIs
	URI_USERS    = URI_PREFIX_V2 + "users"
	URI_USERINFO = URI_PREFIX_V1 + "userinfo"

	// Permission URIs
	URI_PERMISSIONS = URI_PREFIX_V2 + "perms"
)
