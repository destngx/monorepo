package domain

type CacheEntry struct {
	Key   string `json:"key"`
	Value string `json:"value"`
	Found bool   `json:"found"`
}
