v=$1
export IMG=allenhaozi/webhook.tar:v0.0.${v}
k delete MutatingWebhookConfiguration mutating-webhook-configuration
make deploy
