package argocdModel

type SyncResponse struct {
	Status struct {
		Sync struct {
			Status string `json:"status"`
		} `json:"sync"`
	}
}
