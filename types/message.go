package types

type Message struct {
	Alerts            []Alert           `json:"alerts"`
	CommonAnnotations map[string]string `json:"commonAnnotations"`
	CommonLabels      map[string]string `json:"commonLabels"`
	Status            string            `json:"status"`
}
