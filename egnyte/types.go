package egnyte

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// Custom Client with extra metadata
type Client struct {
	http.Client
	clientId         string
	token            string
	headers          map[string]string
	root             string
	domain           string
	workgroupId      string
	dcName           string
	Username         string
	Email            string
	insecure         bool
	insecureEos      bool
	usePrivate       bool
	legacyAuthScheme bool
	WebAppURL        string
}

// options that need to be provided with every call to doRequest
type requestOptions struct {
	Root          string            // Required only if need to use a different root URL then Client root
	Method        string            // GET, POST etc
	Path          string            // relative to RootURL
	ExtraHeaders  map[string]string // If need to use any headers not provided in Client or if need to overwrite client headers
	Body          io.Reader         // Request body
	Parameters    url.Values        // URL Query parameters
	DontCloseBody bool              // Dont close resp body in do request
	Insecure      bool              // Use http instead of https
}

// Object represents a file or a folder object
type Object struct {
	Client             *Client
	Body               io.Reader
	Etag               string
	ModTime            time.Time
	Uploaded           int64     `json:"uploaded"`
	Checksum           string    `json:"checksum"`
	Size               int       `json:"size"`
	Path               string    `json:"path"`
	Name               string    `json:"name"`
	Locked             bool      `json:"locked"`
	IsFolder           bool      `json:"is_folder"`
	EntryID            string    `json:"entry_id"`
	GroupID            string    `json:"group_id"`
	LastModifiedStr    string    `json:"last_modified"`
	LastModifiedEpoch  int64     `json:"lastModified"`
	UploadedBy         string    `json:"uploaded_by"`
	NumVersions        int       `json:"num_versions"`
	ParentID           string    `json:"parent_id"`
	FolderID           string    `json:"folder_id"`
	Count              int       `json:"count"`
	Offset             int       `json:"offset"`
	TotalCount         int       `json:"total_count"`
	RestrictMoveDelete bool      `json:"restrict_move_delete"`
	PublicLinks        string    `json:"public_links"`
	AllowLinks         bool      `json:"allow_links"`
	Folders            []*Object `json:"folders"`
	Files              []*Object `json:"files"`
	Versions           []*Object `json:"versions"`
	Err                error     // This is used to store the error for this file in case of batch upload
}

// UserName is used to represent the family name and given name if a user
type UserName struct {
	FamilyName string `json:"familyName"`
	GivenName  string `json:"givenName"`
}

// User represents an Egnyte user
type User struct {
	ID                   int      `json:"id"`
	UserName             string   `json:"userName"`   // The Egnyte username for the user
	ExternalID           string   `json:"externalId"` // This is an immutable unique identifier provided by the API consumer
	Email                string   `json:"email"`      // The email address of the user
	Name                 UserName `json:"name"`       // First and last name of the user
	Active               bool     `json:"active"`     // Whether the user is active or inactive
	CreatedDate          string   `json:"createdDate"`
	LastModificationDate string   `json:"lastModificationDate"`
	LastActiveDate       string   `json:"lastActiveDate"`
	Locked               bool     `json:"locked"`
	AuthType             string   `json:"authType"` // The authentication type for the user
	UserType             string   `json:"userType"` // The type of the user
	// This is the way the user is identified within the SAML Response from an
	// SSO Identity Provider, i.e. the SAML Subject (e.g. jsmith)
	IdpUserID          string      `json:"idpUserId"`
	IsServiceAccount   bool        `json:"isServiceAccount"` // Whether user is a service account or not
	Language           string      `json:"language"`
	DeleteOnExpiry     interface{} `json:"deleteOnExpiry"`
	EmailChangePending bool        `json:"emailChangePending"`
	ExpiryDate         string      `json:"expiryDate"`
	Role               string      `json:"role"` // The role assigned to the user. Only applicable for Power Users
	// Used to bind child authentication policies to a user when using Active
	// Directory authentication in a multi-domain setup (e.g. jmiller@example.com)
	UserPrincipalName string `json:"userPrincipalName"`
	Groups            []struct {
		DisplayName string `json:"displayName"`
		Value       string `json:"value"`
	} `json:"groups"`
}

// createUserRequest is the request payload for create user API
type createUserRequest struct {
	UserName         string   `json:"userName"`         // The Egnyte username for the user
	ExternalID       string   `json:"externalId"`       // This is an immutable unique identifier provided by the API consumer
	Email            string   `json:"email"`            // The email address of the user
	Name             UserName `json:"name"`             // First and last name of the user
	Active           bool     `json:"active"`           // Whether the user is active or inactive
	IsServiceAccount bool     `json:"isServiceAccount"` // Whether user is a service account or not
	Language         string   `json:"language"`
	AuthType         string   `json:"authType"` // The authentication type for the user
	UserType         string   `json:"userType"` // The type of the user
	// This is the way the user is identified within the SAML Response from an
	// SSO Identity Provider, i.e. the SAML Subject (e.g. jsmith)

	Role string `json:"role"` // The role assigned to the user. Only applicable for Power Users

	IdpUserID string `json:"idpUserId"` // Used to bind child authentication policies to a user when using Active
	// Directory authentication in a multi-domain setup (e.g. jmiller@example.com)

	UserPrincipalName string `json:"userPrincipalName"`
	SendInvite        bool   `json:"sendInvite"` // If set to true when creating a user, an invitation
	// email will be sent (if the user is created in active
	// state). For authType “egnyte” will always return “True”
}

