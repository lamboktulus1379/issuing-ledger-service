package models

type ReqHeader struct {
	Accept          string `json:"accept"`
	AcceptLanguage  string `json:"accept-language"`
	Connection      string `json:"connection"`
	ContentType     string `json:"content-type"`
	Cookie          string `json:"cookie"`
	Origin          string `json:"origin"`
	Referer         string `json:"referer"`
	SecFetchDest    string `json:"sec-fetch-dest"`
	SecFetchMode    string `json:"sec-fetch-mode"`
	SectFetchSite   string `json:"sec-fetch-site"`
	UserAgent       string `json:"user-agent"`
	XRequestedWith  string `json:"x-requested-with"`
	SecChUa         string `json:"sec-ch-ua"`
	SecChUaMobile   string `json:"sec-ch-ua-mobile"`
	SecChUaPlatform string `json:"sec-ch-ua-platform"`
}
