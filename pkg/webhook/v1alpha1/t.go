package v1alpha1

import (
	"context"
	"encoding/json"
	"net/http"

	workflowv1alpha1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

type ArgoWorkflowHandler struct {
	Client  client.Client
	decoder *admission.Decoder
}

//+kubebuilder:webhook:path=/mutate-v1alpha1-argoworkflow,mutating=true,failurePolicy=fail,sideEffects=None,groups=argoproj.io,resources=workflows,verbs=create;update,versions=v1alpha1,name=mworkflow.argoproj.io,admissionReviewVersions=v1

// +kubebuilder:rbac:groups=workflow.argoproj.io,resources=workflows,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=workflow.argoproj.io,resources=workflows/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=workflow.argoproj.io,resources=workflows/finalizers,verbs=update

// ArgoWorkflowHandler
// TODO(user): Modify the Handle function to
func (a *ArgoWorkflowHandler) Handle(ctx context.Context, req admission.Request) admission.Response {
	workflow := &workflowv1alpha1.Workflow{}

	err := a.decoder.Decode(req, workflow)
	if err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}

	// TODO(user): your logic here

	marshaledWorkflow, err := json.Marshal(workflow)
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}

	return admission.PatchResponseFromRaw(req.Object.Raw, marshaledWorkflow)
}

// ArgoWorkflowHandler InjectDecoder
func (a *ArgoWorkflowHandler) InjectDecoder(d *admission.Decoder) error {
	a.decoder = d
	return nil
}
