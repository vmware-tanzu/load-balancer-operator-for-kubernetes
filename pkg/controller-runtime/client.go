// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package controllerruntime

import (
	"context"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
)

// This file is ported from
// https://github.com/kubernetes-sigs/controller-runtime/pull/1249/files then
// adapted.
// We cannot directly consume it from upstream because we're currently on CAPI
// v0.3.x, which is on controller-runtime v0.5.x
// TODO(fangyuanl): update vendor to directly consume it from upstream once
// controller-runtime could be updated to v0.7.x

// ClientBuilder builder is the interface for the client builder.
type ClientBuilder interface {
	// WithUncached takes a list of runtime objects (plain or lists) that users don't want to cache
	// for this client. This function can be called multiple times, it should append to an internal slice.
	WithUncached(objs ...runtime.Object) ClientBuilder

	// Build returns a new client.
	Build(cache cache.Cache, config *rest.Config, options client.Options) (client.Client, error)
}

// NewClientBuilder returns a builder to build new clients to be passed when creating a Manager.
func NewClientBuilder() ClientBuilder {
	return &newClientBuilder{}
}

type newClientBuilder struct {
	uncached []runtime.Object
}

func (n *newClientBuilder) WithUncached(objs ...runtime.Object) ClientBuilder {
	n.uncached = append(n.uncached, objs...)
	return n
}

func (n *newClientBuilder) Build(cache cache.Cache, config *rest.Config, options client.Options) (client.Client, error) {
	// Create the Client for Write operations.
	c, err := client.New(config, options)
	if err != nil {
		return nil, err
	}

	clientScheme := scheme.Scheme
	if options.Scheme != nil {
		clientScheme = options.Scheme
	}

	return NewDelegatingClient(newDelegatingClientInput{
		CacheReader:     cache,
		Client:          c,
		Scheme:          clientScheme,
		UncachedObjects: n.uncached,
	})
}

// newDelegatingClientInput encapsulates the input parameters to create a new delegating client.
type newDelegatingClientInput struct {
	CacheReader     client.Reader
	Client          client.Client
	Scheme          *runtime.Scheme
	UncachedObjects []runtime.Object
}

// NewDelegatingClient creates a new delegating client.
//
// A delegating client forms a Client by composing separate reader, writer and
// statusclient interfaces.  This way, you can have an Client that reads from a
// cache and writes to the API server.
func NewDelegatingClient(in newDelegatingClientInput) (client.Client, error) {
	uncachedGVKs := map[schema.GroupVersionKind]struct{}{}
	for _, obj := range in.UncachedObjects {
		gvk, err := apiutil.GVKForObject(obj, in.Scheme)
		if err != nil {
			return nil, err
		}
		uncachedGVKs[gvk] = struct{}{}
	}

	return &delegatingClient{
		Reader: &delegatingReader{
			CacheReader:  in.CacheReader,
			ClientReader: in.Client,
			scheme:       in.Scheme,
			uncachedGVKs: uncachedGVKs,
		},
		Writer:       in.Client,
		StatusClient: in.Client,
	}, nil
}

// delegatingClient forms a Client by composing separate reader, writer and
// statusclient interfaces.  This way, you can have an Client that reads from a
// cache and writes to the API server.
type delegatingClient struct {
	client.Reader
	client.Writer
	client.StatusClient
}

// delegatingReader forms a Reader that will cause Get and List requests for
// unstructured types to use the ClientReader while requests for any other type
// of object with use the CacheReader.  This avoids accidentally caching the
// entire cluster in the common case of loading arbitrary unstructured objects
// (e.g. from OwnerReferences).
type delegatingReader struct {
	CacheReader  client.Reader
	ClientReader client.Reader

	uncachedGVKs map[schema.GroupVersionKind]struct{}
	scheme       *runtime.Scheme
}

func (d *delegatingReader) shouldBypassCache(obj runtime.Object) (bool, error) {
	gvk, err := apiutil.GVKForObject(obj, d.scheme)
	if err != nil {
		return false, err
	}
	_, isUncached := d.uncachedGVKs[gvk]
	_, isUnstructured := obj.(*unstructured.Unstructured)
	_, isUnstructuredList := obj.(*unstructured.UnstructuredList)
	return isUncached || isUnstructured || isUnstructuredList, nil
}

// Get retrieves an obj for a given object key from the Kubernetes Cluster.
func (d *delegatingReader) Get(ctx context.Context, key client.ObjectKey, obj runtime.Object) error {
	if isUncached, err := d.shouldBypassCache(obj); err != nil {
		return err
	} else if isUncached {
		return d.ClientReader.Get(ctx, key, obj)
	}
	return d.CacheReader.Get(ctx, key, obj)
}

// List retrieves list of objects for a given namespace and list options.
func (d *delegatingReader) List(ctx context.Context, list runtime.Object, opts ...client.ListOption) error {
	if isUncached, err := d.shouldBypassCache(list); err != nil {
		return err
	} else if isUncached {
		return d.ClientReader.List(ctx, list, opts...)
	}
	return d.CacheReader.List(ctx, list, opts...)
}
