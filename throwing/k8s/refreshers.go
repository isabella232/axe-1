package k8s

import (
	"bytes"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/rancher/norman/types/convert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/apis/meta/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

type wrapper struct {
	group, version, name string
}

func (w wrapper) refreshResource(b *bytes.Buffer) error {
	restConfig, err := clientcmd.BuildConfigFromFlags("", os.Getenv("KUBECONFIG"))
	if err != nil {
		return err
	}
	clientset, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return err
	}

	restClient := clientset.RESTClient()
	apiPrefix := "apis"
	if w.group == "" {
		apiPrefix = "api"
	}
	if w.version == "" {
		w.version = "v1"
	}
	req := restClient.Get().Prefix(apiPrefix, w.group, w.version).Resource(w.name).Param("includeObject", "Object")
	header := "application/json;as=Table;g=meta.k8s.io;v=v1beta1, application/json"
	req.SetHeader("Accept", header)
	table := &v1beta1.Table{}
	if err := req.Do().Into(table); err != nil {
		return err
	}

	namespaced := true
	groupVersion := strings.Trim(fmt.Sprintf("%s/%s", w.group, w.version), "/")
	resourceList, err := clientset.ServerResourcesForGroupVersion(groupVersion)
	if err != nil {
		return err
	}
	for _, r := range resourceList.APIResources {
		if r.Name == w.name {
			namespaced = r.Namespaced
		}
	}

	// insert namespace
	if namespaced {
		table.ColumnDefinitions = append([]v1beta1.TableColumnDefinition{
			{
				Name: "NAMESPACE",
			},
		}, table.ColumnDefinitions...)
	}

	for i, header := range table.ColumnDefinitions {
		b.Write([]byte(strings.ToUpper(header.Name)))
		if i == len(table.ColumnDefinitions)-1 {
			b.Write([]byte("\n"))
		} else {
			b.Write([]byte("\t"))
		}
	}

	for _, row := range table.Rows {
		converted, err := runtime.Decode(unstructured.UnstructuredJSONScheme, row.Object.Raw)
		if err != nil {
			return err
		}
		row.Object.Object = converted
		namespace := ""
		object, ok := row.Object.Object.(metav1.Object)
		if ok {
			namespace = object.GetNamespace()
		}
		if namespaced {
			row.Cells = append([]interface{}{namespace}, row.Cells...)
		}
		for i, column := range row.Cells {
			b.Write([]byte(convert.ToString(column)))
			if i == len(row.Cells)-1 {
				b.Write([]byte("\n"))
			} else {
				b.Write([]byte("\t"))
			}
		}
	}
	return nil
}

func RefreshResourceKind(b *bytes.Buffer) error {
	restConfig, err := clientcmd.BuildConfigFromFlags("", os.Getenv("KUBECONFIG"))
	if err != nil {
		return err
	}
	clientset := kubernetes.NewForConfigOrDie(restConfig)

	Header := []string{
		"NAME",
		"GROUPVERSION",
	}
	list, err := clientset.Discovery().ServerPreferredResources()
	if err != nil {
		return err
	}
	for i, header := range Header {
		b.Write([]byte(header))
		if i == len(Header)-1 {
			b.Write([]byte("\n"))
		} else {
			b.Write([]byte("\t"))
		}
	}
	var resources []struct {
		Name         string
		GroupVersion string
	}


	for _, l := range list {
		for _, r := range l.APIResources {
			resources = append(resources, struct {
				Name         string
				GroupVersion string
			}{Name: r.Name, GroupVersion: l.GroupVersion})
		}
	}

	sort.Slice(resources, func(i, j int) bool {
		return resources[i].Name < resources[j].Name
	})

	for _, r := range resources {
		b.Write([]byte(r.Name))
		b.Write([]byte("\t"))
		b.Write([]byte(r.GroupVersion))
		b.Write([]byte("\n"))
	}
	return nil
}
