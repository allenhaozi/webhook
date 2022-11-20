package v1alpha1

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"text/template"

	argoworkflowv1alpha1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

type ArgoWorkflowHandler struct {
	Client  client.Client
	decoder *admission.Decoder
	Log     logr.Logger
}

//+kubebuilder:webhook:path=/mutate-v1alpha1-argoworkflow,mutating=true,failurePolicy=fail,sideEffects=None,groups=argoproj.io,resources=workflows,verbs=create;update,versions=v1alpha1,name=mworkflow.argoproj.io,admissionReviewVersions=v1

// +kubebuilder:rbac:groups=workflow.argoproj.io,resources=workflows,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=workflow.argoproj.io,resources=workflows/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=workflow.argoproj.io,resources=workflows/finalizers,verbs=update

// +kubebuilder:rbac:groups=sparkoperator.k8s.io,resources=sparkapplications,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=sparkoperator.k8s.io,resources=sparkapplications/status,verbs=get;update;patch

// podAnnotator adds an annotation to every incoming pods.
func (a *ArgoWorkflowHandler) Handle(ctx context.Context, req admission.Request) admission.Response {
	workflow := &argoworkflowv1alpha1.Workflow{}

	err := a.decoder.Decode(req, workflow)
	if err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}
	a.Log.Info("Workflow webhook Handle", "got workflow: %v", workflow)
	for k, v := range workflow.Spec.Templates {
		v := v
		manifest := v.Resource.Manifest
		manifest = a.process(manifest)
		v.Resource.Manifest = manifest
		workflow.Spec.Templates[k] = v
	}
	a.Log.Info("Workflow webhook Handle", "render workflow: %v", workflow)

	marshaledWorkflow, err := json.Marshal(workflow)
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}

	return admission.PatchResponseFromRaw(req.Object.Raw, marshaledWorkflow)
}

func (a *ArgoWorkflowHandler) process(manifest string) string {
	var buf bytes.Buffer
	m := map[string]interface{}{}

	core := make(map[string]string, 0)
	core["cores"] = "1"

	m["driver"] = core

	tmpl, _ := template.New("test").Parse(manifest)
	_ = tmpl.Execute(&buf, m)

	return buf.String()
}

// podAnnotator implements admission.DecoderInjector.
// A decoder will be automatically injected.

// InjectDecoder injects the decoder.
func (a *ArgoWorkflowHandler) InjectDecoder(d *admission.Decoder) error {
	a.decoder = d
	return nil
}
