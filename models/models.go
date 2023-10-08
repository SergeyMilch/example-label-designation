package models

type EventData struct {
	ID     string `json:"_id"`
	Key    string `json:"_key"`
	Rev    string `json:"_rev"`
	Author struct {
		MoID     int    `json:"mo_id"`
		UserID   int    `json:"user_id"`
		UserName string `json:"user_name"`
	} `json:"author"`
	Group  string `json:"group"`
	Msg    string `json:"msg"`
	Params struct {
		IndicatorToMoID int `json:"indicator_to_mo_id"`
		Period          struct {
			End     string `json:"end"`
			Start   string `json:"start"`
			TypeID  int    `json:"type_id"`
			TypeKey string `json:"type_key"`
		} `json:"period"`
		Platform string `json:"platform"`
	} `json:"params"`
	Time string `json:"time"`
	Type string `json:"type"`
}

type MySQLResponse struct {
	IndicatorToMoFactID int `json:"indicator_to_mo_fact_id"`
}
