/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package argocdModel

import "time"

// Application is required with full specification since ArgoCD needs a PUT to update the syncPolicy, and there is no
// PATCH available
type V1alpha1Application struct {
	Metadata struct {
		Name              string    `json:"name"`
		Namespace         string    `json:"namespace"`
		Uid               string    `json:"uid"`
		ResourceVersion   string    `json:"resourceVersion"`
		Generation        int       `json:"generation"`
		CreationTimestamp time.Time `json:"creationTimestamp"`
		ManagedFields     []struct {
			Manager    string    `json:"manager"`
			Operation  string    `json:"operation"`
			ApiVersion string    `json:"apiVersion"`
			Time       time.Time `json:"time"`
			FieldsType string    `json:"fieldsType"`
			FieldsV1   struct {
				FSpec struct {
					Field1 struct {
					} `json:"."`
					FDestination struct {
						Field1 struct {
						} `json:"."`
						FNamespace struct {
						} `json:"f:namespace"`
						FServer struct {
						} `json:"f:server"`
					} `json:"f:destination"`
					FProject struct {
					} `json:"f:project"`
					FSource struct {
						Field1 struct {
						} `json:"."`
						FPath struct {
						} `json:"f:path"`
						FRepoURL struct {
						} `json:"f:repoURL"`
					} `json:"f:source"`
					FSyncPolicy struct {
					} `json:"f:syncPolicy"`
				} `json:"f:spec,omitempty"`
				FStatus struct {
					Field1 struct {
					} `json:".,omitempty"`
					FHealth struct {
						FStatus struct {
						} `json:"f:status,omitempty"`
					} `json:"f:health"`
					FSummary struct {
						FImages struct {
						} `json:"f:images,omitempty"`
					} `json:"f:summary"`
					FSync struct {
						Field1 struct {
						} `json:".,omitempty"`
						FComparedTo struct {
							Field1 struct {
							} `json:".,omitempty"`
							FDestination struct {
								FNamespace struct {
								} `json:"f:namespace,omitempty"`
								FServer struct {
								} `json:"f:server,omitempty"`
							} `json:"f:destination"`
							FSource struct {
								FPath struct {
								} `json:"f:path,omitempty"`
								FRepoURL struct {
								} `json:"f:repoURL,omitempty"`
							} `json:"f:source"`
						} `json:"f:comparedTo"`
						FRevision struct {
						} `json:"f:revision,omitempty"`
						FStatus struct {
						} `json:"f:status,omitempty"`
					} `json:"f:sync"`
					FHistory struct {
					} `json:"f:history,omitempty"`
					FOperationState struct {
						Field1 struct {
						} `json:"."`
						FFinishedAt struct {
						} `json:"f:finishedAt"`
						FMessage struct {
						} `json:"f:message"`
						FOperation struct {
							Field1 struct {
							} `json:"."`
							FInitiatedBy struct {
								Field1 struct {
								} `json:"."`
								FUsername struct {
								} `json:"f:username"`
							} `json:"f:initiatedBy"`
							FRetry struct {
							} `json:"f:retry"`
							FSync struct {
								Field1 struct {
								} `json:"."`
								FRevision struct {
								} `json:"f:revision"`
								FSyncStrategy struct {
									Field1 struct {
									} `json:"."`
									FHook struct {
									} `json:"f:hook"`
								} `json:"f:syncStrategy"`
							} `json:"f:sync"`
						} `json:"f:operation"`
						FPhase struct {
						} `json:"f:phase"`
						FStartedAt struct {
						} `json:"f:startedAt"`
						FSyncResult struct {
							Field1 struct {
							} `json:"."`
							FResources struct {
							} `json:"f:resources"`
							FRevision struct {
							} `json:"f:revision"`
							FSource struct {
								Field1 struct {
								} `json:"."`
								FPath struct {
								} `json:"f:path"`
								FRepoURL struct {
								} `json:"f:repoURL"`
							} `json:"f:source"`
						} `json:"f:syncResult"`
					} `json:"f:operationState,omitempty"`
					FReconciledAt struct {
					} `json:"f:reconciledAt,omitempty"`
					FResources struct {
					} `json:"f:resources,omitempty"`
					FSourceType struct {
					} `json:"f:sourceType,omitempty"`
				} `json:"f:status"`
			} `json:"fieldsV1"`
		} `json:"managedFields"`
	} `json:"metadata"`
	Spec struct {
		Source struct {
			RepoURL string `json:"repoURL"`
			Path    string `json:"path"`
		} `json:"source"`
		Destination struct {
			Server    string `json:"server"`
			Namespace string `json:"namespace"`
		} `json:"destination"`
		Project    string `json:"project"`
		SyncPolicy struct {
		} `json:"syncPolicy"`
	} `json:"spec"`
	Status struct {
		Resources []struct {
			Version   string `json:"version"`
			Kind      string `json:"kind"`
			Namespace string `json:"namespace"`
			Name      string `json:"name"`
			Status    string `json:"status"`
			Health    struct {
				Status  string `json:"status"`
				Message string `json:"message,omitempty"`
			} `json:"health"`
			Group string `json:"group,omitempty"`
		} `json:"resources"`
		Sync struct {
			Status     string `json:"status"`
			ComparedTo struct {
				Source struct {
					RepoURL string `json:"repoURL"`
					Path    string `json:"path"`
				} `json:"source"`
				Destination struct {
					Server    string `json:"server"`
					Namespace string `json:"namespace"`
				} `json:"destination"`
			} `json:"comparedTo"`
			Revision string `json:"revision"`
		} `json:"sync"`
		Health struct {
			Status string `json:"status"`
		} `json:"health"`
		History []struct {
			Revision   string    `json:"revision"`
			DeployedAt time.Time `json:"deployedAt"`
			Id         int       `json:"id"`
			Source     struct {
				RepoURL string `json:"repoURL"`
				Path    string `json:"path"`
			} `json:"source"`
			DeployStartedAt time.Time `json:"deployStartedAt"`
		} `json:"history"`
		ReconciledAt   time.Time `json:"reconciledAt"`
		OperationState struct {
			Operation struct {
				Sync struct {
					Revision     string `json:"revision"`
					SyncStrategy struct {
						Hook struct {
						} `json:"hook"`
					} `json:"syncStrategy"`
				} `json:"sync"`
				InitiatedBy struct {
					Username string `json:"username"`
				} `json:"initiatedBy"`
				Retry struct {
				} `json:"retry"`
			} `json:"operation"`
			Phase      string `json:"phase"`
			Message    string `json:"message"`
			SyncResult struct {
				Resources []struct {
					Group     string `json:"group"`
					Version   string `json:"version"`
					Kind      string `json:"kind"`
					Namespace string `json:"namespace"`
					Name      string `json:"name"`
					Status    string `json:"status"`
					Message   string `json:"message"`
					HookPhase string `json:"hookPhase"`
					SyncPhase string `json:"syncPhase"`
				} `json:"resources"`
				Revision string `json:"revision"`
				Source   struct {
					RepoURL string `json:"repoURL"`
					Path    string `json:"path"`
				} `json:"source"`
			} `json:"syncResult"`
			StartedAt  time.Time `json:"startedAt"`
			FinishedAt time.Time `json:"finishedAt"`
		} `json:"operationState"`
		SourceType string `json:"sourceType"`
		Summary    struct {
			Images []string `json:"images"`
		} `json:"summary"`
	} `json:"status"`
}