// listUserResponse is the response payload of the list users API
type listUserResponse struct {
	TotalResults int     `json:"totalResults"`
	ItemsPerPage int     `json:"itemsPerPage"`
	StartIndex   int     `json:"startIndex"`
	Resources    []*User `json:"resources"`
}

// GroupMember represents a member of an Egnyte group
type GroupMember struct {
	Username string `json:"username"` // The username of a group member
	ID       int64  `json:"value"`    // The globally unique id of a group member
	Display  string `json:"display"`  // The display name of a group member
}

// Group represents an Egnyte group
type Group struct {
	ID          string         `json:"id"`          // The globally unique group ID.
	DisplayName string         `json:"displayName"` // The name of the group.
	Members     []*GroupMember `json:"members"`     // A JSON array containing all users in the group.
}

// listGroupResponse is the response payload of the list groups API
type listGroupResponse struct {
	Schemas      []string `json:"schemas"`      // The SCIM schema version of the response.
	TotalResults int      `json:"totalResults"` // The total number of results matching the query.
	ItemsPerPage int      `json:"itemsPerPage"` // The number of results returned.
	StartIndex   int      `json:"startIndex"`   // The 1-based index of the first result in the current set of results.
	Resources    []*Group `json:"resources"`    // A JSON array that holds all of the group objects.
}

// createGroupRequest is the request payload of the create group API
type createGroupRequest struct {
	DisplayName string         `json:"displayName"` // The name of the group.
	Members     []*GroupMember `json:"members"`     // A JSON array containing all users in the group.
}

// FolderPermission represents the permissions of a folder in Egnyte
type FolderPermission struct {
	UserPerms             map[string]string `json:"userPerms"`
	GroupPerms            map[string]string `json:"groupPerms"`
	InheritsPermissions   bool              `json:"inheritsPermissions"`
	KeepParentPermissions bool              `json:"keepParentPermissions"`
}

var timeoutStatusCodes = []int{408, 504, 598, 599}

type Error struct {
	// StatusCode is the HTTP response status code and will always be populated
	StatusCode int `json:"status_code"`
	// ErrorCode is the Egnyte error code as returned in the response
	ErrorCode string
	Message   string
	// Body is the raw response returned by the server.
	// It is often but not always JSON, depending on how the request fails.
	Body string
	// Header contains the response header fields from the server.
	Header http.Header
}

// Error returns a string for the error and satisfies the error interface
func (e *Error) Error() string {
	if e.Message == "" {
		return fmt.Sprintf("got HTTP response code %d with body: %v", e.StatusCode, e.Body)
	}
	return e.Message
}

func (e *Error) Timeout() bool {
	for _, statusCode := range timeoutStatusCodes {
		if e.StatusCode == statusCode {
			return true
		}
	}
	return false
}

// errorReply is the response received from Egnyte in case of an error
type errorReply struct {
	ResponseCode string `json:"responseCode"`
	ResponseMsg  string `json:"responseMsg"`
	Success      bool   `json:"success"`
}

// errorReply2 is another response structure received from Egnyte in case of an error :-(
type errorReply2 struct {
	FormErrors []struct {
		Code string `json:"code"`
		Msg  string `json:"msg"`
	} `json:"formErrors"`
	InputErrors struct {
		Code string `json:"code"`
		Msg  string `json:"msg"`
	} `json:"inputErrors"`
}

// errorReply3 is another response structure received from Egnyte in case of an error :-( :-(
type errorReply3 struct {
	ErrorMessage string `json:"errorMessage"`
}

// userInfoResponse is the response of userinfo API
type userInfoResponse struct {
	Username string `json:"username"`
	Email    string `json:"email"`
}

// createFolderRequest is the request for create folder API
type createFolderRequest struct {
	Action string `json:"action"`
}

type MultiPutFileContent struct {
	LastModified   string `json:"lastModified"`
	Content        string `json:"content"`
	LastModifiedBy string `json:"lastModifiedBy"`
}

// Event is the latest  build ID
type EventID struct {
	Timestamp     string `json:"timestamp"`
	LatestEventID int64  `json:"latest_event_id"`
	OldestEventID int64  `json:"oldest_event_id"`
}

type UploadInfo struct {
	Path      string
	Data      io.Reader
	Csum      string
	ChunkNum  int
	ChunkSize int64
	UploadID  string
}
