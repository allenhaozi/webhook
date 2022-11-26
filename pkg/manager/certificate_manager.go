package manager

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	apitypes "k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/allenhaozi/alog"
	"github.com/allenhaozi/webhook/api/common"
	webhookutils "github.com/allenhaozi/webhook/pkg/utils"
)

type CertificateManager struct {
	CertDir string
	client.Client
	Log logr.Logger
}

func NewCertificateManager(c client.Client, l logr.Logger, certDir string) *CertificateManager {
	r := &CertificateManager{}
	r.CertDir = certDir
	r.Log = l
	r.Client = c

	return r
}

func (c *CertificateManager) GenerateCertificate(ns, name string) (*common.CertContext, error) {
	// if secret key exists, use the secret values
	// won't create new certificates

	ctx := context.Background()
	objectKey := apitypes.NamespacedName{
		Namespace: ns,
		Name:      name,
	}
	secret := &corev1.Secret{}
	err := c.Get(ctx, objectKey, secret)
	alog.TT(err)
	// if secret not found
	if kerrors.IsNotFound(err) {
		// trigger generate ca logic
		certContext, err := webhookutils.GenerateCert(objectKey.Namespace, common.WebHookName)
		if err != nil {
			c.Log.Error(err, "get namespace environment failure")
		}
		// write certificate file to local directory
		if err := certContext.WriteCertFileToLocal(c.CertDir); err != nil {
			c.Log.Error(err, "write certificate file to local failure")
		}
		// persist certificate to secret
		secret := certContext.ComposeSecrets(objectKey.Namespace, objectKey.Name)
		// TODO: use patch
		// if err := c.Create(ctx, secret); err != nil {
		if err := c.Patch(ctx, secret, client.Apply); err != nil {
			return nil, errors.Wrap(err, "create secret failure")
		}
		return certContext, nil
	}

	if err != nil {
		return nil, errors.Wrap(err, "generate certificate failure")
	}

	// certificate secret exists, use the values
	certContext, err := common.GenerateCertBySecret(secret)
	if err != nil {
		return nil, errors.Wrap(err, "parse secret to certificate failure")
	}

	// generate local certificate
	if err := certContext.WriteCertFileToLocal(c.CertDir); err != nil {
		return nil, errors.Wrap(err, "write certificate file to local failure")
	}

	return certContext, nil
}
