package jobrunner

import (
	"testing"

	"k8s.io/client-go/kubernetes"

	"github.com/dollarshaveclub/furan/pkg/models"
)

func TestK8sJobRunner_Run(t *testing.T) {
	type fields struct {
		client    *kubernetes.Clientset
		imageInfo ImageInfo
		JobFunc   JobFactoryFunc
	}
	type args struct {
		build models.Build
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kr := K8sJobRunner{
				client:    tt.fields.client,
				imageInfo: tt.fields.imageInfo,
				JobFunc:   tt.fields.JobFunc,
			}
			if err := kr.Run(tt.args.build); (err != nil) != tt.wantErr {
				t.Errorf("Run() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
