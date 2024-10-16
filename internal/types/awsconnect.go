package types

type AWSConnect struct {
	AWS_ACCESS_KEY_ID     string `json:"access_key_id"`
	AWS_SECRET_ACCESS_KEY string `json:"secret_access_key"`
	OIDC_ENDPOINT         string `json:"oidc_endpoint"`
}
