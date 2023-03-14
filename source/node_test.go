/*
Copyright 2019 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package source

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes/fake"

	"sigs.k8s.io/external-dns/endpoint"
)

func TestNodeSource(t *testing.T) {
	t.Parallel()

	t.Run("NewNodeSource", testNodeSourceNewNodeSource)
	t.Run("Endpoints", testNodeSourceEndpoints)
}

// testNodeSourceNewNodeSource tests that NewNodeService doesn't return an error.
func testNodeSourceNewNodeSource(t *testing.T) {
	t.Parallel()

	for _, ti := range []struct {
		title            string
		annotationFilter string
		fqdnTemplate     string
		labelSelector    labels.Selector
		expectError      bool
	}{
		{
			title:         "invalid template",
			expectError:   true,
			fqdnTemplate:  "{{.Name",
			labelSelector: labels.Everything(),
		},
		{
			title:         "valid empty template",
			expectError:   false,
			labelSelector: labels.Everything(),
		},
		{
			title:         "valid template",
			expectError:   false,
			fqdnTemplate:  "{{.Name}}-{{.Namespace}}.ext-dns.test.com",
			labelSelector: labels.Everything(),
		},
		{
			title:            "non-empty annotation filter label",
			expectError:      false,
			annotationFilter: "kubernetes.io/ingress.class=nginx",
			labelSelector:    labels.Everything(),
		},
	} {
		ti := ti
		t.Run(ti.title, func(t *testing.T) {
			t.Parallel()

			_, err := NewNodeSource(
				context.TODO(),
				fake.NewSimpleClientset(),
				ti.annotationFilter,
				ti.fqdnTemplate,
				ti.labelSelector,
			)

			if ti.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// testNodeSourceEndpoints tests that various node generate the correct endpoints.
func testNodeSourceEndpoints(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		title            string
		annotationFilter string
		fqdnTemplate     string
		labelSelector    labels.Selector
		nodeName         string
		nodeAddresses    []v1.NodeAddress
		labels           map[string]string
		annotations      map[string]string
		expected         []*endpoint.Endpoint
		expectError      bool
	}{
		{
			"node with short hostname returns one endpoint",
			"",
			"",
			labels.Everything(),
			"node1",
			[]v1.NodeAddress{{Type: v1.NodeExternalIP, Address: "1.2.3.4"}},
			map[string]string{},
			map[string]string{},
			[]*endpoint.Endpoint{
				{RecordType: "A", DNSName: "node1", Targets: endpoint.Targets{"1.2.3.4"}},
			},
			false,
		},
		{
			"node with fqdn returns one endpoint",
			"",
			"",
			labels.Everything(),
			"node1.example.org",
			[]v1.NodeAddress{{Type: v1.NodeExternalIP, Address: "1.2.3.4"}},
			map[string]string{},
			map[string]string{},
			[]*endpoint.Endpoint{
				{RecordType: "A", DNSName: "node1.example.org", Targets: endpoint.Targets{"1.2.3.4"}},
			},
			false,
		},
		{
			"node with fqdn template returns endpoint with expanded hostname",
			"",
			"{{.Name}}.example.org",
			labels.Everything(),
			"node1",
			[]v1.NodeAddress{{Type: v1.NodeExternalIP, Address: "1.2.3.4"}},
			map[string]string{},
			map[string]string{},
			[]*endpoint.Endpoint{
				{RecordType: "A", DNSName: "node1.example.org", Targets: endpoint.Targets{"1.2.3.4"}},
			},
			false,
		},
		{
			"node with fqdn and fqdn template returns one endpoint",
			"",
			"{{.Name}}.example.org",
			labels.Everything(),
			"node1.example.org",
			[]v1.NodeAddress{{Type: v1.NodeExternalIP, Address: "1.2.3.4"}},
			map[string]string{},
			map[string]string{},
			[]*endpoint.Endpoint{
				{RecordType: "A", DNSName: "node1.example.org.example.org", Targets: endpoint.Targets{"1.2.3.4"}},
			},
			false,
		},
		{
			"node with fqdn template returns two endpoints with multiple IP addresses and expanded hostname",
			"",
			"{{.Name}}.example.org",
			labels.Everything(),
			"node1",
			[]v1.NodeAddress{{Type: v1.NodeExternalIP, Address: "1.2.3.4"}, {Type: v1.NodeExternalIP, Address: "5.6.7.8"}},
			map[string]string{},
			map[string]string{},
			[]*endpoint.Endpoint{
				{RecordType: "A", DNSName: "node1.example.org", Targets: endpoint.Targets{"1.2.3.4", "5.6.7.8"}},
			},
			false,
		},
		{
			"node with both external and internal IP returns an endpoint with external IP",
			"",
			"",
			labels.Everything(),
			"node1",
			[]v1.NodeAddress{{Type: v1.NodeExternalIP, Address: "1.2.3.4"}, {Type: v1.NodeInternalIP, Address: "2.3.4.5"}},
			map[string]string{},
			map[string]string{},
			[]*endpoint.Endpoint{
				{RecordType: "A", DNSName: "node1", Targets: endpoint.Targets{"1.2.3.4"}},
			},
			false,
		},
		{
			"node with only internal IP returns an endpoint with internal IP",
			"",
			"",
			labels.Everything(),
			"node1",
			[]v1.NodeAddress{{Type: v1.NodeInternalIP, Address: "2.3.4.5"}},
			map[string]string{},
			map[string]string{},
			[]*endpoint.Endpoint{
				{RecordType: "A", DNSName: "node1", Targets: endpoint.Targets{"2.3.4.5"}},
			},
			false,
		},
		{
			"node with neither external nor internal IP returns no endpoints",
			"",
			"",
			labels.Everything(),
			"node1",
			[]v1.NodeAddress{},
			map[string]string{},
			map[string]string{},
			[]*endpoint.Endpoint{},
			true,
		},
		{
			"annotated node without annotation filter returns endpoint",
			"",
			"",
			labels.Everything(),
			"node1",
			[]v1.NodeAddress{{Type: v1.NodeExternalIP, Address: "1.2.3.4"}},
			map[string]string{},
			map[string]string{
				"service.beta.kubernetes.io/external-traffic": "OnlyLocal",
			},
			[]*endpoint.Endpoint{
				{RecordType: "A", DNSName: "node1", Targets: endpoint.Targets{"1.2.3.4"}},
			},
			false,
		},
		{
			"annotated node with matching annotation filter returns endpoint",
			"service.beta.kubernetes.io/external-traffic in (Global, OnlyLocal)",
			"",
			labels.Everything(),
			"node1",
			[]v1.NodeAddress{{Type: v1.NodeExternalIP, Address: "1.2.3.4"}},
			map[string]string{},
			map[string]string{
				"service.beta.kubernetes.io/external-traffic": "OnlyLocal",
			},
			[]*endpoint.Endpoint{
				{RecordType: "A", DNSName: "node1", Targets: endpoint.Targets{"1.2.3.4"}},
			},
			false,
		},
		{
			"annotated node with non-matching annotation filter returns endpoint",
			"service.beta.kubernetes.io/external-traffic in (Global, OnlyLocal)",
			"",
			labels.Everything(),
			"node1",
			[]v1.NodeAddress{{Type: v1.NodeExternalIP, Address: "1.2.3.4"}},
			map[string]string{},
			map[string]string{
				"service.beta.kubernetes.io/external-traffic": "SomethingElse",
			},
			[]*endpoint.Endpoint{},
			false,
		},
		{
			"our controller type is dns-controller",
			"",
			"",
			labels.Everything(),
			"node1",
			[]v1.NodeAddress{{Type: v1.NodeExternalIP, Address: "1.2.3.4"}},
			map[string]string{},
			map[string]string{
				controllerAnnotationKey: controllerAnnotationValue,
			},
			[]*endpoint.Endpoint{
				{RecordType: "A", DNSName: "node1", Targets: endpoint.Targets{"1.2.3.4"}},
			},
			false,
		},
		{
			"different controller types are ignored",
			"",
			"",
			labels.Everything(),
			"node1",
			[]v1.NodeAddress{{Type: v1.NodeExternalIP, Address: "1.2.3.4"}},
			map[string]string{},
			map[string]string{
				controllerAnnotationKey: "not-dns-controller",
			},
			[]*endpoint.Endpoint{},
			false,
		},
		{
			"ttl not annotated should have RecordTTL.IsConfigured set to false",
			"",
			"",
			labels.Everything(),
			"node1",
			[]v1.NodeAddress{{Type: v1.NodeExternalIP, Address: "1.2.3.4"}},
			map[string]string{},
			map[string]string{},
			[]*endpoint.Endpoint{
				{RecordType: "A", DNSName: "node1", Targets: endpoint.Targets{"1.2.3.4"}, RecordTTL: endpoint.TTL(0)},
			},
			false,
		},
		{
			"ttl annotated but invalid should have RecordTTL.IsConfigured set to false",
			"",
			"",
			labels.Everything(),
			"node1",
			[]v1.NodeAddress{{Type: v1.NodeExternalIP, Address: "1.2.3.4"}},
			map[string]string{},
			map[string]string{
				ttlAnnotationKey: "foo",
			},
			[]*endpoint.Endpoint{
				{RecordType: "A", DNSName: "node1", Targets: endpoint.Targets{"1.2.3.4"}, RecordTTL: endpoint.TTL(0)},
			},
			false,
		},
		{
			"ttl annotated and is valid should set Record.TTL",
			"",
			"",
			labels.Everything(),
			"node1",
			[]v1.NodeAddress{{Type: v1.NodeExternalIP, Address: "1.2.3.4"}},
			map[string]string{},
			map[string]string{
				ttlAnnotationKey: "10",
			},
			[]*endpoint.Endpoint{
				{RecordType: "A", DNSName: "node1", Targets: endpoint.Targets{"1.2.3.4"}, RecordTTL: endpoint.TTL(10)},
			},
			false,
		},
		{
			"node with nil Lables returns valid endpoint",
			"",
			"",
			labels.Everything(),
			"node1",
			[]v1.NodeAddress{{Type: v1.NodeExternalIP, Address: "1.2.3.4"}},
			nil,
			map[string]string{},
			[]*endpoint.Endpoint{
				{RecordType: "A", DNSName: "node1", Targets: endpoint.Targets{"1.2.3.4"}, Labels: map[string]string{}},
			},
			false,
		},
		{
			"node with labels that match label filter returns endpoint",
			"",
			"",
			labels.SelectorFromSet(labels.Set{"node-label": "include"}),
			"node1",
			[]v1.NodeAddress{{Type: v1.NodeExternalIP, Address: "1.2.3.4"}},
			map[string]string{
				"node-label": "include",
			},
			map[string]string{},
			[]*endpoint.Endpoint{
				{RecordType: "A", DNSName: "node1", Targets: endpoint.Targets{"1.2.3.4"}},
			},
			false,
		},
		{
			"node with labels that don't match label filter is ignored",
			"",
			"",
			labels.SelectorFromSet(labels.Set{"node-label": "include"}),
			"node1",
			[]v1.NodeAddress{{Type: v1.NodeExternalIP, Address: "1.2.3.4"}},
			map[string]string{
				"node-label": "exclude",
			},
			map[string]string{},
			[]*endpoint.Endpoint{},
			false,
		},
	} {
		tc := tc
		t.Run(tc.title, func(t *testing.T) {
			t.Parallel()

			// Create a Kubernetes testing client
			kubernetes := fake.NewSimpleClientset()

			node := &v1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name:        tc.nodeName,
					Labels:      tc.labels,
					Annotations: tc.annotations,
				},
				Status: v1.NodeStatus{
					Addresses: tc.nodeAddresses,
				},
			}

			_, err := kubernetes.CoreV1().Nodes().Create(context.Background(), node, metav1.CreateOptions{})
			require.NoError(t, err)

			// Create our object under test and get the endpoints.
			client, err := NewNodeSource(
				context.TODO(),
				kubernetes,
				tc.annotationFilter,
				tc.fqdnTemplate,
				tc.labelSelector,
			)
			require.NoError(t, err)

			endpoints, err := client.Endpoints(context.Background())
			if tc.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			// Validate returned endpoints against desired endpoints.
			validateEndpoints(t, endpoints, tc.expected)
		})
	}
}
