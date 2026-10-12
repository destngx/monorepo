package domain

type Accounts struct {
	Range string     `json:"range"`
	Rows  [][]string `json:"rows"`
}
