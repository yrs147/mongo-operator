/*
Copyright 2023 YashRaj.

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

package controller

import (
	"context"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	databasesv1alpha1 "github.com/yrs147/mongo-operator/api/v1alpha1"
)

// MongoDBReconciler reconciles a MongoDB object
type MongoDBReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=databases.core.yrs.io,resources=mongodbs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=databases.core.yrs.io,resources=mongodbs/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=databases.core.yrs.io,resources=mongodbs/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the MongoDB object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.15.0/pkg/reconcile
func (r *MongoDBReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

	//Fetching MongoDB instance
	mongo := &databasesv1alpha1.MongoDB{}
	if err := r.Get(ctx, req.NamespacedName, mongo); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	//Generating Serivce object managed by MongoDB

	//Service struct to create or update
	service := &corev1.Service{
		ObjectMeta: ctrl.ObjectMeta{
			Name:      req.Name + "-mongodb-service",
			Namespace: req.Namespace,
		},
	}

	//Create or update stateful set
	_, err := ctrl.CreateOrUpdate(ctx, r.Client, service, func() error {

		util.SetServiceFields(service, mongo)

		//Adding owner refernece for garbage collection
		return controllerutil.SetControllerReference(mongo, service, r.Scheme)

	})
	if err != nil {
		return ctrl.Result{}, err
	}

	//Generating a Stateful managed by MongoDB
	
	stateful := &appsv1.StatefulSet{
		ObjectMeta: ctrl.ObjectMeta{
			Name: req.Name+"-mongodb-stateful",
			Namespace: req.Namespace,
		},
	}

	//Create or Delete StatefulSet
	_,err = ctrl.CreateOrUpdate(ctx, r.Client, service, func() error{
	util.SetStatefulSetFields(stateful, service, mongo, mongo.Spec.Replicas, mongo.Spec.Storage)

	//Adding owner reference for garbage collection
	return controllerutil.SetControllerReference(mongo, stateful, r.Scheme)
	})
	if err !=nil {
		return ctrl.Result{},err
	}

	//Updating MongoDB Status

	mongo.Status.StatefulSetStatus = stateful.Status
	mongo.Status.ServiceStatus = service.Status
	mongo.Status.ClusterIP = service.Spec.ClusterIP

	err = r.Status().Update(ctx,mongo)
	
	if err !=nil{
		return ctrl.Result{},err
	}

}

// SetupWithManager sets up the controller with the Manager.
func (r *MongoDBReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&databasesv1alpha1.MongoDB{}).
		Owns(&appsv1.StatefulSet{}).
		Owns(&corev1.Service{}).
		Complete(r)
}
