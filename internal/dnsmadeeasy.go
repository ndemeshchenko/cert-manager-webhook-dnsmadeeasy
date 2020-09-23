package internal

// Config struct
type Config struct {
	APIKey    string `json:"apiKey"`
	SecretKey string `json:"secretKey"`
	ZoneName  string `json:"zoneName"`
	APIURL    string `json:"apiURL"`
}

// DomainResponse struct representing json response
type DomainResponse struct {
	TotalRecords int `json:"totalRecords"`
	TotalPages   int `json:"totalPages"`
	Data         []struct {
		ProcessMulti       bool          `json:"processMulti"`
		ActiveThirdParties []interface{} `json:"activeThirdParties"`
		FolderID           int           `json:"folderId"`
		PendingActionID    int           `json:"pendingActionId"`
		GtdEnabled         bool          `json:"gtdEnabled"`
		Updated            int64         `json:"updated"`
		Created            int64         `json:"created"`
		Name               string        `json:"name"`
		ID                 int           `json:"id"`
	} `json:"data"`
	Page int `json:"page"`
}

// RecordResponse struct representing json response
type RecordResponse struct {
	TotalRecords int `json:"totalRecords"`
	TotalPages   int `json:"totalPages"`
	Data         []struct {
		Failed bool   `json:"failed"`
		Name   string `json:"name"`
		ID     int    `json:"id"`
	} `json:"data"`
	Page int `json:"page"`
}
