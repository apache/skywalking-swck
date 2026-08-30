// Licensed to Apache Software Foundation (ASF) under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Apache Software Foundation (ASF) licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package v1alpha1

import (
	"context"
	"fmt"

	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// UIKindHorizon is the only supported UI. Exported so the CRD enum, the defaulter, the validator
// and the controller that has to skip anything else cannot drift apart.
const (
	UIKindHorizon = "horizon"

	// Where Horizon reads dashboard templates from. live goes to OAP's store over the admin host;
	// readonly renders the bundle inside the image.
	uiTemplatesModeLive     = "live"
	uiTemplatesModeReadonly = "readonly"
)

// log is for logging in this package.
var uilog = logf.Log.WithName("ui-resource")

func (r *UI) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr, r).
		WithDefaulter(r).
		WithValidator(r).
		Complete()
}

// nolint: lll
// +kubebuilder:webhook:admissionReviewVersions=v1,sideEffects=None,path=/mutate-operator-skywalking-apache-org-v1alpha1-ui,mutating=true,failurePolicy=fail,groups=operator.skywalking.apache.org,resources=uis,verbs=create;update,versions=v1alpha1,name=mui.kb.io

// Default implements webhook.CustomDefaulter so a webhook will be registered for the type
func (r *UI) Default(_ context.Context, ui *UI) error {
	uilog.Info("default", "name", ui.Name)

	if ui.Spec.Kind == "" {
		ui.Spec.Kind = UIKindHorizon
	}

	if ui.Spec.Image == "" {
		// Horizon releases share the skywalking-ui repository with the retired Booster UI and are
		// told apart by a horizon- prefix on the tag. There is no apache/skywalking-horizon-ui
		// repository on Docker Hub, so naming one produced an image that could never be pulled.
		ui.Spec.Image = fmt.Sprintf("apache/skywalking-ui:horizon-%s", ui.Spec.Version)
	}

	// live template mode reads and seeds through OAP 11's /ui-management/templates* REST API, which
	// Horizon reaches on oap.adminUrl -- not the query host. The admin address is deliberately not
	// derived (see below), so defaulting the mode to live would leave a UI probing 127.0.0.1:17128,
	// failing the ui-management preflight, and blocking every layer-driven page. readonly renders
	// the templates bundled in the image and never calls that API, which is the only mode that can
	// work without an admin host.
	if ui.Spec.TemplatesMode == "" {
		if ui.Spec.OAPServerAdminAddress != "" {
			ui.Spec.TemplatesMode = uiTemplatesModeLive
		} else {
			ui.Spec.TemplatesMode = uiTemplatesModeReadonly
		}
	}

	ui.Spec.Service.Template.Default()
	if ui.Spec.OAPServerAddress == "" {
		ui.Spec.OAPServerAddress = fmt.Sprintf("http://%s-oap.%s:12800", ui.Name, ui.Namespace)
	}
	// OAPServerAdminAddress and OAPServerZipkinAddress are deliberately NOT defaulted.
	//
	// The OAP admin host arrived in 11.x. On 10.x, which this operator still supports, port 17128
	// is the AI-pipeline URI-recognition server -- so deriving an admin URL from the OAP address
	// would hand Horizon a live endpoint that is the wrong service. Unset, Horizon falls back to
	// localhost and reports the admin API unreachable, which is a failure an operator can read.
	//
	// Zipkin is the same shape of guess: the OAPServer this operator deploys exposes no Zipkin
	// query port, so <queryUrl>/zipkin would be a URL nothing serves.

	return nil
}

// nolint: lll
// +kubebuilder:webhook:admissionReviewVersions=v1,sideEffects=None,verbs=create;update,path=/validate-operator-skywalking-apache-org-v1alpha1-ui,mutating=false,failurePolicy=fail,groups=operator.skywalking.apache.org,resources=uis,versions=v1alpha1,name=vui.kb.io

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type
func (r *UI) ValidateCreate(_ context.Context, ui *UI) (admission.Warnings, error) {
	uilog.Info("validate create", "name", ui.Name)
	return nil, ui.validate()
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type
func (r *UI) ValidateUpdate(_ context.Context, ui *UI, _ *UI) (admission.Warnings, error) {
	uilog.Info("validate update", "name", ui.Name)
	return nil, ui.validate()
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type
func (r *UI) ValidateDelete(_ context.Context, ui *UI) (admission.Warnings, error) {
	uilog.Info("validate delete", "name", ui.Name)
	return nil, nil
}

func (r *UI) validate() error {
	if r.Spec.Kind != "" && r.Spec.Kind != UIKindHorizon {
		return fmt.Errorf("unsupported ui kind %q: only %q is supported. The Booster UI was "+
			"removed from apache/skywalking in 11.0.0 and no image is built for it; move to "+
			"kind: horizon with a Horizon version, for example 1.0.0", r.Spec.Kind, UIKindHorizon)
	}
	if r.Spec.Image == "" {
		return fmt.Errorf("image is absent")
	}
	if err := r.Spec.Service.Template.Validate(); err != nil {
		return fmt.Errorf("service template is invalid: %w", err)
	}
	if r.Spec.OAPServerAddress == "" {
		return fmt.Errorf("oap server address is absent")
	}
	return nil
}
