// Copyright (c) 2020 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package patch

import (
	"context"
	"fmt"
	"reflect"

	"github.com/pkg/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// OperationResult is the action result of a CreateOrPatch call.
type OperationResult string

const ( // They should complete the sentence "Deployment default/foo has been ..."
	// OperationResultNone means that the resource has not been changed
	OperationResultNone OperationResult = "unchanged"
	// OperationResultCreated means that a new resource is created
	OperationResultCreated OperationResult = "created"
	// OperationResultUpdated means that an existing resource is updated
	OperationResultUpdated OperationResult = "updated"
	// OperationResultUpdatedStatus means that an existing resource and its status is updated
	OperationResultUpdatedStatus OperationResult = "updatedStatus"
	// OperationResultUpdatedStatusOnly means that only an existing status is updated
	OperationResultUpdatedStatusOnly OperationResult = "updatedStatusOnly"
)

// CreateOrPatch creates or patches the given object in the Kubernetes
// cluster. The object's desired state must be reconciled with the before
// state inside the passed in callback MutateFn.
//
// The MutateFn is called regardless of creating or updating an object.
//
// It returns the executed operation and an error.
//
// nolint:gocognit
func CreateOrPatch(ctx context.Context, c client.Client, obj runtime.Object, f MutateFn) (OperationResult, error) {
	if obj == nil {
		return OperationResultNone, errors.Errorf("expected non-nil resource")
	}

	key, err := client.ObjectKeyFromObject(obj)
	if err != nil {
		return OperationResultNone, err
	}

	if err := c.Get(ctx, key, obj); err != nil {
		if !apierrors.IsNotFound(err) {
			return OperationResultNone, err
		}
		if f != nil {
			if err := mutate(f, key, obj); err != nil {
				return OperationResultNone, err
			}
		}
		if err := c.Create(ctx, obj); err != nil {
			return OperationResultNone, err
		}
		return OperationResultCreated, nil
	}

	return Patch(ctx, c, obj, f)
}

// Patch patches the given object in the Kubernetes cluster. The object's
// desired state must be reconciled with the before state inside the passed in
// callback MutateFn.
//
// It returns the executed operation and an error.
//
// nolint:gocognit
func Patch(ctx context.Context, c client.Client, obj runtime.Object, f MutateFn) (OperationResult, error) {
	if obj == nil {
		return OperationResultNone, errors.Errorf("expected non-nil resource")
	}

	key, err := client.ObjectKeyFromObject(obj)
	if err != nil {
		return OperationResultNone, err
	}

	// Create patches for the object and its possible status.
	objPatch := client.MergeFrom(obj.DeepCopyObject())
	statusPatch := client.MergeFrom(obj.DeepCopyObject())

	// Create a copy of the original object as well as converting that copy to
	// unstructured data.
	before, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj.DeepCopyObject())
	if err != nil {
		return OperationResultNone, err
	}

	// Attempt to extract the status from the resource for easier comparison later
	beforeStatus, hasBeforeStatus, err := unstructured.NestedFieldCopy(before, "status")
	if err != nil {
		return OperationResultNone, err
	}

	// If the resource contains a status then remove it from the unstructured
	// copy to avoid unnecessary patching later.
	if hasBeforeStatus {
		unstructured.RemoveNestedField(before, "status")
	}

	// Mutate the original object.
	if f != nil {
		if err := mutate(f, key, obj); err != nil {
			return OperationResultNone, err
		}
	}

	// Convert the resource to unstructured to compare against our before copy.
	after, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
	if err != nil {
		return OperationResultNone, err
	}

	// Attempt to extract the status from the resource for easier comparison later
	afterStatus, hasAfterStatus, err := unstructured.NestedFieldCopy(after, "status")
	if err != nil {
		return OperationResultNone, err
	}

	// If the resource contains a status then remove it from the unstructured
	// copy to avoid unnecessary patching later.
	if hasAfterStatus {
		unstructured.RemoveNestedField(after, "status")
	}

	result := OperationResultNone

	if !reflect.DeepEqual(before, after) {
		// Only issue a Patch if the before and after resources (minus status) differ
		if err := c.Patch(ctx, obj, objPatch); err != nil {
			return result, err
		}
		result = OperationResultUpdated
	}

	if (hasBeforeStatus || hasAfterStatus) && !reflect.DeepEqual(beforeStatus, afterStatus) {
		// Only issue a Status Patch if the resource has a status and the beforeStatus
		// and afterStatus copies differ
		if err := c.Status().Patch(ctx, obj, statusPatch); err != nil {
			return result, err
		}
		if result == OperationResultUpdated {
			result = OperationResultUpdatedStatus
		} else {
			result = OperationResultUpdatedStatusOnly
		}
	}

	return result, nil
}

// MutateFn is a function which mutates the existing object into it's desired state.
type MutateFn func() error

// mutate wraps a MutateFn and applies validation to its result
func mutate(f MutateFn, key client.ObjectKey, obj runtime.Object) error {
	if err := f(); err != nil {
		return err
	}
	if newKey, err := client.ObjectKeyFromObject(obj); err != nil || key != newKey {
		return fmt.Errorf("MutateFn cannot mutate object name and/or object namespace")
	}
	return nil
}
