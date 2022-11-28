package main

import "github.com/allenhaozi/webhook/pkg/helm"

func main() {
	chartPath := "/Users/mahao/go/src/github.com/allenhaozi/webhook/charts/salesforecast"
	valueFile := "/Users/mahao/go/src/github.com/allenhaozi/webhook/charts/salesforecast/values.yaml"
	helm.Template(chartPath, valueFile)
}
