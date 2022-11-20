k delete MutatingWebhookConfiguration mutating-webhook-configuration

kustomize build config/webhook/ | k apply -f -

